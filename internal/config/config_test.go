package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadTokenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := SaveToken("tok-123"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "flywheel", "token"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("token mode = %v, want 0600", info.Mode().Perm())
	}
	got, err := LoadToken()
	if err != nil || got != "tok-123" {
		t.Fatalf("LoadToken = %q,%v", got, err)
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.APIBase != "https://api.mowl.com" || c.AppPublicKey == "" || c.ClientVersion == "" {
		t.Fatalf("defaults not applied: %+v", c)
	}
}
