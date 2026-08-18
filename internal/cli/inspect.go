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
			// Durations are available immediately; BPM (Tempo) may lag a little
			// after a fresh import, so poll briefly for it.
			imported, err := cl.ImportSpotifyPlaylist(ctx, args[0])
			if err != nil {
				return err
			}
			var pl mowl.Playlist
			for attempt := 0; attempt < 8; attempt++ {
				pl, err = cl.SpotifyPlaylist(ctx, args[0])
				if err != nil {
					return err
				}
				if playlistHydrated(pl, imported.TrackCount) && bpmComplete(pl) {
					break
				}
				if attempt < 7 {
					time.Sleep(2 * time.Second)
				}
			}
			if missing := missingBPM(pl); missing > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "note: BPM not yet available for %d track(s)\n", missing)
			}

			type trackOut struct {
				Index    int       `json:"index"`
				Title    string    `json:"title"`
				Artist   string    `json:"artist"`
				BPM      int       `json:"bpm"`
				Duration int       `json:"duration_sec"`
				Sections []float64 `json:"section_starts,omitempty"`
			}
			var out []trackOut
			for i, tr := range pl.Tracks {
				to := trackOut{Index: i + 1, Title: tr.Title, Artist: tr.Artist, BPM: tr.BPM, Duration: tr.DurationMs / 1000}
				if withSections {
					secs, err := cl.AudioAnalysis(ctx, tr.SpotifyTrackID)
					if err == nil {
						for _, s := range secs {
							to.Sections = append(to.Sections, s.Start)
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
					fmt.Fprintf(cmd.OutOrStdout(), "      sections: %v\n", t.Sections)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&withSections, "sections", false, "also fetch audio-analysis section starts per track")
	return cmd
}
