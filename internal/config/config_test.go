package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Server.ProxyPort != 8080 {
		t.Fatalf("expected proxy port 8080, got %d", cfg.Server.ProxyPort)
	}
	if cfg.Policy.DefaultAction != "rewrite" {
		t.Fatalf("expected default action rewrite, got %s", cfg.Policy.DefaultAction)
	}
}

func TestLoadWithEnvExpansion(t *testing.T) {
	yamlContent := `
server:
  proxy_port: 8888
upstream:
  api_key: "${TEST_CLAIMGUARD_KEY}"
`
	tmpFile, err := os.CreateTemp("", "claimguard-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(yamlContent)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	os.Setenv("TEST_CLAIMGUARD_KEY", "secret-test-key-123")
	defer os.Unsetenv("TEST_CLAIMGUARD_KEY")

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.ProxyPort != 8888 {
		t.Fatalf("expected 8888, got %d", cfg.Server.ProxyPort)
	}
	if cfg.Upstream.APIKey != "secret-test-key-123" {
		t.Fatalf("expected secret-test-key-123, got %s", cfg.Upstream.APIKey)
	}
}
