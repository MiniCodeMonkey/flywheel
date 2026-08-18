package mowl

import "encoding/json"

type envelope struct {
	Data    json.RawMessage `json:"Data"`
	Error   json.RawMessage `json:"Error"`
	Stack   json.RawMessage `json:"Stack"`
	Message string          `json:"Message"` // non-enveloped framework errors (400/404)
}

// errorMessage returns a human error string from the envelope's Error field,
// or "" when there is no error. MOWL returns Error as a JSON string ("" on
// success, a message on failure); some framework-level errors instead come
// back un-enveloped as {"Message": "..."}. Both shapes are handled.
func (e envelope) errorMessage() string {
	if n := len(e.Error); n > 0 && string(e.Error) != "null" {
		var s string
		if json.Unmarshal(e.Error, &s) == nil {
			if s != "" {
				return s
			}
		} else {
			var obj struct{ Message string }
			if json.Unmarshal(e.Error, &obj) == nil && obj.Message != "" {
				return obj.Message
			} else {
				return string(e.Error)
			}
		}
	}
	if e.Message != "" {
		return e.Message
	}
	return ""
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
