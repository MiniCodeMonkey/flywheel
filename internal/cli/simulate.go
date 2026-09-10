package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/minicodemonkey/flywheel/internal/simulate"
	"github.com/minicodemonkey/flywheel/internal/spec"
	"github.com/spf13/cobra"
)

func newSimulateCmd() *cobra.Command {
	var tolerance int
	cmd := &cobra.Command{
		Use:   "simulate <course.yaml>",
		Short: "Walk a course against its music and report what does not line up",
		Long: `Walk a course against the music it will actually play.

preview checks that a course is valid. simulate checks that it is any good:
whether each interval boundary lands on a musical section boundary, whether
the hard part lands on the loud part, and whether the ride is rideable in
sequence. It never writes to MOWL and never fails a ride -- it reports.

The output is deterministic, so it can be diffed between runs.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			course, err := spec.ParseCourse(b)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			cl, _, err := newClient(ctx)
			if err != nil {
				return err
			}
			pl, err := cl.SpotifyPlaylist(ctx, course.Playlist.SpotifyID)
			if err != nil {
				return err
			}
			var tracks []simulate.Track
			for i, tr := range pl.Tracks {
				t := simulate.Track{Index: i + 1, Title: tr.Title, Artist: tr.Artist,
					BPM: tr.BPM, DurationSec: tr.DurationMs / 1000}
				secs, err := cl.AudioAnalysis(ctx, tr.SpotifyTrackID)
				if err == nil {
					for _, s := range secs {
						t.Sections = append(t.Sections, simulate.Section{
							Duration: s.Duration, Loudness: s.Loudness})
					}
				}
				tracks = append(tracks, t)
			}

			tl := simulate.Build(&course, tracks, course.Playlist.Crossfade())
			rep := simulate.Run(tl, tolerance)
			rep.Name = course.Name

			if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
				out, err := json.MarshalIndent(rep, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			}
			fmt.Fprint(cmd.OutOrStdout(), rep.Text(tolerance))
			return nil
		},
	}
	cmd.Flags().IntVar(&tolerance, "tolerance", 3,
		"how far an interval boundary may sit from a musical section boundary (seconds)")
	return cmd
}
