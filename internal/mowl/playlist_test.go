package mowl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/minicodemonkey/flywheel/internal/config"
)

func serveFixtures(t *testing.T, routes map[string]string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check specific routes first (longer prefixes) before generic ones
		for prefix, file := range routes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				b, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				w.Write(b)
				return
			}
		}
		t.Fatalf("no fixture for %s", r.URL.Path)
	}))
	t.Cleanup(srv.Close)
	return New(config.Config{APIBase: srv.URL, ClientVersion: "8.8.2"}, "t", srv.Client())
}

func TestImportSpotifyPlaylist(t *testing.T) {
	c := serveFixtures(t, map[string]string{
		"/v1/Spotify/Playlist": "../../testdata/playlist_import.json",
	})
	pl, err := c.ImportSpotifyPlaylist(context.Background(), "EXPLAYLIST0000000000001")
	if err != nil {
		t.Fatal(err)
	}
	if pl.PlaylistID == 0 {
		t.Fatalf("empty PlaylistID: %+v", pl)
	}
	if pl.TrackCount == 0 {
		t.Fatalf("empty TrackCount: %+v", pl)
	}
	// Import response has Tracks: null; tracks are populated asynchronously
	// and are read via GET /v1/Spotify/Playlists/{id}
}

func TestSpotifyPlaylist(t *testing.T) {
	c := serveFixtures(t, map[string]string{
		"/v1/Spotify/Playlists/": "../../testdata/program_playlist.json",
	})
	pl, err := c.SpotifyPlaylist(context.Background(), "some-id")
	if err != nil {
		t.Fatal(err)
	}
	if len(pl.Tracks) == 0 {
		t.Fatalf("no tracks in playlist: %+v", pl)
	}
	if pl.Tracks[0].BPM == 0 {
		t.Fatalf("track missing BPM: %+v", pl.Tracks[0])
	}
	if pl.Tracks[0].DurationMs == 0 {
		t.Fatalf("track missing DurationMs: %+v", pl.Tracks[0])
	}
}

func TestAudioAnalysisSections(t *testing.T) {
	c := serveFixtures(t, map[string]string{
		"/v1/Spotify/Tracks/": "../../testdata/audio_analysis.json",
	})
	secs, err := c.AudioAnalysis(context.Background(), "7zGeoy0A1F7NU0wgI4mqoY")
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) == 0 {
		t.Fatal("no sections")
	}
}
