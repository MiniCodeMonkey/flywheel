package spec

// DefaultCrossfadeSec is how much of each track MOWL's timeline actually
// overlaps with the next. MOWL asks for Spotify's crossfade to be set to 10
// seconds, but its own editor lays tracks out about 9 apart: fitting segment
// starts to track starts across four official rides puts the error at 1-3s for
// 9 and 6-10s for either 8 or 10.
const DefaultCrossfadeSec = 9

// Crossfade is the overlap between consecutive tracks, in seconds. An absent
// crossfade_sec means the required default; an explicit 0 disables it.
func (p Playlist) Crossfade() int {
	if p.CrossfadeSec == nil {
		return DefaultCrossfadeSec
	}
	return *p.CrossfadeSec
}

// CrossfadeTracks shortens every track that has another track after it,
// leaving the last one whole: those trailing seconds play underneath the next
// track's fade-in rather than occupying the ride's timeline.
func CrossfadeTracks(tracks map[int]TrackInfo, crossfadeSec int) map[int]TrackInfo {
	last := 0
	for idx := range tracks {
		if idx > last {
			last = idx
		}
	}
	out := make(map[int]TrackInfo, len(tracks))
	for idx, ti := range tracks {
		if idx != last {
			ti.DurationSec -= crossfadeSec
		}
		out[idx] = ti
	}
	return out
}
