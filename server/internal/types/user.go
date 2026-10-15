package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// UserPreferences holds per-user preferences persisted server-side
// so they sync across devices/browsers. Fields are pointers so we can
// distinguish "client didn't send this key" (leave existing value alone)
// from "client explicitly set false" — the partial-update merge in
// UpdateUserPreferences relies on this.
//
// Adding a new preference key:
//  1. Add a *T field below + JSON tag (snake_case, must match the front-end key).
//  2. Extend the merge logic in service.UserService.UpdateUserPreferences.
//  3. Surface the new knob in the frontend settings store.
//
// No DB DDL is required — preferences is a single jsonb column.
type UserPreferences struct {
	// Gallery holds the user's image-gallery UI state: the search
	// activation mode and the per-attribute search-field toggles. Written
	// by the gallery's mode / checkbox interactions via the preferences
	// PUT; read back by GET /knowledge-bases/:id/gallery-config so the
	// gallery renders the same activation state on every device. Nil = the
	// user never touched the gallery controls (mode defaults to "all").
	Gallery *GalleryUserPrefs `json:"gallery,omitempty"`
}

// GalleryUserPrefs is the user-tier slice of the gallery configuration. The
// full tier schema (types.GalleryPolicyTier) also allows usage overrides,
// but those are admin/operator concerns; the user layer only records the
// activation mode and which search fields they personally switched on.
type GalleryUserPrefs struct {
	// Mode is "all" (every search-eligible field active) or "custom"
	// (per-field Status governs). Empty = "all".
	Mode string `json:"mode,omitempty"`
	// Status maps a namespaced attribute id ("builtin:caption") to
	// "on"/"off". Only consulted in custom mode; an unrecorded field is
	// off. Source-declared on/off does not exist — activation is always
	// the user's own record.
	Status map[string]string `json:"status,omitempty"`
}

