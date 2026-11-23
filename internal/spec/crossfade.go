package spec

// DefaultCrossfadeSec is how much of each track MOWL's timeline overlaps with
// the next. MOWL requires Spotify's crossfade at exactly 10 seconds, with
// Automix and gapless playback turned off; either of those moves the overlap
// away from 10 and desyncs the ride.
const DefaultCrossfadeSec = 10

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
