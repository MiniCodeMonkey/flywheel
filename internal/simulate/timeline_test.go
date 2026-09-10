package simulate

import "testing"

func trk(idx, dur int, secs ...Section) Track {
	return Track{Index: idx, DurationSec: dur, Sections: secs}
}

func TestTrackStartsAccountForCrossfade(t *testing.T) {
	tl := Build(nil, []Track{trk(1, 200), trk(2, 180), trk(3, 160)}, 9)
	want := []int{0, 191, 362} // each track starts 9s before the previous ends
	for i, w := range want {
		if tl.TrackStart[i] != w {
			t.Errorf("track %d starts at %d, want %d", i+1, tl.TrackStart[i], w)
		}
	}
}

func TestSectionEdgesLandInsideTheirTrack(t *testing.T) {
	tl := Build(nil, []Track{
		trk(1, 100, Section{Duration: 40, Loudness: -10}, Section{Duration: 60, Loudness: -4}),
		trk(2, 100, Section{Duration: 50, Loudness: -8}, Section{Duration: 50, Loudness: -3}),
	}, 9)
	// track 1 starts at 0 with an edge at 40; track 2 starts at 91 with one at 141
	for _, want := range []int{40, 141} {
		found := false
		for _, e := range tl.SectionEdges {
			if e == want {
				found = true
			}
		}
		if !found {
			t.Errorf("expected a section edge at %ds, got %v", want, tl.SectionEdges)
		}
	}
}

func TestSectionEdgesAreSortedAndUnique(t *testing.T) {
	tl := Build(nil, []Track{
		trk(1, 100, Section{Duration: 50, Loudness: -6}, Section{Duration: 50, Loudness: -3}),
		trk(2, 100, Section{Duration: 50, Loudness: -6}, Section{Duration: 50, Loudness: -3}),
	}, 9)
	for i := 1; i < len(tl.SectionEdges); i++ {
		if tl.SectionEdges[i] <= tl.SectionEdges[i-1] {
			t.Fatalf("section edges not strictly increasing: %v", tl.SectionEdges)
		}
	}
}

func TestLoudnessRankIsPerTrack(t *testing.T) {
	// the quiet section of a loud track still ranks 0 within its own track
	tl := Build(nil, []Track{
		trk(1, 100, Section{Duration: 50, Loudness: -20}, Section{Duration: 50, Loudness: -18}),
	}, 0)
	if got := tl.RankAt(10); got != 0 {
		t.Errorf("rank at 10s = %v, want 0", got)
	}
	if got := tl.RankAt(70); got != 1 {
		t.Errorf("rank at 70s = %v, want 1", got)
	}
}
