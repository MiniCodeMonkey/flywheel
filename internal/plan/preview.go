// internal/plan/preview.go
package plan

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/minicodemonkey/flywheel/internal/spec"
)

type SegPreview struct {
	Name          string  `json:"name"`
	DurationSec   int     `json:"duration_sec"`
	TSS           float64 `json:"tss"`
	IntervalCount int     `json:"interval_count"`
}

type Preview struct {
	TotalSec  int          `json:"total_sec"`
	EstTSS    float64      `json:"est_tss"`
	Segments  []SegPreview `json:"segments"`
	Errors    []string     `json:"errors,omitempty"`
	StyleNote string       `json:"style_note,omitempty"`
}

func BuildPreview(c spec.Course, tracks map[int]spec.TrackInfo, styles spec.Styles, errs []error) Preview {
	p := Preview{StyleNote: styles.Describe(c.Style)}
	for _, s := range c.Segments {
		dur := 0
		for _, iv := range s.Intervals {
			dur += iv.Duration
		}
		p.Segments = append(p.Segments, SegPreview{
			Name: s.Name, DurationSec: dur, TSS: SegmentTSS(s), IntervalCount: len(s.Intervals),
		})
		p.TotalSec += dur
	}
	p.EstTSS = EstimateTSS(c)
	for _, e := range errs {
		p.Errors = append(p.Errors, e.Error())
	}
	return p
}

func mmss(sec int) string { return fmt.Sprintf("%d:%02d", sec/60, sec%60) }

func (p Preview) Text(t spec.Targets) string {
	var b strings.Builder
	for _, s := range p.Segments {
		fmt.Fprintf(&b, "  %-16s %6s   TSS %5.1f   (%d intervals)\n", s.Name, mmss(s.DurationSec), s.TSS, s.IntervalCount)
	}
	fmt.Fprintf(&b, "  %-16s %6s   TSS %5.1f\n", "TOTAL", mmss(p.TotalSec), p.EstTSS)
	fmt.Fprintf(&b, "  target:          %6s   TSS %5d\n", mmss(t.DurationMin*60), t.TSS)
	if p.StyleNote != "" {
		fmt.Fprintf(&b, "  %s\n", p.StyleNote)
	}
	for _, e := range p.Errors {
		fmt.Fprintf(&b, "  ERROR: %s\n", e)
	}
	return b.String()
}

func (p Preview) JSON() ([]byte, error) { return json.MarshalIndent(p, "", "  ") }
