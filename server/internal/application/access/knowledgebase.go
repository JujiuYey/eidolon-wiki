// Package access resolves resource permissions independently of HTTP routing.
// Tenant roles, API-key capabilities, and resource ownership guards remain
// separate checks; a KB grant does not replace them.
package access

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
)

// Resource access errors retain distinct adapter response mappings.
var (
	ErrUnauthorized = errors.New("kb_access: unauthorized")
	ErrNotFound     = errors.New("kb_access: not found")
	ErrForbidden    = errors.New("kb_access: forbidden")
)

// KBRequest keeps the authenticated caller separate from the resource tenant.
type KBRequest struct {
	Caller types.Caller
}

// KBAccess binds a resolved resource permission to its original caller.
type KBAccess struct {
	KnowledgeBase     *types.KnowledgeBase
	Caller            types.Caller
	EffectiveTenantID uint64
	Permission        types.OrgMemberRole
	// A read resolution must not mint a write grant even for the KB owner.
	operationPermission types.OrgMemberRole
}

// Context scopes resource operations without changing the authenticated caller.
func (a *KBAccess) Context(ctx context.Context) context.Context {
	if a == nil || a.KnowledgeBase == nil {
		return ctx
	}
	scoped := a.WithGrant(ctx)
	if types.CallerFromContext(scoped) != a.Caller.Normalize() {
		return ctx
	}
	return types.WithExecutionTenant(scoped, a.EffectiveTenantID)
}

// WithGrant carries this authorization without changing execution scope.
func (a *KBAccess) WithGrant(ctx context.Context) context.Context {
	if a == nil || a.KnowledgeBase == nil {
		return ctx
	}
	caller := a.Caller.Normalize()
	if existing, ok := ctx.Value(types.CallerContextKey).(types.Caller); ok && existing != caller {
		return ctx // A grant cannot replace an already authenticated caller.
	}
	grants, _ := ctx.Value(types.KBGrantsContextKey).([]kbGrant)
	permission := a.Permission
	if a.operationPermission.IsValid() && permission.HasPermission(a.operationPermission) {
		permission = a.operationPermission
	}
	grants = append(
		append([]kbGrant(nil), grants...),
		kbGrant{caller: caller, kbID: a.KnowledgeBase.ID, tenantID: a.EffectiveTenantID, permission: permission},
	)
	return context.WithValue(types.WithCaller(ctx, caller), types.KBGrantsContextKey, grants)
}

// ResolveKB authorizes a server-loaded KB (or a document's persisted KB/tenant
// reference). Its tenant is authoritative; the caller must own it. It never
// treats an effective resource tenant as the caller.
func ResolveKB(ctx context.Context, request KBRequest, kb *types.KnowledgeBase, required types.OrgMemberRole,
) (*KBAccess, error) {
	if request.Caller.TenantID == 0 {
		return nil, ErrUnauthorized
	}
	if kb == nil || kb.ID == "" {
		return nil, ErrNotFound
	}
	if err := types.AuthorizeTenantAPIKeyKnowledgeBases(ctx, kb.ID); err != nil {
		return nil, err
	}
	request.Caller = request.Caller.Normalize()
	if caller, ok := ctx.Value(types.CallerContextKey).(types.Caller); ok && caller != request.Caller {
		return nil, ErrForbidden
	}
	if kb.TenantID != request.Caller.TenantID {
		return nil, ErrForbidden
	}
	return &KBAccess{
		KnowledgeBase: kb, Caller: request.Caller,
		EffectiveTenantID: kb.TenantID, Permission: types.OrgRoleAdmin, operationPermission: required,
	}, nil
}
