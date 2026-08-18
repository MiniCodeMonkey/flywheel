// internal/plan/preview_test.go
package plan

import (
	"strings"
	"testing"

	"github.com/minicodemonkey/flywheel/internal/spec"
)

func TestBuildPreviewTotalsAndText(t *testing.T) {
	c := spec.Course{
		Style: []string{"road_cycling"},
		Segments: []spec.Segment{{
			Name: "Main",
			Intervals: []spec.Interval{
				{Duration: 1800, Intensity: spec.IntensityValue{From: 100, To: 100}},
			},
		}},
	}
	p := BuildPreview(c, nil, spec.Styles{}, nil)
	if p.TotalSec != 1800 {
		t.Fatalf("total = %d", p.TotalSec)
	}
	if p.EstTSS <= 0 {
		t.Fatalf("tss = %f", p.EstTSS)
	}
	out := p.Text(spec.Targets{DurationMin: 30, TSS: 50})
	if !strings.Contains(out, "Main") || !strings.Contains(out, "TSS") {
		t.Fatalf("text missing content:\n%s", out)
	}
}
