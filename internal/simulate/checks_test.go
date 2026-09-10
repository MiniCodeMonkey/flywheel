package simulate

import (
	"strings"
	"testing"

	"github.com/minicodemonkey/flywheel/internal/spec"
)

func course(durs ...int) *spec.Course {
	var ivs []spec.Interval
	for i, d := range durs {
		band := bands[2]
		if i%2 == 1 {
			band = bands[4]
		}
		ivs = append(ivs, spec.Interval{Duration: d, Cadence: [2]int{80, 81},
			Intensity: spec.IntensityValue{From: band[0], To: band[1]}, Position: "seated"})
	}
	return &spec.Course{Segments: []spec.Segment{{Name: "W", Type: "intervals", Intervals: ivs}}}
}

func TestAlignmentPerfectWhenEdgesMatchSections(t *testing.T) {
	tracks := []Track{trk(1, 120, Section{Duration: 40, Loudness: -10},
		Section{Duration: 40, Loudness: -4}, Section{Duration: 40, Loudness: -8})}
	tl := Build(course(40, 40, 40), tracks, 0)
	a := tl.Alignment(3)
	if a.WithinTolerance != 1 {
		t.Fatalf("alignment %v, want every edge on a section boundary; worst %v", a.WithinTolerance, a.Worst)
	}
}

func TestAlignmentCatchesDrift(t *testing.T) {
	tracks := []Track{trk(1, 120, Section{Duration: 40, Loudness: -10},
		Section{Duration: 40, Loudness: -4}, Section{Duration: 40, Loudness: -8})}
	tl := Build(course(25, 55, 40), tracks, 0) // first edge at 25s, section at 40s
	a := tl.Alignment(3)
	if a.WithinTolerance == 1 {
		t.Fatal("expected drift to be reported")
	}
	if len(a.Worst) == 0 || a.Worst[0].Off < 10 {
		t.Fatalf("expected a ~15s miss in the worst list, got %v", a.Worst)
	}
}

func TestLoudnessMatchIsPositiveWhenHardLandsOnLoud(t *testing.T) {
	tracks := []Track{trk(1, 80, Section{Duration: 40, Loudness: -20}, Section{Duration: 40, Loudness: -3})}
	tl := Build(course(40, 40), tracks, 0) // block 2 is the harder one and the louder one
	if got := tl.LoudnessMatch(); got <= 0 {
		t.Fatalf("loudness match %v, want positive", got)
	}
}

func TestRideabilityFlagsCadenceJumpAndFlapping(t *testing.T) {
	c := course(30, 30, 30, 30)
	c.Segments[0].Intervals[1].Cadence = [2]int{110, 111} // 80 -> 110 across a boundary
	c.Segments[0].Intervals[1].Position = "standing"
	c.Segments[0].Intervals[2].Position = "seated"
	c.Segments[0].Intervals[3].Position = "standing"
	tl := Build(c, []Track{trk(1, 120, Section{Duration: 120, Loudness: -6})}, 0)
	issues := tl.Rideability()
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "cadence") {
		t.Errorf("expected a cadence jump to be flagged, got: %v", issues)
	}
	if !strings.Contains(joined, "position") {
		t.Errorf("expected position flapping to be flagged, got: %v", issues)
	}
}

func TestMonotonyMeasuresLongestUnchangedStretch(t *testing.T) {
	c := course(60, 60)
	c.Segments[0].Intervals[1].Intensity = c.Segments[0].Intervals[0].Intensity // same zone
	tl := Build(c, []Track{trk(1, 120, Section{Duration: 120, Loudness: -6})}, 0)
	if got := tl.Monotony(); got != 120 {
		t.Errorf("monotony %ds, want 120", got)
	}
}

func TestTrackStartCountsAsAMusicalBoundary(t *testing.T) {
	tracks := []Track{
		trk(1, 100, Section{Duration: 100, Loudness: -6}),
		trk(2, 100, Section{Duration: 100, Loudness: -6}),
	}
	tl := Build(course(100, 100), tracks, 0) // second block starts exactly where track 2 does
	if a := tl.Alignment(3); a.WithinTolerance != 1 {
		t.Fatalf("a block starting on a new song should align; got %v, worst %v", a.WithinTolerance, a.Worst)
	}
}

func TestCadenceJumpAtATrackChangeIsNotFlagged(t *testing.T) {
	c := course(100, 100)
	c.Segments[0].Intervals[1].Cadence = [2]int{110, 111}
	tl := Build(c, []Track{
		trk(1, 100, Section{Duration: 100, Loudness: -6}),
		trk(2, 100, Section{Duration: 100, Loudness: -6}),
	}, 0)
	for _, s := range tl.Rideability() {
		if strings.Contains(s, "cadence") {
			t.Fatalf("a new song may bring a new cadence, but it was flagged: %v", s)
		}
	}
}
