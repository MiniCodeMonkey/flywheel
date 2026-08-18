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
