package handler

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/handler/dto"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
)

var liteSetupToken string

// SetLiteSetupToken is called by the native desktop host before serving requests.
// Web deployments leave it empty and cannot issue anonymous administrator tokens.
func SetLiteSetupToken(token string) { liteSetupToken = token }

// AuthHandler implements HTTP request handlers for user authentication
// Provides functionality for user registration, login, logout, and token management
// through the REST API endpoints
type AuthHandler struct {
	userService      interfaces.UserService
	tenantService    interfaces.TenantService
	configInfo       *config.Config
	systemSettingSvc interfaces.SystemSettingService
}

// NewAuthHandler creates a new auth handler instance with the provided services
// Parameters:
//   - userService: An implementation of the UserService interface for business logic
//   - tenantService: An implementation of the TenantService interface for tenant management
//   - systemSettingSvc: 3-tier resolver for runtime-tunable settings such as
//     the password-complexity switch.
//
// Returns a pointer to the newly created AuthHandler
func NewAuthHandler(configInfo *config.Config,
	userService interfaces.UserService, tenantService interfaces.TenantService,
	systemSettingSvc interfaces.SystemSettingService,
) *AuthHandler {
	return &AuthHandler{
		configInfo:       configInfo,
		userService:      userService,
		tenantService:    tenantService,
		systemSettingSvc: systemSettingSvc,
	}
}

// resolveDefaultTenantMode returns the provisioning policy for a new
// local user account.
// Priority: DB system_settings > cfg.Auth > hard default (create_personal).
// Shared by public registration and the SystemAdmin create-user endpoint.
func resolveDefaultTenantMode(
	ctx context.Context,
	configInfo *config.Config,
	systemSettingSvc interfaces.SystemSettingService,
) types.TenantProvisioningMode {
	def := config.AuthDefaultTenantModeCreatePersonal
	if configInfo != nil && configInfo.Auth != nil {
		if mode := strings.TrimSpace(configInfo.Auth.DefaultTenantMode); mode != "" {
			def = mode
		}
	}
	mode := def
	if systemSettingSvc != nil {
		mode = systemSettingSvc.GetString(
			ctx,
			"auth.default_tenant_mode",
			"WEKNORA_AUTH_DEFAULT_TENANT_MODE",
			def,
		)
	}
	if mode == config.AuthDefaultTenantModeTenantless {
		return types.TenantProvisioningTenantless
	}
	return types.TenantProvisioningCreatePersonal
}

// resolveDefaultTenantMode returns the provisioning policy for ordinary
// public password registrations.
func (h *AuthHandler) resolveDefaultTenantMode(ctx context.Context) types.TenantProvisioningMode {
	return resolveDefaultTenantMode(ctx, h.configInfo, h.systemSettingSvc)
}

// resolveDefaultTenantMode resolves the same policy for users provisioned
// by a SystemAdmin via POST /api/v1/system/admin/users/create.
func (h *SystemHandler) resolveDefaultTenantMode(ctx context.Context) types.TenantProvisioningMode {
	return resolveDefaultTenantMode(ctx, h.cfg, h.systemSettingSvc)
}

func (h *AuthHandler) complexPasswordEnabled(ctx context.Context) bool {
	return service.ResolveComplexPasswordEnabled(ctx, h.configInfo, h.systemSettingSvc)
}

func (h *SystemHandler) complexPasswordEnabled(ctx context.Context) bool {
	return service.ResolveComplexPasswordEnabled(ctx, h.cfg, h.systemSettingSvc)
}

