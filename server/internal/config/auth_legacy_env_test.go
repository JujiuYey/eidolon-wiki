package config

import "testing"

func TestApplyAuthAndTenantDefaults_DefaultTenantMode(t *testing.T) {
	t.Run("historical default creates a personal tenant", func(t *testing.T) {
		t.Setenv("WEKNORA_AUTH_DEFAULT_TENANT_MODE", "")
		cfg := &Config{Auth: &AuthConfig{}}

		applyAuthAndTenantDefaults(cfg)

		if cfg.Auth.DefaultTenantMode != AuthDefaultTenantModeCreatePersonal {
			t.Fatalf("default_tenant_mode = %q, want %q", cfg.Auth.DefaultTenantMode, AuthDefaultTenantModeCreatePersonal)
		}
	})

	t.Run("environment overrides yaml", func(t *testing.T) {
		t.Setenv("WEKNORA_AUTH_DEFAULT_TENANT_MODE", AuthDefaultTenantModeTenantless)
		cfg := &Config{Auth: &AuthConfig{DefaultTenantMode: AuthDefaultTenantModeCreatePersonal}}

		applyAuthAndTenantDefaults(cfg)

		if cfg.Auth.DefaultTenantMode != AuthDefaultTenantModeTenantless {
			t.Fatalf("default_tenant_mode = %q, want %q", cfg.Auth.DefaultTenantMode, AuthDefaultTenantModeTenantless)
		}
	})

	t.Run("invalid environment value fails validation", func(t *testing.T) {
		t.Setenv("WEKNORA_AUTH_DEFAULT_TENANT_MODE", "create_magic")
		cfg := &Config{Auth: &AuthConfig{}}

		applyAuthAndTenantDefaults(cfg)

		if err := ValidateConfig(cfg); err == nil {
			t.Fatal("ValidateConfig unexpectedly accepted an invalid default tenant mode")
		}
	})
}

func TestApplyAuthAndTenantDefaults_ComplexPasswordEnabledEnv(t *testing.T) {
	t.Run("1 enables via ParseBool", func(t *testing.T) {
		t.Setenv("WEKNORA_AUTH_COMPLEX_PASSWORD_ENABLED", "1")
		cfg := &Config{Auth: &AuthConfig{}}
		applyAuthAndTenantDefaults(cfg)
		if !cfg.Auth.ComplexPasswordEnabled {
			t.Fatal("WEKNORA_AUTH_COMPLEX_PASSWORD_ENABLED=1 should enable complex passwords")
		}
	})
	t.Run("false disables", func(t *testing.T) {
		t.Setenv("WEKNORA_AUTH_COMPLEX_PASSWORD_ENABLED", "false")
		cfg := &Config{Auth: &AuthConfig{ComplexPasswordEnabled: true}}
		applyAuthAndTenantDefaults(cfg)
		if cfg.Auth.ComplexPasswordEnabled {
			t.Fatal("WEKNORA_AUTH_COMPLEX_PASSWORD_ENABLED=false should disable complex passwords")
		}
	})
	t.Run("unset leaves yaml", func(t *testing.T) {
		t.Setenv("WEKNORA_AUTH_COMPLEX_PASSWORD_ENABLED", "")
		cfg := &Config{Auth: &AuthConfig{ComplexPasswordEnabled: true}}
		applyAuthAndTenantDefaults(cfg)
		if !cfg.Auth.ComplexPasswordEnabled {
			t.Fatal("empty env should leave YAML complex-password flag untouched")
		}
	})
}
