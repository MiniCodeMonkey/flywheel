package spec

// DefaultCrossfadeSec is the Spotify crossfade MOWL requires: "crossfade has
// to be set to 10 seconds exactly". Spotify starts each track that many
// seconds before the previous one ends, so a playlist's timeline is shorter
// than the sum of its track durations by one crossfade per transition, and
// MOWL lays its editor out the same way.
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
