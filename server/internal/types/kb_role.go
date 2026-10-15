package types

// OrgMemberRole is the knowledge-base permission ladder used by access
// resolution and route guards.
type OrgMemberRole string

const (
	// OrgRoleAdmin has full control over the tenant's knowledge bases
	OrgRoleAdmin OrgMemberRole = "admin"
	// OrgRoleEditor can edit knowledge base content but cannot manage settings
	OrgRoleEditor OrgMemberRole = "editor"
	// OrgRoleViewer can only view and search knowledge bases
	OrgRoleViewer OrgMemberRole = "viewer"
)

// IsValid checks if the role is valid
func (r OrgMemberRole) IsValid() bool {
	switch r {
	case OrgRoleAdmin, OrgRoleEditor, OrgRoleViewer:
		return true
	default:
		return false
	}
}

// HasPermission checks if this role has at least the required permission level
func (r OrgMemberRole) HasPermission(required OrgMemberRole) bool {
	roleLevel := map[OrgMemberRole]int{
		OrgRoleAdmin:  3,
		OrgRoleEditor: 2,
		OrgRoleViewer: 1,
	}
	return roleLevel[r] >= roleLevel[required]
}

// MinOrgRole returns whichever of a / b is the lower role on the
// admin > editor > viewer ladder. Used to apply caps when combining
// role grants from several sources. A zero/empty role is treated
// as "less than viewer" so it short-circuits to whatever the other
// argument is.
func MinOrgRole(a, b OrgMemberRole) OrgMemberRole {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a.HasPermission(b) {
		return b
	}
	return a
}