// Register godoc
// @Summary      用户注册
// @Description  注册新用户账号
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      types.RegisterRequest  true  "注册请求参数"
// @Success      201      {object}  types.RegisterResponse
// @Failure      400      {object}  errors.AppError  "请求参数错误"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start user registration")

	var req types.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse registration request parameters", err)
		appErr := errors.NewValidationError("Invalid registration parameters").WithDetails(err.Error())
		c.Error(appErr)
		return
	}
	req.Username = secutils.SanitizeForLog(req.Username)
	req.Email = secutils.SanitizeForLog(req.Email)
	// Password is intentionally NOT sanitized: SanitizeForLog replaces
	// \n, \r, \t and strips other control characters so a string is safe
	// to write into a log line. Applying it to a real password would
	// silently rewrite the credential before hashing, so registration
	// would succeed but login with the original password would fail.
	// Passwords must never be logged, so they don't need that defence.

	// Validate required fields
	if req.Username == "" || req.Email == "" || req.Password == "" {
		logger.Error(ctx, "Missing required registration fields")
		appErr := errors.NewValidationError("Username, email and password are required")
		c.Error(appErr)
		return
	}

	// Validate password against the runtime policy (DB system_settings
	// first) before creating the account.
	if err := service.ValidatePasswordPolicy(req.Password, h.complexPasswordEnabled(ctx)); err != nil {
		logger.Error(ctx, "Invalid password policy")
		appErr := errors.NewValidationError(err.Error())
		_ = c.Error(appErr)
		return
	}

	req.Username = secutils.SanitizeForLog(req.Username)
	req.Email = secutils.SanitizeForLog(req.Email)
	req.TenantProvisioning = h.resolveDefaultTenantMode(ctx)
	// Call service to register user
	user, err := h.userService.Register(ctx, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to register user: %v", err)
		appErr := errors.NewBadRequestError(err.Error())
		c.Error(appErr)
		return
	}

	// Return success response
	response := &types.RegisterResponse{
		Success: true,
		Message: "Registration successful",
		User:    user,
	}

	logger.Infof(ctx, "User registered successfully: %s", secutils.SanitizeForLog(user.Email))
	c.JSON(http.StatusCreated, response)
}

// Login godoc
// @Summary      用户登录
// @Description  用户登录并获取访问令牌
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      types.LoginRequest  true  "登录请求参数"
// @Success      200      {object}  types.LoginResponse
// @Failure      401      {object}  errors.AppError  "认证失败"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start user login")

	var req types.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse login request parameters", err)
		appErr := errors.NewValidationError("Invalid login parameters").WithDetails(err.Error())
		c.Error(appErr)
		return
	}
	email := secutils.SanitizeForLog(req.Email)

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		logger.Error(ctx, "Missing required login fields")
		appErr := errors.NewValidationError("Email and password are required")
		c.Error(appErr)
		return
	}

	// Call service to authenticate user
	response, err := h.userService.Login(ctx, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to login user: %v", err)
		appErr := errors.NewUnauthorizedError("Login failed").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	// Check if login was successful
	if !response.Success {
		logger.Warnf(ctx, "Login failed: %s", response.Message)
		c.JSON(http.StatusUnauthorized, dto.NewAuthLoginResponse(response))
		return
	}

	// User is already in the correct format from service

	logger.Infof(ctx, "User logged in successfully, email: %s", email)
	c.JSON(http.StatusOK, dto.NewAuthLoginResponse(response))
}

// Logout godoc
// @Summary      用户登出
// @Description  撤销当前访问令牌并登出
// @Tags         认证
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "登出成功"
// @Failure      400  {object}  errors.AppError         "请求参数错误"
// @Security     Bearer
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start user logout")

	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		logger.Error(ctx, "Missing Authorization header")
		appErr := errors.NewValidationError("Authorization header is required")
		c.Error(appErr)
		return
	}

	// Parse Bearer token
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		logger.Error(ctx, "Invalid Authorization header format")
		appErr := errors.NewValidationError("Invalid Authorization header format")
		c.Error(appErr)
		return
	}

	token := tokenParts[1]

	// Revoke every outstanding session for this user so refresh tokens
	// cannot keep working after logout.
	err := h.userService.Logout(ctx, token)
	if err != nil {
		logger.Errorf(ctx, "Failed to revoke token: %v", err)
		appErr := errors.NewInternalServerError("Logout failed").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	logger.Info(ctx, "User logged out successfully")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logout successful",
	})
}

