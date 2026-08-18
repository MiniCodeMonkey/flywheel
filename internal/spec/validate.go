package spec

import "fmt"

type TrackInfo struct {
	DurationSec int
	Title       string
}

func Validate(c Course, tracks map[int]TrackInfo, segTypes, positions map[string]int, tolSec int) []error {
	var errs []error
	seen := map[int]int{} // track index → count of segments claiming it

	for _, seg := range c.Segments {
		if _, ok := segTypes[seg.Type]; !ok {
			errs = append(errs, fmt.Errorf("segment %q: unknown type %q", seg.Name, seg.Type))
		}
		total := 0
		for _, idx := range seg.Tracks {
			seen[idx]++
			ti, ok := tracks[idx]
			if !ok {
				errs = append(errs, fmt.Errorf("segment %q: track %d not in playlist", seg.Name, idx))
				continue
			}
			total += ti.DurationSec
		}
		sum := 0
		for _, iv := range seg.Intervals {
			sum += iv.Duration
			if _, ok := positions[iv.Position]; !ok {
				errs = append(errs, fmt.Errorf("segment %q: unknown position %q", seg.Name, iv.Position))
			}
		}
		if diff := sum - total; diff < -tolSec || diff > tolSec {
			errs = append(errs, fmt.Errorf("segment %q: intervals sum to %ds but its tracks are %ds (±%ds)", seg.Name, sum, total, tolSec))
		}
	}
	for idx := range tracks {
		if seen[idx] == 0 {
			errs = append(errs, fmt.Errorf("track %d is not covered by any segment", idx))
		}
		if seen[idx] > 1 {
			errs = append(errs, fmt.Errorf("track %d is claimed by %d segments", idx, seen[idx]))
		}
	}
	return errs
}
