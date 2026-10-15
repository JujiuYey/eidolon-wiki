package storageallowlist

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowedMap_DefaultAllowsAll(t *testing.T) {
	t.Setenv(AllowListEnv, "")
	allowed := AllowedMap()
	for _, provider := range Supported() {
		assert.True(t, allowed[provider], provider)
	}
}

func TestAllowedMap_RespectsEnv(t *testing.T) {
	t.Setenv(AllowListEnv, "minio")
	allowed := AllowedMap()
	assert.True(t, allowed["minio"])
	assert.False(t, allowed["local"])
	// Providers this build does not ship can never be allowed, even when
	// STORAGE_ALLOW_LIST names them.
	t.Setenv(AllowListEnv, "minio,cos,obs")
	allowed = AllowedMap()
	assert.True(t, allowed["minio"])
	assert.False(t, allowed["cos"])
	assert.False(t, allowed["obs"])
}

func TestFirstAllowed(t *testing.T) {
	t.Setenv(AllowListEnv, "minio")
	assert.Equal(t, "minio", FirstAllowed())
}

func TestAllowedList(t *testing.T) {
	t.Setenv(AllowListEnv, "minio,local")
	assert.Equal(t, []string{"local", "minio"}, AllowedList())
}

func TestIsAllowed_EmptyProvider(t *testing.T) {
	t.Setenv(AllowListEnv, "minio")
	require.True(t, IsAllowed(""))
}
