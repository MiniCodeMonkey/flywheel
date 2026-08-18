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

type Interval struct {
	Duration       int `json:"Duration"`
	RPMFrom        int `json:"RPMFrom"`
	RPMTo          int `json:"RPMTo"`
	Intensity      int `json:"Intensity"`
	FTPFrom        int `json:"FTPFrom"`
	FTPTo          int `json:"FTPTo"`
	PositionTypeID int `json:"PositionTypeID"`
}

type Program struct {
	ProgramID         int    `json:"ProgramID"`
	ProgramCategoryID int    `json:"ProgramCategoryID"`
	Name              string `json:"Name"`
	Description       string `json:"Description,omitempty"`
	IsPublic          bool   `json:"IsPublic"`
	ActivityTypeID    int    `json:"ActivityTypeID"`
	BikeTypeID        int    `json:"BikeTypeID"`
	PlaylistID        int    `json:"PlaylistID,omitempty"`
	SegmentCount      int    `json:"SegmentCount,omitempty"`
	TotalDuration     int    `json:"TotalDuration,omitempty"`
}

type SegmentFlags struct {
	IsWarmup         bool
	IsActiveRecovery bool
	IsCooldown       bool
}
