// Package scaffold turns an inspected playlist into a course spec whose
// intervals follow the music: one interval per musical section, intensity
// driven by how loud that section is relative to the rest of its own track.
package scaffold

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/minicodemonkey/flywheel/internal/mowl"
	"github.com/minicodemonkey/flywheel/internal/plan"
	"github.com/minicodemonkey/flywheel/internal/spec"
)

// Section is one musical section of a track, as reported by the
// audio-analysis endpoint.
type Section struct {
	Duration float64
	Loudness float64
}

// Track is the subset of an inspected track that scaffolding needs.
type Track struct {
	Index       int
	Title       string
	BPM         int
	DurationSec int
	Sections    []Section
}

// SegmentSpec names one segment of the ride and the playlist tracks it spans.
type SegmentSpec struct {
	Name   string
	Type   string
	Tracks []int
}

// Options tune how sections become intervals.
type Options struct {
	TargetTSS  int
	MinSection int  // sections shorter than this fold into their neighbour
	MaxSection int  // sections longer than this split into equal parts
	EndHot     bool // work segments end on their hardest zone
}

// Defaults returns the option set used when flags are left alone.
func Defaults() Options {
	return Options{TargetTSS: 75, MinSection: 13, MaxSection: 180, EndHot: true}
}

// zone bands, low to high, as [from,to] %FTP; index is the ladder position
var ladder = [][2]int{{0, 55}, {56, 75}, {76, 90}, {91, 105}, {106, 120}, {121, 150}}

// bandFor returns the ladder range a segment role is allowed to use. Recovery
// and cooldown stay soft; work segments top out at red so that zone 6, which
// dominates TSS, is spent only on the deliberate segment-ending accent.
func bandFor(segType string) (lo, hi int) {
	switch segType {
	case "warmup":
		return 0, 3
	case "recovery", "cooldown":
		return 0, 1
	}
	return 1, 4
}

// cadenceFor picks one cadence for a whole track by dividing its BPM until it
// lands in a rideable range. Varying cadence per interval reads as noise, so
// the whole track rides a single narrow range.
func cadenceFor(bpm int, segType string) (int, int) {
	if bpm <= 0 {
		return 85, 86
	}
	target := 90.0
	if segType == "climb" {
		target = 65.0
	}
	best, bestErr := 0.0, math.MaxFloat64
	for _, div := range []float64{1, 1.5, 2, 3, 4} {
		v := float64(bpm) / div
		if v < 55 || v > 110 {
			continue
		}
		if e := math.Abs(v - target); e < bestErr {
			best, bestErr = v, e
		}
	}
	if best == 0 {
		best = 85
	}
	n := int(math.Round(best))
	return n, n + 1
}

// sectionsOf normalises a track's sections: fold anything shorter than
// MinSection into its neighbour carrying loudness as the duration-weighted
// mean, split anything longer than MaxSection, then absorb rounding drift so
// the durations sum to the track's real length (course validation requires it).
func sectionsOf(t Track, o Options) []Section {
	var out []Section
	for _, s := range t.Sections {
		d := math.Round(s.Duration)
		if len(out) > 0 && int(d) < o.MinSection {
			p := &out[len(out)-1]
			p.Loudness = (p.Loudness*p.Duration + s.Loudness*d) / (p.Duration + d)
			p.Duration += d
			continue
		}
		out = append(out, Section{Duration: d, Loudness: s.Loudness})
	}
	for len(out) > 1 && int(out[len(out)-1].Duration) < o.MinSection {
		last := out[len(out)-1]
		out = out[:len(out)-1]
		p := &out[len(out)-1]
		p.Loudness = (p.Loudness*p.Duration + last.Loudness*last.Duration) / (p.Duration + last.Duration)
		p.Duration += last.Duration
	}
	if o.MaxSection > 0 {
		var split []Section
		for _, s := range out {
			n := int(math.Ceil(s.Duration / float64(o.MaxSection)))
			if n < 1 {
				n = 1
			}
			each := s.Duration / float64(n)
			for i := 0; i < n; i++ {
				split = append(split, Section{Duration: math.Round(each), Loudness: s.Loudness})
			}
		}
		out = split
	}
	if len(out) == 0 {
		return []Section{{Duration: float64(t.DurationSec)}}
	}
	total := 0.0
	for _, s := range out {
		total += s.Duration
	}
	out[len(out)-1].Duration += float64(t.DurationSec) - total
	return out
}

// rank returns each section's loudness position within its own track, 0 for
// the quietest and 1 for the loudest. This is what separates a chorus from a
// verse without needing to know anything about the song.
func rank(secs []Section) []float64 {
	lo, hi := math.MaxFloat64, -math.MaxFloat64
	for _, s := range secs {
		lo = math.Min(lo, s.Loudness)
		hi = math.Max(hi, s.Loudness)
	}
	span := hi - lo
	if span <= 0 {
		span = 1
	}
	out := make([]float64, len(secs))
	for i, s := range secs {
		out[i] = (s.Loudness - lo) / span
	}
	return out
}

