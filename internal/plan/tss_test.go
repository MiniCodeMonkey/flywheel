package plan

import (
	"math"
	"testing"

	"github.com/minicodemonkey/flywheel/internal/spec"
)

func TestCogganZoneBoundaries(t *testing.T) {
	cases := map[int]int{40: 1, 55: 1, 56: 2, 75: 2, 80: 3, 90: 3, 100: 4, 105: 4, 115: 5, 140: 6, 160: 7}
	for ftp, want := range cases {
		if got := CogganZone(ftp); got != want {
			t.Errorf("CogganZone(%d) = %d, want %d", ftp, got, want)
		}
	}
}

func TestEstimateTSSUsesZoneIF(t *testing.T) {
	// 1 hour at 100% FTP -> Zone 4 -> IF 0.98 -> 96.04 TSS (matches MOWL).
	c := spec.Course{Segments: []spec.Segment{{
		Intervals: []spec.Interval{{Duration: 3600, Intensity: spec.IntensityValue{From: 100, To: 100}}},
	}}}
	if got := EstimateTSS(c); math.Abs(got-96.04) > 0.1 {
		t.Fatalf("TSS = %f, want ~96.04 (zone 4)", got)
	}
}

func TestEstimateTSSRampUsesMidpointZone(t *testing.T) {
	// Ramp [50,150] -> avg 100 -> Zone 4 -> 96.04.
	c := spec.Course{Segments: []spec.Segment{{
		Intervals: []spec.Interval{{Duration: 3600, Intensity: spec.IntensityValue{From: 50, To: 150}}},
	}}}
	if got := EstimateTSS(c); math.Abs(got-96.04) > 0.1 {
		t.Fatalf("TSS = %f, want ~96.04", got)
	}
}

func TestEstimateTSSZone1Recovery(t *testing.T) {
	// 1 hour at 45% FTP -> Zone 1 -> IF 0.2775 -> ~7.7 TSS.
	c := spec.Course{Segments: []spec.Segment{{
		Intervals: []spec.Interval{{Duration: 3600, Intensity: spec.IntensityValue{From: 45, To: 45}}},
	}}}
	if got := EstimateTSS(c); math.Abs(got-7.70) > 0.1 {
		t.Fatalf("TSS = %f, want ~7.70 (zone 1)", got)
	}
}

func TestEstimateTSSNormalizedPower(t *testing.T) {
	// z2(1800s @ 65% -> IF 0.6575) + z3(1800s @ 83% -> IF 0.83).
	// NP: IF_np = ((1800*0.6575^4 + 1800*0.83^4)/3600)^0.25 -> TSS ~57.5.
	// Verified against the live MOWL API (server returned 57.57).
	c := spec.Course{Segments: []spec.Segment{{
		Intervals: []spec.Interval{
			{Duration: 1800, Intensity: spec.IntensityValue{From: 65, To: 65}},
			{Duration: 1800, Intensity: spec.IntensityValue{From: 83, To: 83}},
		},
	}}}
	if got := EstimateTSS(c); math.Abs(got-57.5) > 0.3 {
		t.Fatalf("TSS = %f, want ~57.5 (NP-weighted)", got)
	}
}