// RefreshToken godoc
// @Summary      刷新令牌
// @Description  使用刷新令牌获取新的访问令牌
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      object{refreshToken=string}  true  "刷新令牌"
// @Success      200      {object}  map[string]interface{}       "新令牌"
// @Failure      401      {object}  errors.AppError              "令牌无效"
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start token refresh")

	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse refresh token request", err)
		appErr := errors.NewValidationError("Invalid refresh token request").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	// Call service to refresh token
	accessToken, newRefreshToken, err := h.userService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		logger.Errorf(ctx, "Failed to refresh token: %v", err)
		appErr := errors.NewUnauthorizedError("Token refresh failed").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	logger.Info(ctx, "Token refreshed successfully")
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Token refreshed successfully",
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
	})
}

// GetCurrentUser godoc
// @Summary      获取当前用户信息
// @Description  获取当前登录用户的详细信息
// @Tags         认证
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "用户信息"
// @Failure      401  {object}  errors.AppError         "未授权"
// @Security     Bearer
// @Router       /auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	ctx := c.Request.Context()

	// Get current user from service (which extracts from context)
	user, err := h.userService.GetCurrentUser(ctx)
	if err != nil {
		logger.Errorf(ctx, "Failed to get current user: %v", err)
		appErr := errors.NewUnauthorizedError("Failed to get user information").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	// Get tenant information for the *active* tenant (the one the auth
	// middleware resolved from the JWT claim), not the user's home
	// tenant. user.TenantID is the row stored on the users table at
	// signup time; the two diverge after a home-tenant repair. Pull the
	// active tenant id from context instead.
	var tenant *types.Tenant
	activeTenantID, _ := types.TenantIDFromContext(ctx)
	if activeTenantID == 0 {
		activeTenantID = user.TenantID
	}
	if activeTenantID > 0 {
		tenant, err = h.tenantService.GetTenantByID(ctx, activeTenantID)
		if err != nil {
			logger.Warnf(ctx, "Failed to get tenant info for user %s, tenant ID %d: %v", user.Email, activeTenantID, err)
			// Don't fail the request if tenant info is not available
		}
	}
	userInfo := user.ToUserInfo()
	// 同步返回当前用户的 memberships，让前端在页面刷新（仅命中 /auth/me）
	// 后也能恢复 currentTenantRole，避免角色信息只在 login 那一刻可用。
	memberships := h.userService.BuildLoginMemberships(ctx, user, tenant)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user":            userInfo,
			"tenant":          dto.NewTenantResponse(ctx, tenant),
			"memberships":     memberships,
			"tenant_required": tenant == nil,
		},
	})
}

// updateMyPreferencesRequest is the body for PUT /auth/me/preferences.
// Fields are pointers so the handler can distinguish "key not present"
// (preserve existing value) from "explicit false". See
// types.UserPreferences for the persistence-layer counterpart.
type updateMyPreferencesRequest struct{}

// UpdateMyPreferences godoc
// @Summary      更新当前用户的个性化设置
// @Description  按 PATCH 语义合并用户偏好（仅覆盖请求体里出现的字段，其余字段保持不变），
// @Description  数据存放在 users.preferences (JSON)，跨设备/浏览器自动同步。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      updateMyPreferencesRequest  true  "Preferences patch"
// @Success      200      {object}  map[string]interface{}      "更新后的偏好"
// @Failure      400      {object}  errors.AppError             "请求参数错误"
// @Failure      401      {object}  errors.AppError             "未授权"
// @Security     Bearer
// @Router       /auth/me/preferences [put]
func (h *AuthHandler) UpdateMyPreferences(c *gin.Context) {
	ctx := c.Request.Context()

	user, err := h.userService.GetCurrentUser(ctx)
	if err != nil {
		appErr := errors.NewUnauthorizedError("Failed to get user information").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	var req updateMyPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewValidationError("Invalid preferences request").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	patch := types.UserPreferences{}
	prefs, err := h.userService.UpdateUserPreferences(ctx, user.ID, patch)
	if err != nil {
		logger.Errorf(ctx, "Failed to update preferences for user %s: %v", user.Email, err)
		appErr := errors.NewBadRequestError("Failed to update preferences").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    prefs,
	})
}

