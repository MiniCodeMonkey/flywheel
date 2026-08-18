// internal/cli/root_test.go
package cli

import (
	"bytes"
	"testing"
)

func TestRootCmdHasName(t *testing.T) {
	cmd := NewRootCmd()
	if cmd.Use != "flywheel" {
		t.Fatalf("Use = %q, want flywheel", cmd.Use)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help failed: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("expected help output")
	}
}
