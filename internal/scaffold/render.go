package scaffold

import (
	"fmt"
	"strings"

	"github.com/minicodemonkey/flywheel/internal/spec"
)

// Render writes a course as course.yaml. It is emitted by hand rather than
// marshalled so intervals stay one per line in flow style, which is what makes
// a scaffolded file readable and hand-editable.
func Render(c spec.Course, spotifyID string, targetMin, targetTSS int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "name: %q\n", c.Name)
	fmt.Fprintf(&b, "category: %q\n", c.Category)
	fmt.Fprintf(&b, "activity: %s\n", c.Activity)
	fmt.Fprintf(&b, "targets: { duration_min: %d, tss: %d }\n", targetMin, targetTSS)
	if len(c.Style) > 0 {
		fmt.Fprintf(&b, "style: [%s]\n", strings.Join(c.Style, ", "))
	}
	fmt.Fprintf(&b, "playlist:\n  spotify_id: %q\n  crossfade_sec: %d\nsegments:\n",
		spotifyID, c.Playlist.Crossfade())
	for _, s := range c.Segments {
		tracks := make([]string, len(s.Tracks))
		for i, t := range s.Tracks {
			tracks[i] = fmt.Sprint(t)
		}
		fmt.Fprintf(&b, "  - name: %q\n    type: %s\n    tracks: [%s]\n    intervals:\n",
			s.Name, s.Type, strings.Join(tracks, ", "))
		for _, iv := range s.Intervals {
			cycle := ""
			if iv.Cycle != "" {
				cycle = fmt.Sprintf(", cycle: %s", iv.Cycle)
			}
			fmt.Fprintf(&b, "      - { duration: %3d, cadence: [%d, %d], intensity: [%d, %d], position: %s%s }\n",
				iv.Duration, iv.Cadence[0], iv.Cadence[1], iv.Intensity.From, iv.Intensity.To, iv.Position, cycle)
		}
	}
	return b.String()
}
