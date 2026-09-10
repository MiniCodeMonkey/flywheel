// Package simulate walks a course against the music it will actually play,
// second by second, and reports what a static read of the spec cannot see:
// whether interval boundaries land on the song's own boundaries, whether the
// hard part lands on the loud part, and whether the ride is rideable in
// sequence. Everything here is pure computation -- same input, same output.
package simulate

import (
	"math"
	"sort"

	"github.com/minicodemonkey/flywheel/internal/spec"
)

// Section is one musical section of a track.
type Section struct {
	Duration float64
	Loudness float64
}

// Track is a playlist track with the sections the audio analysis found.
type Track struct {
	Index       int
	Title       string
	Artist      string
	BPM         int
	DurationSec int
	Sections    []Section
}

// Block is one interval placed on the ride's clock.
type Block struct {
	Start, End  int
	Zone        int
	CadenceFrom int
	CadenceTo   int
	Position    string
	Cycle       string
	Segment     string
	SegmentType string
}

// Timeline places the ride and the music on one clock.
type Timeline struct {
	Blocks       []Block
	TrackStart   []int // absolute start of each track
	Tracks       []Track
	SectionEdges []int // absolute section boundaries, sorted and unique
	Crossfade    int

	rank []float64 // loudness rank within its own track, per second of music
}

var bands = [][2]int{{0, 55}, {56, 75}, {76, 90}, {91, 105}, {106, 120}, {121, 150}, {151, 200}}

// zoneOf maps an interval's %FTP band to its Coggan zone.
func zoneOf(iv spec.Interval) int {
	for i, b := range bands {
		if iv.Intensity.From == b[0] {
			return i + 1
		}
	}
	return plan(iv.Intensity.From)
}

func plan(ftp int) int {
	switch {
	case ftp <= 55:
		return 1
	case ftp <= 75:
		return 2
	case ftp <= 90:
		return 3
	case ftp <= 105:
		return 4
	case ftp <= 120:
		return 5
	case ftp <= 150:
		return 6
	}
	return 7
}

// Build places every interval and every musical section on the same clock.
// A track starts `crossfade` seconds before the previous one ends, which is
// what MOWL's own timeline does.
func Build(c *spec.Course, tracks []Track, crossfade int) *Timeline {
	tl := &Timeline{Tracks: tracks, Crossfade: crossfade}

	run := 0
	for k, t := range tracks {
		tl.TrackStart = append(tl.TrackStart, run-crossfade*k)
		run += t.DurationSec
	}

	edges := map[int]bool{}
	musicEnd := 0
	if len(tracks) > 0 {
		musicEnd = tl.TrackStart[len(tracks)-1] + tracks[len(tracks)-1].DurationSec
	}
	tl.rank = make([]float64, musicEnd+1)
	for k, t := range tracks {
		lo, hi := math.MaxFloat64, -math.MaxFloat64
		for _, s := range t.Sections {
			lo = math.Min(lo, s.Loudness)
			hi = math.Max(hi, s.Loudness)
		}
		span := hi - lo
		if span <= 0 {
			span = 1
		}
		// a track owns the clock until the next one starts
		until := musicEnd
		if k+1 < len(tracks) {
			until = tl.TrackStart[k+1]
		}
		// the start of a song is a musical boundary, and the strongest one
		if k > 0 {
			edges[tl.TrackStart[k]] = true
		}
		at := float64(tl.TrackStart[k])
		for i, s := range t.Sections {
			if i > 0 && int(at) < until && int(at) > tl.TrackStart[k] {
				edges[int(at)] = true
			}
			r := (s.Loudness - lo) / span
			for sec := int(at); sec < int(at+s.Duration) && sec < len(tl.rank); sec++ {
				if sec >= 0 {
					tl.rank[sec] = r
				}
			}
			at += s.Duration
		}
	}
	for e := range edges {
		tl.SectionEdges = append(tl.SectionEdges, e)
	}
	sort.Ints(tl.SectionEdges) // map order is randomised; output must not be

	if c != nil {
		at := 0
		for _, sg := range c.Segments {
			for _, iv := range sg.Intervals {
				tl.Blocks = append(tl.Blocks, Block{
					Start: at, End: at + iv.Duration,
					Zone:        zoneOf(iv),
					CadenceFrom: iv.Cadence[0], CadenceTo: iv.Cadence[1],
					Position: iv.Position, Cycle: iv.Cycle,
					Segment: sg.Name, SegmentType: sg.Type,
				})
				at += iv.Duration
			}
		}
	}
	return tl
}

// RankAt is how loud the music is at that second, relative to the rest of its
// own track: 0 is that track's quietest section, 1 its loudest.
func (t *Timeline) RankAt(sec int) float64 {
	if sec < 0 || sec >= len(t.rank) {
		return 0
	}
	return t.rank[sec]
}

// Duration is the length of the ride itself, which may be shorter than the
// music when the playlist carries a cooldown tail.
func (t *Timeline) Duration() int {
	if len(t.Blocks) == 0 {
		return 0
	}
	return t.Blocks[len(t.Blocks)-1].End
}

// TrackAt returns the index into Tracks playing at that second, or -1.
func (t *Timeline) TrackAt(sec int) int {
	idx := -1
	for k, s := range t.TrackStart {
		if sec >= s {
			idx = k
		}
	}
	return idx
}
