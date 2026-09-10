package simulate

import (
	"fmt"
	"strings"
)

// Report is the deterministic result of walking a ride against its music.
// Same course and same playlist always produce the same bytes, so it can be
// diffed between runs and read by something other than a person.
type Report struct {
	Name      string          `json:"name"`
	Duration  int             `json:"duration_sec"`
	MusicLen  int             `json:"music_sec"`
	Blocks    int             `json:"blocks"`
	Tracks    int             `json:"tracks"`
	Crossfade int             `json:"crossfade_sec"`
	Alignment AlignmentReport `json:"alignment"`
	Loudness  float64         `json:"loudness_match"`
	Monotony  int             `json:"monotony_sec"`
	Issues    []string        `json:"rideability_issues"`
}

// Run walks the ride and produces the report.
func Run(t *Timeline, tolerance int) Report {
	musicLen := 0
	if n := len(t.Tracks); n > 0 {
		musicLen = t.TrackStart[n-1] + t.Tracks[n-1].DurationSec
	}
	return Report{
		Duration: t.Duration(), MusicLen: musicLen,
		Blocks: len(t.Blocks), Tracks: len(t.Tracks), Crossfade: t.Crossfade,
		Alignment: t.Alignment(tolerance),
		Loudness:  t.LoudnessMatch(),
		Monotony:  t.Monotony(),
		Issues:    t.Rideability(),
	}
}

// Text renders the report for a person. The verdict lines name a time, not a
// percentage, so a problem can be found in the editor.
func (r Report) Text(tolerance int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", r.Name)
	fmt.Fprintf(&b, "  %s ride over %s of music, %d blocks across %d tracks, %ds crossfade\n\n",
		clock(r.Duration), clock(r.MusicLen), r.Blocks, r.Tracks, r.Crossfade)

	a := r.Alignment
	fmt.Fprintf(&b, "  music alignment   %.0f%% of %d edges land within %ds (median %ds off)\n",
		a.WithinTolerance*100, a.Edges, tolerance, a.Median)
	fmt.Fprintf(&b, "  loudness match    %+.2f  %s\n", r.Loudness, loudnessVerdict(r.Loudness))
	fmt.Fprintf(&b, "  longest flat run  %ds at one zone  %s\n", r.Monotony, monotonyVerdict(r.Monotony))

	if len(a.Worst) > 0 {
		fmt.Fprintf(&b, "\n  worst boundaries\n")
		for _, m := range a.Worst {
			fmt.Fprintf(&b, "    %-7s misses the nearest section by %ds\n", clock(m.At), m.Off)
		}
	}
	if len(r.Issues) > 0 {
		fmt.Fprintf(&b, "\n  rideability\n")
		for _, s := range r.Issues {
			fmt.Fprintf(&b, "    %s\n", s)
		}
	}
	return b.String()
}

// MOWL's own rides hold one zone for 191-537s, so a long flat run is not by
// itself a fault. Measured across six official programs.
func monotonyVerdict(sec int) string {
	if sec > 560 {
		return "(longer than any MOWL ride sampled)"
	}
	return "(MOWL's own run 191-537s)"
}

func loudnessVerdict(v float64) string {
	switch {
	case v >= 0.5:
		return "the hard part lands on the loud part"
	case v >= 0.25:
		return "loosely follows the music"
	case v >= 0:
		return "barely related to the music"
	}
	return "inverted: the ride works hardest where the music is quietest"
}
