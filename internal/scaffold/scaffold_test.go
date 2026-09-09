package scaffold

import (
	"testing"

	"github.com/minicodemonkey/flywheel/internal/mowl"
	"github.com/minicodemonkey/flywheel/internal/spec"
)

func track(idx, bpm, dur int, secs ...Section) Track {
	return Track{Index: idx, BPM: bpm, DurationSec: dur, Sections: secs}
}

func TestParseSegment(t *testing.T) {
	s, err := ParseSegment("Roll Out:warmup:1-3,5")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Roll Out" || s.Type != "warmup" {
		t.Fatalf("got %+v", s)
	}
	if got := s.Tracks; len(got) != 4 || got[0] != 1 || got[3] != 5 {
		t.Fatalf("tracks = %v", got)
	}
}

func TestParseSegmentRejectsBadInput(t *testing.T) {
	for _, in := range []string{"noparts", "A:warmup", "A:nosuchtype:1", "A:warmup:", "A:warmup:0", ":warmup:1"} {
		if _, err := ParseSegment(in); err == nil {
			t.Errorf("ParseSegment(%q) should have failed", in)
		}
	}
}

// Interval durations must sum to the track's real length or the course fails
// validation, so folding and splitting have to be exact.
func TestSectionsSumToTrackDuration(t *testing.T) {
	o := Defaults()
	tr := track(1, 120, 200,
		Section{Duration: 5, Loudness: -10},   // folds forward
		Section{Duration: 95.4, Loudness: -5}, // stays
		Section{Duration: 99.6, Loudness: -8},
	)
	total := 0.0
	for _, s := range sectionsOf(tr, o) {
		total += s.Duration
	}
	if int(total) != 200 {
		t.Fatalf("sections sum to %v, want 200", total)
	}
}

func TestSectionsFoldShortAndSplitLong(t *testing.T) {
	o := Options{MinSection: 13, MaxSection: 60}
	got := sectionsOf(track(1, 120, 190, Section{Duration: 180, Loudness: -5}, Section{Duration: 10, Loudness: -9}), o)
	for _, s := range got {
		if s.Duration > 60 {
			t.Fatalf("section of %vs exceeds max", s.Duration)
		}
	}
	if len(got) < 3 {
		t.Fatalf("expected the 180s section to split, got %d sections", len(got))
	}
}

func TestSectionsNeverSplitWhenMaxIsZero(t *testing.T) {
	got := sectionsOf(track(1, 120, 180, Section{Duration: 180, Loudness: -5}), Options{MinSection: 13})
	if len(got) != 1 {
		t.Fatalf("got %d sections, want 1", len(got))
	}
}

func TestCadenceLandsInRideableRange(t *testing.T) {
	for _, bpm := range []int{88, 124, 128, 146, 170, 175, 186, 189} {
		lo, _ := cadenceFor(bpm, "intervals")
		if lo < 55 || lo > 110 {
			t.Errorf("bpm %d -> %d rpm, outside rideable range", bpm, lo)
		}
	}
	if lo, _ := cadenceFor(124, "climb"); lo > 75 {
		t.Errorf("climb at 124bpm -> %d rpm, want a climbing cadence", lo)
	}
}

func demoTracks() []Track {
	mk := func(i int) Track {
		return track(i, 150, 120,
			Section{Duration: 40, Loudness: -20},
			Section{Duration: 40, Loudness: -10},
			Section{Duration: 40, Loudness: -3},
		)
	}
	return []Track{mk(1), mk(2)}
}

func TestBuildEndsWorkSegmentsHot(t *testing.T) {
	c, _, err := Build(demoTracks(),
		[]SegmentSpec{{Name: "W", Type: "warmup", Tracks: []int{1}}, {Name: "M", Type: "intervals", Tracks: []int{2}}},
		Defaults())
	if err != nil {
		t.Fatal(err)
	}
	last := c.Segments[1].Intervals[len(c.Segments[1].Intervals)-1]
	if last.Intensity.From != 121 || last.Position != "standing" {
		t.Fatalf("work segment ends %+v, want a standing fire interval", last)
	}
}