// Value implements driver.Valuer so GORM persists UserPreferences as
// JSON text (Postgres jsonb column / SQLite TEXT). Empty struct serialises
// to "{}", matching the NOT NULL DEFAULT '{}' column constraint.
func (p UserPreferences) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan implements sql.Scanner so GORM can hydrate UserPreferences back
// from the underlying column. Accept []byte (Postgres jsonb / SQLite blob)
// and string (some drivers hand TEXT as string) for portability.
func (p *UserPreferences) Scan(value interface{}) error {
	if value == nil {
		*p = UserPreferences{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("UserPreferences.Scan: unsupported type")
	}
	if len(data) == 0 {
		*p = UserPreferences{}
		return nil
	}
	return json.Unmarshal(data, p)
}

// User represents a user in the system
type User struct {
	// Unique identifier of the user
	ID string `json:"id"         gorm:"type:varchar(36);primaryKey"`
	// Username of the user
	Username string `json:"username"   gorm:"type:varchar(100);uniqueIndex;not null"`
	// Email address of the user
	Email string `json:"email"      gorm:"type:varchar(255);uniqueIndex;not null"`
	// Hashed password of the user
	PasswordHash string `json:"-"          gorm:"type:varchar(255);not null"`
	// Avatar URL of the user
	Avatar string `json:"avatar"     gorm:"type:varchar(500)"`
	// Workspace ID that the user belongs to
	TenantID uint64 `json:"tenant_id"  gorm:"index"`
	// Whether the user is active
	IsActive bool `json:"is_active"  gorm:"default:true"`
	// Whether the user is a system administrator (independent of workspace roles)
	IsSystemAdmin bool `json:"is_system_admin" gorm:"default:false;index"`
	// Per-user UI/feature preferences.
	// Stored as JSON (jsonb on Postgres, TEXT on SQLite) via the
	// driver.Valuer / sql.Scanner methods on UserPreferences.
	Preferences UserPreferences `json:"preferences" gorm:"type:jsonb;not null;default:'{}'"`
	// Creation time of the user
	CreatedAt time.Time `json:"created_at"`
	// Last updated time of the user
	UpdatedAt time.Time `json:"updated_at"`
	// Deletion time of the user
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Association relationship, not stored in the database
	Tenant *Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
}

// AuthToken represents an authentication token
type AuthToken struct {
	// Unique identifier of the token
	ID string `json:"id"         gorm:"type:varchar(36);primaryKey"`
	// User ID that owns this token
	UserID string `json:"user_id"    gorm:"type:varchar(36);index;not null"`
	// Token value (JWT or other format)
	Token string `json:"token"      gorm:"type:text;not null"`
	// Token type (access_token, refresh_token)
	TokenType string `json:"token_type" gorm:"type:varchar(50);not null"`
	// Token expiration time
	ExpiresAt time.Time `json:"expires_at"`
	// Whether the token is revoked
	IsRevoked bool `json:"is_revoked" gorm:"default:false"`
	// Creation time of the token
	CreatedAt time.Time `json:"created_at"`
	// Last updated time of the token
	UpdatedAt time.Time `json:"updated_at"`

	// Association relationship
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=2,max=50"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`

	// TenantProvisioning is server-controlled registration context. It is
	// deliberately excluded from JSON so a public caller cannot choose its
	// own tenancy semantics. Empty preserves the historical behaviour and is
	// treated as create_personal by UserService.Register.
	TenantProvisioning TenantProvisioningMode `json:"-"`
}

// AdminCreateUserRequest is the payload for a SystemAdmin provisioning a
// new local user via POST /api/v1/system/admin/users/create.
//
// Password is optional: when absent (or null), the service generates a
// random one and returns it exactly once. Any provided value, the
// empty string included, is subject to the password policy.
type AdminCreateUserRequest struct {
	Username string  `json:"username" binding:"required,min=2,max=50"`
	Email    string  `json:"email"    binding:"required,email"`
	Password *string `json:"password"`
}

// TenantProvisioningMode controls what UserService.Register does after it
// has validated the identity fields. Joining an existing tenant is handled
// by tenant member management, not by registration.
type TenantProvisioningMode string

const (
	TenantProvisioningCreatePersonal TenantProvisioningMode = "create_personal"
	TenantProvisioningTenantless     TenantProvisioningMode = "tenantless"
)

func (m TenantProvisioningMode) IsValid() bool {
	return m == TenantProvisioningCreatePersonal || m == TenantProvisioningTenantless
}

// LoginResponse represents a login response
type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	User    *User  `json:"user,omitempty"`
	// ActiveTenant is the workspace whose ID is encoded in the issued JWT;
	// future requests are scoped to it. Defaults to the user's home
	// workspace on a fresh login.
	ActiveTenant *Tenant `json:"active_tenant,omitempty"`
	// Memberships lists every workspace the user can authenticate into,
	// along with their role in each. Always populated (length 1 for users
	// who only belong to their home workspace) so frontends can render a
	// workspace switcher without a follow-up request. Serialised without
	// omitempty so the field is always present as a JSON array (possibly
	// empty) — the "always populated" contract relies on the server side
	// guaranteeing a non-nil slice.
	Memberships  []Membership `json:"memberships"`
	Token        string       `json:"token,omitempty"`
	RefreshToken string       `json:"refresh_token,omitempty"`
}

// RegisterResponse represents a registration response
type RegisterResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message,omitempty"`
	User    *User   `json:"user,omitempty"`
	Tenant  *Tenant `json:"tenant,omitempty"`
}

// UserInfo represents user information for API responses
type UserInfo struct {
	ID            string          `json:"id"`
	Username      string          `json:"username"`
	Email         string          `json:"email"`
	Avatar        string          `json:"avatar"`
	TenantID      uint64          `json:"tenant_id"`
	IsActive      bool            `json:"is_active"`
	IsSystemAdmin bool            `json:"is_system_admin"`
	Preferences   UserPreferences `json:"preferences"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// ToUserInfo converts User to UserInfo (without sensitive data)
func (u *User) ToUserInfo() *UserInfo {
	return &UserInfo{
		ID:            u.ID,
		Username:      u.Username,
		Email:         u.Email,
		Avatar:        u.Avatar,
		TenantID:      u.TenantID,
		IsActive:      u.IsActive,
		IsSystemAdmin: u.IsSystemAdmin,
		Preferences:   u.Preferences,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}
