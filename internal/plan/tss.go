package plan

import "github.com/minicodemonkey/flywheel/internal/spec"

func intervalTSS(iv spec.Interval) float64 {
	hours := float64(iv.Duration) / 3600.0
	iff := float64(iv.Intensity.From+iv.Intensity.To) / 2.0 / 100.0
	return hours * iff * iff * 100.0
}

func SegmentTSS(s spec.Segment) float64 {
	var t float64
	for _, iv := range s.Intervals {
		t += intervalTSS(iv)
	}
	return t
}

func EstimateTSS(c spec.Course) float64 {
	var t float64
	for _, s := range c.Segments {
		t += SegmentTSS(s)
	}
	return t
}
