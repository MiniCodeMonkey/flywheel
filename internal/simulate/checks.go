package simulate

import (
	"fmt"
	"math"
	"sort"
)

// Miss is one interval boundary that does not land on a musical boundary.
type Miss struct {
	At  int // seconds into the ride
	Off int // distance to the nearest section edge
}

// AlignmentReport says how well the ride's boundaries follow the song's.
type AlignmentReport struct {
	Edges           int
	WithinTolerance float64 // share of edges landing on a section boundary
	Median          int
	Worst           []Miss
}

// Alignment measures every interval boundary against the nearest musical
// section boundary. This is the premise the whole design method rests on: a
// ride whose edges land on the music reads as designed, one whose edges land
// on a stopwatch reads as generated.
func (t *Timeline) Alignment(tolerance int) AlignmentReport {
	var r AlignmentReport
	if len(t.SectionEdges) == 0 || len(t.Blocks) < 2 {
		return r
	}
	var offs []int
	var misses []Miss
	for _, b := range t.Blocks[1:] {
		off := nearest(t.SectionEdges, b.Start)
		offs = append(offs, off)
		if off <= tolerance {
			r.WithinTolerance++
		} else {
			misses = append(misses, Miss{At: b.Start, Off: off})
		}
	}
	r.Edges = len(offs)
	r.WithinTolerance /= float64(r.Edges)
	sort.Ints(offs)
	r.Median = offs[len(offs)/2]
	sort.Slice(misses, func(a, b int) bool {
		if misses[a].Off != misses[b].Off {
			return misses[a].Off > misses[b].Off
		}
		return misses[a].At < misses[b].At // stable: never depend on input order alone
	})
	if len(misses) > 8 {
		misses = misses[:8]
	}
	r.Worst = misses
	return r
}

func nearest(sorted []int, v int) int {
	i := sort.SearchInts(sorted, v)
	best := math.MaxInt32
	for _, j := range []int{i - 1, i} {
		if j >= 0 && j < len(sorted) {
			if d := abs(sorted[j] - v); d < best {
				best = d
			}
		}
	}
	return best
}

// isTrackStart reports whether a second is where a new song takes over.
func (t *Timeline) isTrackStart(sec int) bool {
	for _, s := range t.TrackStart {
		if s == sec {
			return true
		}
	}
	return false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// LoudnessMatch correlates how hard a block is against how loud the music
// under it actually is, weighted by how long the block lasts. Positive means
// the chorus is the hard part; near zero means the ride ignores the song.
func (t *Timeline) LoudnessMatch() float64 {
	var zs, rs, ws []float64
	for _, b := range t.Blocks {
		if b.SegmentType == "recovery" || b.SegmentType == "cooldown" {
			continue // deliberately soft regardless of the music
		}
		sum, n := 0.0, 0
		for s := b.Start; s < b.End; s++ {
			sum += t.RankAt(s)
			n++
		}
		if n == 0 {
			continue
		}
		zs = append(zs, float64(b.Zone))
		rs = append(rs, sum/float64(n))
		ws = append(ws, float64(b.End-b.Start))
	}
	return weightedPearson(zs, rs, ws)
}

func weightedPearson(x, y, w []float64) float64 {
	if len(x) < 2 {
		return 0
	}
	var sw, mx, my float64
	for i := range x {
		sw += w[i]
		mx += x[i] * w[i]
		my += y[i] * w[i]
	}
	if sw == 0 {
		return 0
	}
	mx /= sw
	my /= sw
	var cov, vx, vy float64
	for i := range x {
		dx, dy := x[i]-mx, y[i]-my
		cov += w[i] * dx * dy
		vx += w[i] * dx * dx
		vy += w[i] * dy * dy
	}
	if vx == 0 || vy == 0 {
		return 0
	}
	return cov / math.Sqrt(vx*vy)
}

// Rideability reports things a rider cannot actually do in sequence, as
// opposed to things that merely look wrong in a table.
func (t *Timeline) Rideability() []string {
	var out []string
	for i := 1; i < len(t.Blocks); i++ {
		a, b := t.Blocks[i-1], t.Blocks[i]
		if a.CadenceFrom > 0 && b.CadenceFrom > 0 && !t.isTrackStart(b.Start) {
			// a new song legitimately brings a new cadence; a jump mid-song does not
			if d := abs(b.CadenceFrom - a.CadenceFrom); d > 15 {
				out = append(out, fmt.Sprintf("%s cadence jumps %d rpm (%d to %d)",
					clock(b.Start), d, a.CadenceFrom, b.CadenceFrom))
			}
		}
	}
	// position flapping: three or more changes inside a minute
	var changes []int
	for i := 1; i < len(t.Blocks); i++ {
		if t.Blocks[i].Position != t.Blocks[i-1].Position {
			changes = append(changes, t.Blocks[i].Start)
		}
	}
	for i := 2; i < len(changes); i++ {
		if changes[i]-changes[i-2] <= 60 {
			out = append(out, fmt.Sprintf("%s position changes 3 times in %ds",
				clock(changes[i-2]), changes[i]-changes[i-2]))
			i += 2
		}
	}
	run, runStart := 0, 0
	for _, b := range t.Blocks {
		if b.Position == "standing" {
			if run == 0 {
				runStart = b.Start
			}
			run += b.End - b.Start
		} else {
			if run > 90 {
				out = append(out, fmt.Sprintf("%s standing for %ds", clock(runStart), run))
			}
			run = 0
		}
	}
	if run > 90 {
		out = append(out, fmt.Sprintf("%s standing for %ds", clock(runStart), run))
	}
	for _, b := range t.Blocks {
		if b.Position == "standing" && b.CadenceFrom > 0 && (b.CadenceFrom < 55 || b.CadenceFrom > 80) {
			out = append(out, fmt.Sprintf("%s standing at %d rpm", clock(b.Start), b.CadenceFrom))
		}
	}
	return out
}

// Monotony is the longest stretch, in seconds, with no change of zone.
func (t *Timeline) Monotony() int {
	longest, run := 0, 0
	for i, b := range t.Blocks {
		if i > 0 && b.Zone == t.Blocks[i-1].Zone {
			run += b.End - b.Start
		} else {
			run = b.End - b.Start
		}
		if run > longest {
			longest = run
		}
	}
	return longest
}

func clock(s int) string { return fmt.Sprintf("%d:%02d", s/60, s%60) }
