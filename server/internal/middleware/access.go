package middleware

import (
	"strconv"
	"strings"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

// access.go centralises the tenant-access check that previously lived
// scattered across handlers:
//
//   - "must the URL :id match the active tenant" — used by every
//     tenant-scoped route. Was duplicated in tenant.go (4 callers) and
//     tenant_member.go (1 caller) before; now lives here as a route
//     middleware so the route declaration is the single source of
//     truth.
//
// Rejections are reported via c.Error(*apperrors.AppError) + c.Abort(),
// not c.JSON: this keeps the response shape consistent with the
// {success,error:{code,message,details}} envelope that ErrorHandler
// renders for the rest of the API. The handler-level checks these
// guards replaced (authorizeTenantAccess, resolveTenantIDFromPath,
// ListAllTenants/SearchTenants if-blocks) all used that envelope, so
// preserving it avoids breaking SDK / frontend consumers that key off
// `error.code`.

// RequirePathTenantMatch gates a route on the URL :id matching the
// caller's active tenant context.
//
// The router places this on the /tenants/:id group so every per-tenant
// endpoint (GetTenant / UpdateTenant / api-key management) shares the
// same check, replacing what was previously a copy-pasted block in each
// handler.
//
// Reads :id off c.Param. If the param is missing or non-numeric the
// request is rejected as 400 — the route can't have meaningfully
// matched without it.
func RequirePathTenantMatch() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		raw := strings.TrimSpace(c.Param("id"))
		if raw == "" {
			_ = c.Error(apperrors.NewValidationError("workspace id is required"))
			c.Abort()
			return
		}
		pathTenantID, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || pathTenantID == 0 {
			_ = c.Error(apperrors.NewValidationError("workspace id must be a positive integer"))
			c.Abort()
			return
		}
		ctxTenantID, ok := types.TenantIDFromContext(ctx)
		if !ok || ctxTenantID == 0 {
			// Auth middleware should always have set this; if not we
			// fail closed rather than silently treating "no context" as
			// a match.
			logger.Warnf(ctx, "[rbac] path-tenant-match: no tenant in ctx, path=%s", c.Request.URL.Path)
			_ = c.Error(apperrors.NewUnauthorizedError("workspace context missing"))
			c.Abort()
			return
		}
		if pathTenantID == ctxTenantID {
			c.Next()
			return
		}
		uid, _ := types.UserIDFromContext(ctx)
		logger.Warnf(ctx,
			"[rbac] path-tenant-match rejected: user=%s ctx_tenant=%d path_tenant=%d path=%s",
			uid, ctxTenantID, pathTenantID, c.Request.URL.Path)
		_ = c.Error(apperrors.NewForbiddenError(
			"Access denied: URL workspace does not match the active workspace"))
		c.Abort()
	}
}
