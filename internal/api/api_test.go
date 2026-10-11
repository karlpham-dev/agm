package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOAuthConfig_Env(t *testing.T) {
	t.Setenv("AGM_CLIENT_ID", "test-client-id")
	t.Setenv("AGM_CLIENT_SECRET", "test-client-secret")

	cfg, err := LoadOAuthConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.ClientID != "test-client-id" || cfg.ClientSecret != "test-client-secret" {
		t.Fatalf("unexpected config values: %+v", cfg)
	}
}

func TestLoadOAuthConfig_File(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("AGM_DATA_DIR", tempDir)
	t.Setenv("AGM_CLIENT_ID", "")
	t.Setenv("AGM_CLIENT_SECRET", "")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")

	// Missing config should error
	_, err := LoadOAuthConfig()
	if err == nil {
		t.Fatal("expected error when config is missing, got nil")
	}

	// Write oauth.json in AGM_DATA_DIR
	content := []byte(`{"client_id": "file-client-id", "client_secret": "file-client-secret"}`)
	if err := os.WriteFile(filepath.Join(tempDir, "oauth.json"), content, 0o600); err != nil {
		t.Fatalf("failed to write test oauth.json: %v", err)
	}

	cfg, err := LoadOAuthConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.ClientID != "file-client-id" || cfg.ClientSecret != "file-client-secret" {
		t.Fatalf("unexpected config values: %+v", cfg)
	}
}