// build lays out the whole course at one gamma. Raising gamma pushes more
// sections toward the bottom of the band while leaving the loudest at the top,
// which lowers overall TSS without flattening the ride.
func build(tracks map[int]Track, segs []SegmentSpec, o Options, gamma float64) spec.Course {
	c := spec.Course{Activity: "cycling"}
	for _, sg := range segs {
		lo, hi := bandFor(sg.Type)
		out := spec.Segment{Name: sg.Name, Type: sg.Type, Tracks: sg.Tracks}
		for _, ti := range sg.Tracks {
			t := tracks[ti]
			secs := sectionsOf(t, o)
			rk := rank(secs)
			c1, c2 := cadenceFor(t.BPM, sg.Type)
			for i, s := range secs {
				top := hi
				if sg.Type == "warmup" && len(secs) > 1 { // ease the ceiling up
					top = lo + int(math.Round(float64(hi-lo)*(0.4+0.6*float64(i)/float64(len(secs)-1))))
				}
				k := lo + int(math.Round(float64(top-lo)*math.Pow(rk[i], gamma)))
				k = int(math.Max(0, math.Min(float64(len(ladder)-1), float64(k))))
				band := ladder[k]
				pos := "seated"
				if (sg.Type == "climb" && rk[i] > 0.72) || (k >= 4 && rk[i] > 0.85) {
					pos = "standing"
				}
				out.Intervals = append(out.Intervals, spec.Interval{
					Duration:  int(s.Duration),
					Cadence:   [2]int{c1, c2},
					Intensity: spec.IntensityValue{From: band[0], To: band[1]},
					Position:  pos,
				})
			}
		}
		if o.EndHot && len(out.Intervals) > 0 && sg.Type != "recovery" && sg.Type != "cooldown" {
			top := ladder[len(ladder)-1] // fire
			if sg.Type == "warmup" {
				top = ladder[4] // red
			}
			last := &out.Intervals[len(out.Intervals)-1]
			last.Intensity = spec.IntensityValue{From: top[0], To: top[1]}
			last.Position = "standing"
		}
		c.Segments = append(c.Segments, out)
	}
	return c
}

// Build scaffolds a course and tunes gamma so the estimated TSS lands as close
// to the target as the music allows. It returns the course and the gamma used.
func Build(tracks []Track, segs []SegmentSpec, o Options) (spec.Course, float64, error) {
	if len(segs) == 0 {
		return spec.Course{}, 0, fmt.Errorf("no segments given")
	}
	byIndex := map[int]Track{}
	for _, t := range tracks {
		byIndex[t.Index] = t
	}
	for _, sg := range segs {
		for _, ti := range sg.Tracks {
			t, ok := byIndex[ti]
			if !ok {
				return spec.Course{}, 0, fmt.Errorf("segment %q: no track %d in playlist", sg.Name, ti)
			}
			if t.DurationSec <= 0 {
				return spec.Course{}, 0, fmt.Errorf("segment %q: track %d has no duration yet; "+
					"MOWL is still indexing the playlist", sg.Name, ti)
			}
		}
	}
	best, bestGamma, bestErr := spec.Course{}, 0.0, math.MaxFloat64
	for i := 10; i <= 160; i++ {
		g := float64(i) / 20
		c := build(byIndex, segs, o, g)
		if e := math.Abs(plan.EstimateTSS(c) - float64(o.TargetTSS)); e < bestErr {
			best, bestGamma, bestErr = c, g, e
		}
	}
	return best, bestGamma, nil
}

// ParseSegment reads a "Name:type:1-3,5" segment flag.
func ParseSegment(s string) (SegmentSpec, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return SegmentSpec{}, fmt.Errorf("segment %q: want Name:type:tracks (e.g. \"Warmup:warmup:1-3\")", s)
	}
	out := SegmentSpec{Name: strings.TrimSpace(parts[0]), Type: strings.TrimSpace(parts[1])}
	if out.Name == "" {
		return SegmentSpec{}, fmt.Errorf("segment %q: empty name", s)
	}
	if _, ok := mowl.SegmentTypeAlias[out.Type]; !ok {
		return SegmentSpec{}, fmt.Errorf("segment %q: unknown type %q", s, out.Type)
	}
	seen := map[int]bool{}
	for _, chunk := range strings.Split(parts[2], ",") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		var from, to int
		if strings.Contains(chunk, "-") {
			if _, err := fmt.Sscanf(chunk, "%d-%d", &from, &to); err != nil {
				return SegmentSpec{}, fmt.Errorf("segment %q: bad track range %q", s, chunk)
			}
		} else {
			if _, err := fmt.Sscanf(chunk, "%d", &from); err != nil {
				return SegmentSpec{}, fmt.Errorf("segment %q: bad track %q", s, chunk)
			}
			to = from
		}
		if from < 1 || to < from {
			return SegmentSpec{}, fmt.Errorf("segment %q: bad track range %q", s, chunk)
		}
		for i := from; i <= to; i++ {
			if !seen[i] {
				seen[i] = true
				out.Tracks = append(out.Tracks, i)
			}
		}
	}
	if len(out.Tracks) == 0 {
		return SegmentSpec{}, fmt.Errorf("segment %q: no tracks", s)
	}
	sort.Ints(out.Tracks)
	return out, nil
}
