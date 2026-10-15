package middleware

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// fakeTenantService is a minimal TenantService stand-in for middleware tests:
// every tenant lookup resolves to the single tenant it holds (nil when unset)
// and everything else is a no-op.
type fakeTenantService struct {
	tenant *types.Tenant
}

func (f *fakeTenantService) GetTenantByID(ctx context.Context, id uint64) (*types.Tenant, error) {
	return f.tenant, nil
}

func (f *fakeTenantService) CreateTenant(ctx context.Context, tenant *types.Tenant) (*types.Tenant, error) {
	return nil, nil
}

func (f *fakeTenantService) GetTenantsByIDs(ctx context.Context, ids []uint64) (map[uint64]*types.Tenant, error) {
	return nil, nil
}

func (f *fakeTenantService) UpdateTenant(ctx context.Context, tenant *types.Tenant) (*types.Tenant, error) {
	return nil, nil
}

func (f *fakeTenantService) DeleteTenant(ctx context.Context, id uint64) error {
	return nil
}

func (f *fakeTenantService) ListTenants(ctx context.Context) ([]*types.Tenant, error) {
	return nil, nil
}

func (f *fakeTenantService) BulkSetStorageQuota(ctx context.Context, quotaBytes int64) (int64, error) {
	return 0, nil
}

func (f *fakeTenantService) GetTenantByIDForUser(
	ctx context.Context, tenantID uint64, userID string,
) (*types.Tenant, error) {
	return f.tenant, nil
}

func (f *fakeTenantService) GetWeKnoraCloudCredentials(ctx context.Context) *types.WeKnoraCloudCredentials {
	return nil
}

var _ interfaces.TenantService = (*fakeTenantService)(nil)
