package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
host: https://bigip.example.com
user: admin
pass: secret
insecure: true
partition: Common
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "https://bigip.example.com" || cfg.User != "admin" || cfg.Pass != "secret" || !cfg.Insecure || cfg.Partition != "Common" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadMissingImplicit(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"), false)
	if err != nil {
		t.Fatalf("missing file with explicit=false should not error, got %v", err)
	}
	if cfg == nil || cfg.Host != "" {
		t.Fatalf("expected empty config, got %+v", cfg)
	}
}

func TestLoadMissingExplicit(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"), true)
	if err == nil {
		t.Fatal("missing file with explicit=true should error")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("host: [unterminated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, true); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestDefaultPathXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	got := DefaultPath()
	want := "/tmp/xdg/f5tui/config.yaml"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
