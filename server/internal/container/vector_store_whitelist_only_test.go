package container

import (
	"errors"
	"testing"

	secutils "github.com/Tencent/WeKnora/internal/utils"
)

func whitelistOnly(t *testing.T, whitelist, only string) {
	t.Helper()
	t.Setenv("SSRF_WHITELIST", whitelist)
	t.Setenv("SSRF_WHITELIST_EXTRA", "")
	t.Setenv("SSRF_DNS_WHITELIST_ONLY", only)
	secutils.ResetSSRFWhitelistForTest()
	t.Cleanup(secutils.ResetSSRFWhitelistForTest)
}

// gRPC resolves its own target before any dialer runs, so an env-configured
// vector store host can only be judged before the client exists (#3378).
// The helper must refuse without opening a connection.
func TestEnvVectorStoreClientsRefuseNonWhitelistedHost(t *testing.T) {
	whitelistOnly(t, "qdrant", "true")

	_, err := newEnvQdrantClient("outside.invalid", 6334, "", false)
	if !errors.Is(err, secutils.ErrSSRFHostNotWhitelisted) {
		t.Fatalf("qdrant: expected the whitelist refusal, got %v", err)
	}

	// The address carries a port the whitelist never matches on; the host
	// inside it is what must be compared. Past the name check the qdrant
	// client is lazy — no connection is opened here.
	if _, err := newEnvQdrantClient("qdrant", 6334, "", false); errors.Is(err, secutils.ErrSSRFHostNotWhitelisted) {
		t.Fatalf("a whitelisted host must pass the name check, got %v", err)
	}
}

func TestEnvVectorStoreClientsIgnoreTheWhitelistWhenTheModeIsOff(t *testing.T) {
	whitelistOnly(t, "qdrant", "false")

	_, err := newEnvQdrantClient("outside.invalid", 6334, "", false)
	if errors.Is(err, secutils.ErrSSRFHostNotWhitelisted) {
		t.Fatalf("with the mode off nothing is refused for being off-whitelist, got %v", err)
	}
}
