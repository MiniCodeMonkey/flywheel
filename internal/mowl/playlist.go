package mowl

import (
	"context"
	"net/url"
)

func (c *Client) ImportSpotifyPlaylist(ctx context.Context, spotifyID string) (Playlist, error) {
	var pl Playlist
	err := c.do(ctx, "PUT", "/v1/Spotify/Playlist",
		map[string]string{"SpotifyPlaylistID": spotifyID}, &pl)
	return pl, err
}

func (c *Client) SpotifyPlaylist(ctx context.Context, spotifyID string) (Playlist, error) {
	var pl Playlist
	err := c.do(ctx, "GET", "/v1/Spotify/Playlists/"+url.PathEscape(spotifyID), nil, &pl)
	return pl, err
}

func (c *Client) AudioAnalysis(ctx context.Context, spotifyTrackID string) ([]Section, error) {
	var out struct {
		Sections []Section `json:"sections"`
	}
	err := c.do(ctx, "GET", "/v1/Spotify/Tracks/"+url.PathEscape(spotifyTrackID)+"/AudioAnalysis", nil, &out)
	return out.Sections, err
}
