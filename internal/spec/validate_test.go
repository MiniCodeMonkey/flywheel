package spec

import "testing"

func baseCourse() Course {
	return Course{
		Segments: []Segment{
			{Name: "W", Type: "warmup", Tracks: []int{1},
				Intervals: []Interval{{Duration: 200, Position: "seated"}}},
			{Name: "M", Type: "intervals", Tracks: []int{2},
				Intervals: []Interval{{Duration: 180, Position: "standing"}}},
		},
	}
}

func fixtures() (map[int]TrackInfo, map[string]int, map[string]int) {
	tracks := map[int]TrackInfo{1: {DurationSec: 200}, 2: {DurationSec: 180}}
	segTypes := map[string]int{"warmup": 11, "intervals": 10}
	pos := map[string]int{"seated": 1, "standing": 2}
	return tracks, segTypes, pos
}

func TestValidateClean(t *testing.T) {
	tr, st, ps := fixtures()
	if errs := Validate(baseCourse(), tr, st, ps, 5); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateDurationMismatch(t *testing.T) {
	c := baseCourse()
	c.Segments[0].Intervals[0].Duration = 100 // track is 200s
	tr, st, ps := fixtures()
	if errs := Validate(c, tr, st, ps, 5); len(errs) == 0 {
		t.Fatal("expected duration mismatch error")
	}
}

func TestValidateUncoveredInteriorTrack(t *testing.T) {
	c := baseCourse()
	c.Segments[1].Tracks = []int{3} // track 2 skipped, track 3 still covered
	c.Segments[1].Intervals[0].Duration = 160
	tr, st, ps := fixtures()
	tr[3] = TrackInfo{DurationSec: 160}
	if errs := Validate(c, tr, st, ps, 5); len(errs) == 0 {
		t.Fatal("expected uncovered-track error for a gap before a covered track")
	}
}

func TestValidateBadType(t *testing.T) {
	c := baseCourse()
	c.Segments[0].Type = "bogus"
	tr, st, ps := fixtures()
	if errs := Validate(c, tr, st, ps, 5); len(errs) == 0 {
		t.Fatal("expected unknown-type error")
	}
}

func TestValidateAllowsSegmentBoundaryOffTrackBoundary(t *testing.T) {
	c := baseCourse()
	// the recovery starts mid-track: W gives up 45s, M takes them on
	c.Segments[0].Intervals[0].Duration = 155
	c.Segments[1].Intervals[0].Duration = 225
	tr, st, ps := fixtures()
	if errs := Validate(c, tr, st, ps, 5); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateRejectsDriftBeyondLimit(t *testing.T) {
	c := baseCourse()
	c.Segments[0].Intervals[0].Duration = 200 - (maxSegmentDriftSec + 10)
	c.Segments[1].Intervals[0].Duration = 180 + (maxSegmentDriftSec + 10)
	tr, st, ps := fixtures()
	if errs := Validate(c, tr, st, ps, 5); len(errs) == 0 {
		t.Fatal("expected a segment drift error")
	}
}

func TestValidateRejectsCourseTotalMismatch(t *testing.T) {
	c := baseCourse()
	c.Segments[0].Intervals[0].Duration = 160 // 40s vanish from the course
	tr, st, ps := fixtures()
	errs := Validate(c, tr, st, ps, 5)
	if len(errs) == 0 {
		t.Fatal("expected a course total error")
	}
}

func TestValidateAllowsUncoveredCooldownTail(t *testing.T) {
	c := baseCourse()
	c.Segments = c.Segments[:1] // track 2 is left to play out as the cooldown
	tr, st, ps := fixtures()
	if errs := Validate(c, tr, st, ps, 5); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateRejectsGapBeforeCoveredTrack(t *testing.T) {
	c := baseCourse()
	c.Segments[0].Tracks = []int{2} // track 1 skipped, track 2 covered
	c.Segments[1].Tracks = []int{}
	c.Segments[0].Intervals[0].Duration = 180
	c.Segments[1].Intervals[0].Duration = 0
	tr, st, ps := fixtures()
	if errs := Validate(c, tr, st, ps, 5); len(errs) == 0 {
		t.Fatal("expected an error for a skipped track before a covered one")
	}
}
