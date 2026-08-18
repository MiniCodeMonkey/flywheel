package mowl

import "encoding/json"

type envelope struct {
	Data  json.RawMessage `json:"Data"`
	Error *apiError       `json:"Error"`
	Stack *string         `json:"Stack"`
}

type apiError struct {
	Message string `json:"Message"`
}

type Track struct {
	SpotifyTrackID string `json:"SpotifyTrackID"`
	Artist         string `json:"Artist"`
	Title          string `json:"Title"`
	BPM            int    `json:"BPM"`
	DurationMs     int    `json:"Duration"`
	TrackID        int    `json:"TrackID"`
}

type Playlist struct {
	PlaylistID        int     `json:"PlaylistID"`
	SpotifyPlaylistID string  `json:"SpotifyPlaylistID"`
	PlaylistName      string  `json:"PlaylistName"`
	TrackCount        int     `json:"TrackCount"`
	Tracks            []Track `json:"Tracks"`
}

type Section struct {
	Start    float64 `json:"start"`
	Duration float64 `json:"duration"`
	Loudness float64 `json:"loudness"`
}
