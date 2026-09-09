package spec

import "testing"

func TestCrossfadeDefaultsToTen(t *testing.T) {
	if got := (Playlist{}).Crossfade(); got != DefaultCrossfadeSec {
		t.Fatalf("Crossfade() = %d, want %d", got, DefaultCrossfadeSec)
	}
}

func TestCrossfadeHonoursExplicitZero(t *testing.T) {
	zero := 0
	if got := (Playlist{CrossfadeSec: &zero}).Crossfade(); got != 0 {
		t.Fatalf("Crossfade() = %d, want 0", got)
	}
}

func TestCrossfadeTracksShortensAllButTheLast(t *testing.T) {
	in := map[int]TrackInfo{1: {DurationSec: 200}, 2: {DurationSec: 180}, 3: {DurationSec: 160}}
	got := CrossfadeTracks(in, 10)
	for idx, want := range map[int]int{1: 190, 2: 170, 3: 160} {
		if got[idx].DurationSec != want {
			t.Errorf("track %d = %ds, want %ds", idx, got[idx].DurationSec, want)
		}
	}
	if in[1].DurationSec != 200 {
		t.Error("CrossfadeTracks mutated its input")
	}
}