func TestBuildKeepsRecoverySoft(t *testing.T) {
	c, _, err := Build(demoTracks(),
		[]SegmentSpec{{Name: "R", Type: "recovery", Tracks: []int{1}}}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	for _, iv := range c.Segments[0].Intervals {
		if iv.Intensity.To > 75 {
			t.Fatalf("recovery interval reaches %d%%FTP", iv.Intensity.To)
		}
	}
}

// The loud section of a track must not end up easier than the quiet one.
func TestLouderSectionsRideHarder(t *testing.T) {
	c, _, err := Build(demoTracks(),
		[]SegmentSpec{{Name: "M", Type: "intervals", Tracks: []int{1}}},
		Options{TargetTSS: 200, MinSection: 13, MaxSection: 180})
	if err != nil {
		t.Fatal(err)
	}
	ivs := c.Segments[0].Intervals
	if ivs[0].Intensity.From > ivs[len(ivs)-1].Intensity.From {
		t.Fatalf("quiet section rides harder than the loud one: %v vs %v", ivs[0], ivs[len(ivs)-1])
	}
}

func TestBuildValidatesAgainstTrackDurations(t *testing.T) {
	c, _, err := Build(demoTracks(), []SegmentSpec{{Name: "M", Type: "intervals", Tracks: []int{1, 2}}}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	// validation sees the same crossfade-adjusted timeline the scaffolder built against
	tracks := spec.CrossfadeTracks(
		map[int]spec.TrackInfo{1: {DurationSec: 120}, 2: {DurationSec: 120}},
		spec.DefaultCrossfadeSec)
	if errs := spec.Validate(c, tracks, mowl.SegmentTypeAlias, mowl.PositionAlias, 5); len(errs) > 0 {
		t.Fatalf("scaffolded course should validate, got %v", errs)
	}
}

func TestBuildRejectsUnindexedTrack(t *testing.T) {
	tr := []Track{track(1, 150, 0, Section{Duration: 10, Loudness: -5})}
	if _, _, err := Build(tr, []SegmentSpec{{Name: "M", Type: "intervals", Tracks: []int{1}}}, Defaults()); err == nil {
		t.Fatal("expected an error for a track with no duration")
	}
}

func TestBuildTrimsEachTrackByTheCrossfade(t *testing.T) {
	tracks := []Track{
		{Index: 1, BPM: 120, DurationSec: 200, Sections: evenSections(200)},
		{Index: 2, BPM: 120, DurationSec: 180, Sections: evenSections(180)},
		{Index: 3, BPM: 120, DurationSec: 160, Sections: evenSections(160)},
	}
	segs := []SegmentSpec{{Name: "W", Type: "warmup", Tracks: []int{1}},
		{Name: "M", Type: "intervals", Tracks: []int{2, 3}}}
	o := Defaults()
	o.Crossfade = 10
	c, _, err := Build(tracks, segs, o)
	if err != nil {
		t.Fatal(err)
	}
	// tracks 1 and 2 each give 10s to the next track's fade-in; track 3 is whole
	want := map[string]int{"W": 190, "M": 170 + 160}
	for _, s := range c.Segments {
		got := 0
		for _, iv := range s.Intervals {
			got += iv.Duration
		}
		if got != want[s.Name] {
			t.Errorf("segment %q sums to %ds, want %ds", s.Name, got, want[s.Name])
		}
	}
}

func evenSections(total int) []Section {
	var out []Section
	for n := 0; n < total; n += 20 {
		d := 20.0
		if n+20 > total {
			d = float64(total - n)
		}
		out = append(out, Section{Duration: d, Loudness: -6 + float64(n%3)})
	}
	return out
}

func longTrack(idx, bpm, dur int) Track {
	return Track{Index: idx, BPM: bpm, DurationSec: dur, Sections: evenSections(dur)}
}

func allIntervals(c spec.Course) []spec.Interval {
	var out []spec.Interval
	for _, s := range c.Segments {
		out = append(out, s.Intervals...)
	}
	return out
}

func TestCadenceNeverUsesTwoThirds(t *testing.T) {
	// 186 BPM at two-thirds is 124 and at three-halves 93; only full/half/quarter are allowed
	for _, bpm := range []int{186, 128, 126, 134, 148, 170, 175} {
		c1, _ := cadenceFor(bpm, "intervals")
		for _, bad := range []int{int(float64(bpm) * 2 / 3), bpm / 3} {
			if bad >= 55 && bad <= 110 && c1 == bad {
				t.Errorf("bpm %d rode at %d, which is a two-thirds or one-third multiple", bpm, c1)
			}
		}
	}
}

func TestStandingNeverExceedsCadenceCap(t *testing.T) {
	o := Defaults()
	c, _, err := Build([]Track{longTrack(1, 186, 300), longTrack(2, 175, 300)},
		[]SegmentSpec{{Name: "M", Type: "intervals", Tracks: []int{1, 2}}}, o)
	if err != nil {
		t.Fatal(err)
	}
	for _, iv := range allIntervals(c) {
		if iv.Position == "standing" && iv.Cadence[0] > o.StandingCadenceMax {
			t.Fatalf("standing at %d rpm, cap is %d", iv.Cadence[0], o.StandingCadenceMax)
		}
	}
}

func TestNoStandingIntervalHasZeroCadence(t *testing.T) {
	c, _, err := Build([]Track{longTrack(1, 124, 400), longTrack(2, 130, 400)},
		[]SegmentSpec{{Name: "M", Type: "intervals", Tracks: []int{1, 2}}}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	for _, iv := range allIntervals(c) {
		if iv.Position == "standing" && iv.Cadence[0] == 0 && iv.Cycle != "acc" {
			t.Fatalf("standing interval has no cadence: %+v", iv)
		}
	}
}

func TestStandingRunsAreBroken(t *testing.T) {
	o := Defaults()
	c, _, err := Build([]Track{longTrack(1, 124, 600)},
		[]SegmentSpec{{Name: "C", Type: "climb", Tracks: []int{1}}}, o)
	if err != nil {
		t.Fatal(err)
	}
	run := 0
	for _, iv := range allIntervals(c) {
		if iv.Position == "standing" {
			run += iv.Duration
			if run > o.MaxStanding {
				t.Fatalf("standing run reached %ds, cap is %ds", run, o.MaxStanding)
			}
		} else {
			run = 0
		}
	}
}

func TestNoAdjacentIdenticalIntervals(t *testing.T) {
	c, _, err := Build([]Track{longTrack(1, 124, 400)},
		[]SegmentSpec{{Name: "M", Type: "intervals", Tracks: []int{1}}}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	ivs := c.Segments[0].Intervals
	for i := 1; i < len(ivs); i++ {
		a, b := ivs[i-1], ivs[i]
		if a.Cadence == b.Cadence && a.Intensity == b.Intensity && a.Position == b.Position && a.Cycle == b.Cycle {
			t.Fatalf("intervals %d and %d are identical: %+v", i-1, i, a)
		}
	}
}

func TestRecoveryAndCooldownAreOneSoftInterval(t *testing.T) {
	c, _, err := Build([]Track{longTrack(1, 120, 300), longTrack(2, 120, 100), longTrack(3, 120, 240)},
		[]SegmentSpec{
			{Name: "M", Type: "intervals", Tracks: []int{1}},
			{Name: "", Type: "recovery", Tracks: []int{2}},
			{Name: "", Type: "cooldown", Tracks: []int{3}},
		}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range c.Segments[1:] {
		if len(s.Intervals) != 1 {
			t.Fatalf("%s segment has %d intervals, want 1", s.Type, len(s.Intervals))
		}
		if s.Intervals[0].Intensity.To > 55 {
			t.Fatalf("%s segment rides at %v, want zone 1", s.Type, s.Intervals[0].Intensity)
		}
	}
}

func TestACCIntervalsAreShortSeatedAndCadenceFree(t *testing.T) {
	c, _, err := Build([]Track{longTrack(1, 124, 400), longTrack(2, 130, 400), longTrack(3, 118, 400)},
		[]SegmentSpec{{Name: "M", Type: "intervals", Tracks: []int{1, 2, 3}}}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	ivs := allIntervals(c)
	var acc int
	for _, iv := range ivs {
		if iv.Cycle != "acc" {
			continue
		}
		acc++
		if iv.Position != "seated" {
			t.Errorf("ACC interval is %s, want seated", iv.Position)
		}
		if iv.Cadence[0] != 0 {
			t.Errorf("ACC interval sets cadence %d; ACC overrules RPM so it must be 0", iv.Cadence[0])
		}
		if iv.Duration > 60 {
			t.Errorf("ACC interval is %ds, want a short burst", iv.Duration)
		}
	}
	if acc == 0 {
		t.Fatal("no ACC intervals were emitted")
	}
}

func TestParseSegmentAllowsUnnamedRecoveryAndCooldown(t *testing.T) {
	for _, in := range []string{":recovery:8", ":cooldown:16"} {
		if _, err := ParseSegment(in); err != nil {
			t.Errorf("ParseSegment(%q) = %v; MOWL's own recovery and cooldown segments are unnamed", in, err)
		}
	}
	if _, err := ParseSegment(":intervals:1-3"); err == nil {
		t.Error("a work segment still needs a name")
	}
}
