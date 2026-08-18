// internal/cli/lookups_maps.go  (helpers shared by preview/apply)
package cli

import (
	"os"
	"path/filepath"

	"github.com/minicodemonkey/flywheel/internal/config"
	"github.com/minicodemonkey/flywheel/internal/mowl"
	"github.com/minicodemonkey/flywheel/internal/spec"
)

func segmentTypeMap() map[string]int { return mowl.SegmentTypeAlias }
func positionMap() map[string]int    { return mowl.PositionAlias }

// loadStyles reads styles.yaml from the config dir, falling back to the empty set.
func loadStyles() (spec.Styles, error) {
	dir, err := config.Dir()
	if err != nil {
		return spec.Styles{}, err
	}
	b, err := os.ReadFile(filepath.Join(dir, "styles.yaml"))
	if os.IsNotExist(err) {
		return spec.Styles{}, nil
	}
	if err != nil {
		return spec.Styles{}, err
	}
	return spec.ParseStyles(b)
}

// playlistHydrated reports whether an imported playlist has enough tracks and
// every track carries a duration (required for validation). BPM may still be
// filling in — see bpmComplete/missingBPM.
func playlistHydrated(p mowl.Playlist, want int) bool {
	if len(p.Tracks) == 0 || len(p.Tracks) < want {
		return false
	}
	for _, t := range p.Tracks {
		if t.DurationMs <= 0 {
			return false
		}
	}
	return true
}

// bpmComplete reports whether every track has a non-zero BPM (Tempo).
func bpmComplete(p mowl.Playlist) bool { return missingBPM(p) == 0 }

// missingBPM counts tracks whose BPM has not populated yet.
func missingBPM(p mowl.Playlist) int {
	n := 0
	for _, t := range p.Tracks {
		if t.BPM <= 0 {
			n++
		}
	}
	return n
}
