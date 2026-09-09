package spec

import "fmt"

// maxSegmentDriftSec bounds how far one segment's intervals may run from the
// tracks it claims. MOWL lines segments up by elapsed time and never sees
// `tracks:`, so a segment is free to start or end mid-track -- an active
// recovery of 45s inside a 4-minute song, say. The drift limit still catches
// an authoring mistake; the course total below is what must hold exactly.
const maxSegmentDriftSec = 120

type TrackInfo struct {
	DurationSec int
	Title       string
}

func Validate(c Course, tracks map[int]TrackInfo, segTypes, positions map[string]int, tolSec int) []error {
	var errs []error
	seen := map[int]int{} // track index → count of segments claiming it
	courseIntervalSec, courseTrackSec := 0, 0

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
		if diff := sum - total; diff < -maxSegmentDriftSec || diff > maxSegmentDriftSec {
			errs = append(errs, fmt.Errorf("segment %q: intervals sum to %ds but its tracks are %ds (drift limit %ds)", seg.Name, sum, total, maxSegmentDriftSec))
		}
		courseIntervalSec += sum
		courseTrackSec += total
	}
	if diff := courseIntervalSec - courseTrackSec; diff < -tolSec || diff > tolSec {
		errs = append(errs, fmt.Errorf("course intervals sum to %ds but the playlist is %ds (±%ds)", courseIntervalSec, courseTrackSec, tolSec))
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
