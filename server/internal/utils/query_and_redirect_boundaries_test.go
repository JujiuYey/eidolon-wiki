package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSSRFCrossOriginRedirect(t *testing.T) {
	reached := false
	sink := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { reached = true }))
	defer sink.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, sink.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	// Test the redirect policy with a plain local transport; target blocking
	// must happen even when both destinations would otherwise be trusted.
	client := &http.Client{CheckRedirect: newSSRFCheckRedirect(10)}
	_, err := client.Post(source.URL, "application/json", strings.NewReader(`{"client_secret":"synthetic"}`))
	require.Error(t, err)
	require.False(t, reached)
	req, err := http.NewRequest("GET", "https://example.com", nil)
	require.NoError(t, err)
	req.Header.Set("Custom-Secret-Header", "synthetic")
	req.Header.Set("Authorization", "Bearer synthetic")
	req.Header.Set("Accept", "application/json")
	stripRedirectSensitiveHeaders(req)
	require.Empty(t, req.Header.Get("Custom-Secret-Header"))
	require.Empty(t, req.Header.Get("Authorization"))
	require.Equal(t, "application/json", req.Header.Get("Accept"))
}

func TestKnownExampleKeyCannotSign(t *testing.T) {
	t.Setenv("SYSTEM_SIGNING_KEY", "")
	t.Setenv("SYSTEM_AES_KEY", "weknora-system-aes-key-32bytes!!")
	require.Nil(t, SystemHMACKey())
	_, err := SignFileURL("https://example.com", "local://1/a", 1, 0)
	require.Error(t, err)
}
