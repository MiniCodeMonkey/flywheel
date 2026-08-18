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
	pl, err := c.ImportSpotifyPlaylist(context.Background(), "0478H01T6WxqFi0fevIPP1")
	if err != nil {
		t.Fatal(err)
	}
	if pl.PlaylistID == 0 || pl.TrackCount == 0 || len(pl.Tracks) == 0 {
		t.Fatalf("empty playlist: %+v", pl)
	}
	if pl.Tracks[0].BPM == 0 || pl.Tracks[0].DurationMs == 0 {
		t.Fatalf("track missing bpm/duration: %+v", pl.Tracks[0])
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
