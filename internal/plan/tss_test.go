// internal/plan/tss_test.go
package plan

import (
	"math"
	"testing"

	"github.com/minicodemonkey/flywheel/internal/spec"
)

func TestEstimateTSS(t *testing.T) {
	// One hour steady at 100% FTP == 100 TSS by definition.
	c := spec.Course{Segments: []spec.Segment{{
		Intervals: []spec.Interval{{Duration: 3600, Intensity: spec.IntensityValue{From: 100, To: 100}}},
	}}}
	if got := EstimateTSS(c); math.Abs(got-100) > 0.01 {
		t.Fatalf("TSS = %f, want 100", got)
	}
}

func TestEstimateTSSRampUsesMidpoint(t *testing.T) {
	// 3600s at ramp [50,150] → IF=1.0 → 100 TSS.
	c := spec.Course{Segments: []spec.Segment{{
		Intervals: []spec.Interval{{Duration: 3600, Intensity: spec.IntensityValue{From: 50, To: 150}}},
	}}}
	if got := EstimateTSS(c); math.Abs(got-100) > 0.01 {
		t.Fatalf("TSS = %f, want 100", got)
	}
}
