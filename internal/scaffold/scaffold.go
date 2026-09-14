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
	Crossfade  int  // seconds each track overlaps the next (see spec.Crossfade)

	// MaxStanding bounds one unbroken standing run. Real MOWL rides stand for
	// about 30s at a time and rarely past 90s: you cannot hold a long one.
	MaxStanding int
	// StandingCadenceMin and StandingCadenceMax bound cadence while out of the
	// saddle. MOWL's own rides stand between 60 and the high 70s: below that
	// is a grind, above it you cannot hold the position.
	StandingCadenceMin int
	StandingCadenceMax int
	// ACCShare is roughly the fraction of work intervals marked as
	// acceleration bursts.
	ACCShare float64
}

// Defaults returns the option set used when flags are left alone.
func Defaults() Options {
	return Options{TargetTSS: 75, MinSection: 13, MaxSection: 180, EndHot: true,
		Crossfade: spec.DefaultCrossfadeSec, MaxStanding: 60,
		StandingCadenceMin: 60, StandingCadenceMax: 78, ACCShare: 0.10}
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
	// Work sits between blue and yellow. Red and fire are accents the segment
	// ending places deliberately: MOWL's own rides spend about 11% of their
	// time at zone 5 or above, and letting the body reach red pushes the
	// loudness curve to extremes, hollowing out the green and yellow middle.
	return 1, 3
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
	for _, div := range []float64{1, 2, 4} {
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

// mergeIdentical collapses neighbouring intervals that ask for exactly the
// same thing. A split the rider cannot act on reads as a cue to change
// something and then turns out not to be one, so the only splits worth keeping
// are the ones where something actually changes.
func mergeIdentical(ivs []spec.Interval) []spec.Interval {
	out := ivs[:0:0]
	for _, iv := range ivs {
		if n := len(out); n > 0 {
			p := &out[n-1]
			if p.Cadence == iv.Cadence && p.Intensity == iv.Intensity &&
				p.Position == iv.Position && p.Cycle == iv.Cycle {
				p.Duration += iv.Duration
				continue
			}
		}
		out = append(out, iv)
	}
	return out
}

// canStand reports whether a track's cadence is one a rider can hold out of
// the saddle. A song does not change tempo, so its cadence is fixed for the
// whole track: when that cadence is too fast or too slow to stand at, the
// answer is to stay seated for this song, never to invent a second cadence
// inside it.
func canStand(cadence int, o Options) bool {
	return cadence >= o.StandingCadenceMin && cadence <= o.StandingCadenceMax
}

// breakStanding sits the rider back down once a standing run reaches the cap.
// The final interval is left alone: it carries the segment's ending.
func breakStanding(ivs []spec.Interval, max int) []spec.Interval {
	run := 0
	for i := 0; i < len(ivs)-1; i++ {
		if ivs[i].Position != "standing" {
			run = 0
			continue
		}
		if run+ivs[i].Duration > max {
			ivs[i].Position = "seated"
			run = 0
			continue
		}
		run += ivs[i].Duration
	}
	// The segment's ending stays standing, so trim back into it: sit the rider
	// down before the finish rather than let the run overshoot through it.
	last := len(ivs) - 1
	if last >= 0 && ivs[last].Position == "standing" && ivs[last].Duration > max {
		// a single standing block longer than the cap: ride the front of it
		// seated and stand only for the finish
		head := ivs[last]
		head.Duration -= max
		head.Position = "seated"
		ivs[last].Duration = max
		ivs = append(ivs[:last], head, ivs[last])
		last = len(ivs) - 1
	}
	if last >= 0 && ivs[last].Position == "standing" {
		tail := ivs[last].Duration
		for i := last - 1; i >= 0 && ivs[i].Position == "standing"; i-- {
			if tail+ivs[i].Duration > max {
				ivs[i].Position = "seated"
				break
			}
			tail += ivs[i].Duration
		}
	}
	return ivs
}

// markACC turns a track's loudest short section into an acceleration burst:
// same gear, higher RPM. ACC overrules RPM, so the cadence is cleared.
func markACC(ivs []spec.Interval, rk []float64, share float64) []spec.Interval {
	if share <= 0 || len(ivs) == 0 {
		return ivs
	}
	want := int(math.Round(share * float64(len(ivs))))
	if want < 1 {
		want = 1
	}
	type cand struct {
		i int
		r float64
	}
	var cands []cand
	for i, iv := range ivs {
		if iv.Duration < 15 || iv.Duration > 60 || iv.Position != "seated" {
			continue
		}
		r := 0.0
		if i < len(rk) {
			r = rk[i]
		}
		cands = append(cands, cand{i, r})
	}
	sort.Slice(cands, func(a, b int) bool { return cands[a].r > cands[b].r })
	if len(cands) > want {
		cands = cands[:want]
	}
	for _, c := range cands {
		best := c.i
		// ACC is "short bursts in the same gear, but with higher RPM": it is an
		// acceleration, not a power step. Promoting its zone made every burst
		// cost TSS, which forced the loudness curve flat to compensate.
		ivs[best].Cycle = "acc"
		ivs[best].Cadence = [2]int{0, 0}
	}
	return ivs
}

// build lays out the whole course at one gamma. Raising gamma pushes more
// sections toward the bottom of the band while leaving the loudest at the top,
// which lowers overall TSS without flattening the ride.
func build(tracks map[int]Track, segs []SegmentSpec, o Options, gamma float64) spec.Course {
	c := spec.Course{Activity: "cycling"}
	// the finale is the last work segment; MOWL ends that one hot and lets the
	// earlier ones finish wherever the music does ([2,5,6], [4,6,6])
	finale := -1
	for i, sg := range segs {
		if sg.Type != "recovery" && sg.Type != "cooldown" {
			finale = i
		}
	}
	for si, sg := range segs {
		lo, hi := bandFor(sg.Type)
		out := spec.Segment{Name: sg.Name, Type: sg.Type, Tracks: sg.Tracks}
		if sg.Type == "recovery" || sg.Type == "cooldown" {
			// MOWL models both as a single soft interval, not a block of them
			total, bpm := 0, 0
			for _, ti := range sg.Tracks {
				total += tracks[ti].DurationSec
				if bpm == 0 {
					bpm = tracks[ti].BPM
				}
			}
			c1, c2 := cadenceFor(bpm, sg.Type)
			out.Intervals = []spec.Interval{{
				Duration:  total,
				Cadence:   [2]int{c1, c2},
				Intensity: spec.IntensityValue{From: ladder[0][0], To: ladder[0][1]},
				Position:  "seated",
			}}
			c.Segments = append(c.Segments, out)
			continue
		}
		for _, ti := range sg.Tracks {
			t := tracks[ti]
			secs := sectionsOf(t, o)
			rk := rank(secs)
			c1, c2 := cadenceFor(t.BPM, sg.Type)
			var ivs []spec.Interval
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
					if canStand(c1, o) {
						pos = "standing"
					}
				}
				cad := [2]int{c1, c2}
				ivs = append(ivs, spec.Interval{
					Duration:  int(s.Duration),
					Cadence:   cad,
					Intensity: spec.IntensityValue{From: band[0], To: band[1]},
					Position:  pos,
				})
			}
			ivs = markACC(mergeIdentical(ivs), rk, o.ACCShare)
			out.Intervals = append(out.Intervals, ivs...)
		}
		if o.EndHot && len(out.Intervals) > 0 && sg.Type != "recovery" && sg.Type != "cooldown" {
			top := ladder[len(ladder)-1] // fire
			if sg.Type == "warmup" {
				top = ladder[4] // red
			}
			// Put the accent on the loudest of the closing intervals rather
			// than blindly the last one: a track that fades out would other-
			// wise be ridden hardest over its fade.
			n := len(out.Intervals)
			hot := n - 1
			for i := n - 1; i >= 0 && i >= n-3; i-- {
				if out.Intervals[i].Intensity.From > out.Intervals[hot].Intensity.From {
					hot = i
				}
			}
			if hot != n-1 && si == finale {
				// only the finale has to finish hard; forcing it on every
				// segment lands red on whatever fade happens to be there
				fin := &out.Intervals[n-1]
				red := ladder[4]
				if fin.Intensity.From < red[0] {
					fin.Intensity = spec.IntensityValue{From: red[0], To: red[1]}
				}
			}
			last := &out.Intervals[hot]
			last.Intensity = spec.IntensityValue{From: top[0], To: top[1]}
			if last.Cycle == "acc" || last.Cadence[0] == 0 {
				last.Cycle = "" // ACC overrules RPM, so a promoted burst needs a cadence back
				for i := len(out.Intervals) - 2; i >= 0; i-- {
					if c := out.Intervals[i].Cadence; c[0] > 0 {
						last.Cadence = c // this track's own cadence, not an invented one
						break
					}
				}
			}
			if canStand(last.Cadence[0], o) {
				last.Position = "standing"
			}
		}
		out.Intervals = breakStanding(out.Intervals, o.MaxStanding)
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
	tracks = trimCrossfade(tracks, o.Crossfade)
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

// trimCrossfade drops the trailing crossfade seconds from every track that has
// another track after it. Spotify starts the next track that early, so those
// seconds are not part of the ride's timeline and no interval may sit in them.
func trimCrossfade(tracks []Track, crossfade int) []Track {
	if crossfade <= 0 {
		return tracks
	}
	last := 0
	for _, t := range tracks {
		if t.Index > last {
			last = t.Index
		}
	}
	out := make([]Track, 0, len(tracks))
	for _, t := range tracks {
		if t.Index == last {
			out = append(out, t)
			continue
		}
		t.DurationSec -= crossfade
		left := float64(crossfade)
		secs := append([]Section(nil), t.Sections...)
		for len(secs) > 0 && left > 0 {
			tail := &secs[len(secs)-1]
			if tail.Duration > left {
				tail.Duration -= left
				left = 0
				break
			}
			left -= tail.Duration
			secs = secs[:len(secs)-1]
		}
		t.Sections = secs
		out = append(out, t)
	}
	return out
}

// ParseSegment reads a "Name:type:1-3,5" segment flag.
func ParseSegment(s string) (SegmentSpec, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return SegmentSpec{}, fmt.Errorf("segment %q: want Name:type:tracks (e.g. \"Warmup:warmup:1-3\")", s)
	}
	out := SegmentSpec{Name: strings.TrimSpace(parts[0]), Type: strings.TrimSpace(parts[1])}
	// MOWL leaves its active-recovery and cooldown segments unnamed; work
	// segments are the ones the rider sees called out.
	if out.Name == "" && out.Type != "recovery" && out.Type != "cooldown" {
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
