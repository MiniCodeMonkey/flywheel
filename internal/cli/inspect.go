// internal/cli/inspect.go
package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/minicodemonkey/flywheel/internal/mowl"
	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "playlist",
		Short: "Inspect Spotify playlists via MOWL",
	}
	cmd.AddCommand(newPlaylistInspectCmd())
	return cmd
}

func newPlaylistInspectCmd() *cobra.Command {
	var withSections bool
	var wait time.Duration
	cmd := &cobra.Command{
		Use:   "inspect <spotify-playlist-id>",
		Short: "Import a Spotify playlist into MOWL and print its tracks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cl, _, err := newClient(ctx)
			if err != nil {
				return err
			}
			// ImportSpotifyPlaylist's returned playlist is intentionally discarded;
			// SpotifyPlaylist below re-fetches it hydrated with track metadata.
			// A freshly imported playlist indexes asynchronously: durations and
			// BPM (Tempo) both arrive late, and can take minutes on a playlist
			// MOWL has not seen before. Poll until both land or --wait expires.
			imported, err := cl.ImportSpotifyPlaylist(ctx, args[0])
			if err != nil {
				return err
			}
			var pl mowl.Playlist
			deadline := time.Now().Add(wait)
			for attempt := 0; ; attempt++ {
				pl, err = cl.SpotifyPlaylist(ctx, args[0])
				if err != nil {
					return err
				}
				if playlistHydrated(pl, imported.TrackCount) && bpmComplete(pl) {
					break
				}
				if time.Now().After(deadline) {
					break
				}
				if attempt == 0 && wait > 0 {
					fmt.Fprintf(cmd.ErrOrStderr(), "waiting up to %s for MOWL to index the playlist...\n", wait)
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(3 * time.Second):
				}
			}
			if missing := missingDurations(pl); missing > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: duration not yet available for %d track(s); "+
					"a course built now will fail validation with \"tracks are 0s\" - "+
					"re-run once MOWL finishes indexing, or raise --wait\n", missing)
			}
			if missing := missingBPM(pl); missing > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "note: BPM not yet available for %d track(s)\n", missing)
			}

			type sectionOut struct {
				Start    float64 `json:"start"`
				Duration float64 `json:"duration"`
				Loudness float64 `json:"loudness"`
			}
			type trackOut struct {
				Index    int          `json:"index"`
				Title    string       `json:"title"`
				Artist   string       `json:"artist"`
				BPM      int          `json:"bpm"`
				Duration int          `json:"duration_sec"`
				Sections []sectionOut `json:"sections,omitempty"`
			}
			var out []trackOut
			for i, tr := range pl.Tracks {
				to := trackOut{Index: i + 1, Title: tr.Title, Artist: tr.Artist, BPM: tr.BPM, Duration: tr.DurationMs / 1000}
				if withSections {
					secs, err := cl.AudioAnalysis(ctx, tr.SpotifyTrackID)
					if err == nil {
						for _, s := range secs {
							to.Sections = append(to.Sections, sectionOut{Start: s.Start, Duration: s.Duration, Loudness: s.Loudness})
						}
					}
				}
				out = append(out, to)
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				b, err := json.MarshalIndent(struct {
					Playlist string     `json:"playlist"`
					Tracks   []trackOut `json:"tracks"`
				}{pl.PlaylistName, out}, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s (%d tracks)\n", pl.PlaylistName, len(out))
			for _, t := range out {
				fmt.Fprintf(cmd.OutOrStdout(), "  %2d. %-30s %-20s %3d bpm   %d:%02d\n",
					t.Index, t.Title, t.Artist, t.BPM, t.Duration/60, t.Duration%60)
				if withSections {
					for _, sec := range t.Sections {
						fmt.Fprintf(cmd.OutOrStdout(), "      %3d:%02d  %5.1fs  %6.1f dB\n",
							int(sec.Start)/60, int(sec.Start)%60, sec.Duration, sec.Loudness)
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&withSections, "sections", false, "also fetch per-section start, duration and loudness for each track")
	cmd.Flags().DurationVar(&wait, "wait", 3*time.Minute, "how long to wait for MOWL to finish indexing a freshly imported playlist")
	return cmd
}
