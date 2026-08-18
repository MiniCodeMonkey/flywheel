package plan

import (
	"math"

	"github.com/minicodemonkey/flywheel/internal/spec"
)

// MOWL computes program TSS with a Normalized-Power model over the whole set of
// intervals: each interval's intensity factor comes from its Coggan zone
// (derived from its average %FTP), the workout's normalized IF is the
// 4th-power (quadratic-mean-of-squares) weighted average of those, and
//
//	TSS = totalHours × IF_np² × 100
//
// Because NP weights higher intensities more heavily, a variable ride scores
// above the simple duration-weighted average — which is why this matches the
// live server value where a linear sum under-reads. Verified against the API.

func intervalIF(iv spec.Interval) float64 {
	avgFTP := (iv.Intensity.From + iv.Intensity.To) / 2
	return ZoneIF(CogganZone(avgFTP))
}

// tssForIntervals applies the NP model to one set of intervals (used both for a
// single segment and, flattened, for the whole course).
func tssForIntervals(ivs []spec.Interval) float64 {
	var totalSec int
	var sumDurIF4 float64
	for _, iv := range ivs {
		if iv.Duration <= 0 {
			continue
		}
		iff := intervalIF(iv)
		totalSec += iv.Duration
		sumDurIF4 += float64(iv.Duration) * math.Pow(iff, 4)
	}
	if totalSec == 0 {
		return 0
	}
	ifNP := math.Pow(sumDurIF4/float64(totalSec), 0.25)
	hours := float64(totalSec) / 3600.0
	return hours * ifNP * ifNP * 100.0
}

// SegmentTSS estimates the TSS of one segment on its own (its intervals'
// normalized IF). Note that per-segment TSS values do not sum to the course
// total, because normalized power is non-linear — EstimateTSS is authoritative.
func SegmentTSS(s spec.Segment) float64 {
	return tssForIntervals(s.Intervals)
}

// EstimateTSS estimates the whole course's TSS with the NP model across every
// interval, matching MOWL's server-computed value.
func EstimateTSS(c spec.Course) float64 {
	var all []spec.Interval
	for _, s := range c.Segments {
		all = append(all, s.Intervals...)
	}
	return tssForIntervals(all)
}
