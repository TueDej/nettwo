package config

import (
	"os"
	"path/filepath"
	"testing"
)

func useTestConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	testConfigDirMu.Lock()
	previous := testConfigDir
	testConfigDir = dir
	testConfigDirMu.Unlock()
	t.Cleanup(func() {
		testConfigDirMu.Lock()
		testConfigDir = previous
		testConfigDirMu.Unlock()
	})
	return dir
}

func TestLoadEnvironmentOverridesFile(t *testing.T) {
	dir := useTestConfigDir(t)
	t.Setenv("NETTWO_BASE_URL", "http://example.test")
	t.Setenv("NETTWO_REQUEST_TIMEOUT", "15")

	configPath := filepath.Join(dir, configFileName)
	if err := os.WriteFile(configPath, []byte("base_url: https://file.example\nrequest_timeout: 5\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "http://example.test" || cfg.RequestTimeout != 15 {
		t.Fatalf("environment did not override file config: %+v", cfg)
	}
}

func TestLoadRejectsInvalidTimeout(t *testing.T) {
	dir := useTestConfigDir(t)
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte("request_timeout: 0\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid timeout to be rejected")
	}
}