// ChangePassword godoc
// @Summary      修改密码
// @Description  修改当前用户的登录密码。新密码须满足 8–32 位且同时包含字母与数字；开启复杂密码后还需包含大小写与特殊字符。成功后所有会话被撤销，需重新登录。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      object{old_password=string,new_password=string}  true  "密码修改请求"
// @Success      200      {object}  map[string]interface{}                           "修改成功"
// @Failure      400      {object}  errors.AppError                                  "请求参数错误"
// @Security     Bearer
// @Router       /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start password change")

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse password change request", err)
		appErr := errors.NewValidationError("Invalid password change request").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	// Get current user
	user, err := h.userService.GetCurrentUser(ctx)
	if err != nil {
		logger.Errorf(ctx, "Failed to get current user: %v", err)
		appErr := errors.NewUnauthorizedError("Failed to get user information").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	// Change password. Policy is enforced in the service after the old
	// password is verified so a wrong current credential is not masked
	// by a complexity error.
	err = h.userService.ChangePassword(ctx, user.ID, req.OldPassword, req.NewPassword)
	if err != nil {
		switch {
		case service.IsPasswordPolicyError(err):
			appErr := errors.NewValidationError("Password policy violation").
				WithDetails(service.DetailPasswordPolicy)
			_ = c.Error(appErr)
			return
		case stderrors.Is(err, service.ErrInvalidOldPassword):
			appErr := errors.NewBadRequestError("Current password is incorrect").
				WithDetails(service.DetailInvalidOldPassword)
			c.Error(appErr)
			return
		case stderrors.Is(err, service.ErrSamePassword):
			appErr := errors.NewValidationError("New password must differ from current password").
				WithDetails(service.DetailSamePassword)
			c.Error(appErr)
			return
		default:
			logger.Errorf(ctx, "Failed to change password: %v", err)
			appErr := errors.NewBadRequestError("Password change failed").WithDetails(err.Error())
			c.Error(appErr)
			return
		}
	}

	logger.Infof(ctx, "Password changed successfully for user: %s", user.Email)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password changed successfully",
	})
}

// GetAuthConfig godoc
// @Summary      获取认证配置
// @Description  返回当前部署的密码复杂度开关，供前端决定密码校验规则
// @Tags         认证
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "认证配置"
// @Router       /auth/config [get]
//
// GetAuthConfig is intentionally a no-auth endpoint: the frontend reads
// it on app load to learn which password complexity rules to apply. We
// expose only what the UI strictly needs; other config stays internal.
func (h *AuthHandler) GetAuthConfig(c *gin.Context) {
	complexPasswordEnabled := service.ResolveComplexPasswordEnabled(
		c.Request.Context(),
		h.configInfo,
		h.systemSettingSvc,
	)
	c.JSON(http.StatusOK, gin.H{
		"success":                  true,
		"complex_password_enabled": complexPasswordEnabled,
	})
}

