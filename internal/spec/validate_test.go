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

func TestValidateUncoveredTrack(t *testing.T) {
	c := baseCourse()
	c.Segments = c.Segments[:1] // drops track 2
	tr, st, ps := fixtures()
	if errs := Validate(c, tr, st, ps, 5); len(errs) == 0 {
		t.Fatal("expected uncovered-track error")
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
