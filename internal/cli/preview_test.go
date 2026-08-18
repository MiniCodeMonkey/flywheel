// internal/cli/preview_test.go
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewCommand(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	// minimal styles + course; preview needs no live playlist if course carries
	// track durations via an override flag --tracks (see impl note) OR inspect.
	course := `name: R
category: My Rides
playlist: { spotify_id: sp }
segments:
  - name: M
    type: intervals
    tracks: [1]
    intervals:
      - { duration: 60, cadence: [90,95], intensity: 100, position: seated }
`
	cf := filepath.Join(dir, "course.yaml")
	os.WriteFile(cf, []byte(course), 0o600)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"preview", cf, "--offline-track-seconds", "60"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("TOTAL")) {
		t.Fatalf("no preview output:\n%s", out.String())
	}
}