// @Summary      自动初始化（Lite 桌面版）
// @Description  Lite 版专用：首次启动时自动创建默认用户和空间并返回令牌，首次及后续启动均须通过桌面原生凭据认证，免除手动注册/登录流程
// @Tags         认证
// @Accept       json
// @Produce      json
// @Success      200  {object}  types.LoginResponse
// @Failure      403  {object}  errors.AppError  "非 Lite 版本"
// @Router       /auth/auto-setup [post]
func (h *AuthHandler) AutoSetup(c *gin.Context) {
	ctx := c.Request.Context()

	if Edition != "lite" {
		appErr := errors.NewForbiddenError("auto-setup is only available in lite edition")
		c.Error(appErr)
		return
	}

	if liteSetupToken == "" ||
		subtle.ConstantTimeCompare([]byte(c.GetHeader("X-WeKnora-Desktop-Token")), []byte(liteSetupToken)) != 1 {
		_ = c.Error(errors.NewUnauthorizedError("desktop authentication is required"))
		return
	}

	const defaultEmail = "admin@weknora.local"

	user, _ := h.userService.GetUserByEmail(ctx, defaultEmail)
	if user == nil {
		logger.Info(ctx, "Auto-setup: creating default user and tenant for lite edition")

		randomBytes := make([]byte, 24)
		if _, err := rand.Read(randomBytes); err != nil {
			appErr := errors.NewInternalServerError("auto-setup failed: unable to generate credentials")
			c.Error(appErr)
			return
		}
		randomPassword := base64.RawURLEncoding.EncodeToString(randomBytes)
		randomUsername := fmt.Sprintf("user_%s", base64.RawURLEncoding.EncodeToString(randomBytes[:6]))

		_, err := h.userService.Register(ctx, &types.RegisterRequest{
			Username: randomUsername,
			Email:    defaultEmail,
			Password: randomPassword,
		})
		if err != nil {
			logger.Errorf(ctx, "Auto-setup: failed to register default user: %v", err)
			appErr := errors.NewInternalServerError("auto-setup failed").WithDetails(err.Error())
			c.Error(appErr)
			return
		}
		user, _ = h.userService.GetUserByEmail(ctx, defaultEmail)
		if user == nil {
			appErr := errors.NewInternalServerError("auto-setup failed: user not found after registration")
			c.Error(appErr)
			return
		}
	}

	accessToken, refreshToken, err := h.userService.GenerateTokens(ctx, user)
	if err != nil {
		logger.Errorf(ctx, "Auto-setup: failed to generate tokens: %v", err)
		appErr := errors.NewInternalServerError("auto-setup failed").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	tenant, _ := h.tenantService.GetTenantByID(ctx, user.TenantID)

	logger.Info(ctx, "Auto-setup: completed successfully")
	c.JSON(http.StatusOK, dto.NewAuthLoginResponse(&types.LoginResponse{
		Success:      true,
		Message:      "Auto-setup successful",
		User:         user,
		ActiveTenant: tenant,
		Memberships: []types.Membership{{
			TenantID:   user.TenantID,
			TenantName: tenantNameOrEmpty(tenant),
			Role:       types.TenantRoleOwner,
		}},
		Token:        accessToken,
		RefreshToken: refreshToken,
	}))
}

// tenantNameOrEmpty returns t.Name when t is non-nil, "" otherwise.
// Used by AutoSetup to populate Membership.TenantName without crashing
// if the tenant lookup failed.
func tenantNameOrEmpty(t *types.Tenant) string {
	if t == nil {
		return ""
	}
	return t.Name
}

// ValidateToken godoc
// @Summary      验证令牌
// @Description  验证访问令牌是否有效
// @Tags         认证
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "令牌有效"
// @Failure      401  {object}  errors.AppError         "令牌无效"
// @Security     Bearer
// @Router       /auth/validate [get]
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start token validation")

	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		logger.Error(ctx, "Missing Authorization header")
		appErr := errors.NewValidationError("Authorization header is required")
		c.Error(appErr)
		return
	}

	// Parse Bearer token
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		logger.Error(ctx, "Invalid Authorization header format")
		appErr := errors.NewValidationError("Invalid Authorization header format")
		c.Error(appErr)
		return
	}

	token := tokenParts[1]

	// Validate token
	user, _, err := h.userService.ValidateToken(ctx, token)
	if err != nil {
		logger.Errorf(ctx, "Failed to validate token: %v", err)
		appErr := errors.NewUnauthorizedError("Token validation failed").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	logger.Infof(ctx, "Token validated successfully for user: %s", user.Email)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Token is valid",
		"user":    user.ToUserInfo(),
	})
}
