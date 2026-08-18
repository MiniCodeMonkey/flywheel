package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitWritesStyles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "flywheel", "styles.yaml")); err != nil {
		t.Fatalf("styles.yaml not written: %v", err)
	}
}
