// internal/cli/apply.go
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/minicodemonkey/flywheel/internal/apply"
	"github.com/minicodemonkey/flywheel/internal/mowl"
	"github.com/minicodemonkey/flywheel/internal/simulate"
	"github.com/minicodemonkey/flywheel/internal/spec"
	"github.com/spf13/cobra"
)

func newApplyCmd() *cobra.Command {
	var skipCheck bool
	cmd := &cobra.Command{
		Use:   "apply <course.yaml>",
		Short: "Validate a course and create it in MOWL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			b, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			course, err := spec.ParseCourse(b)
			if err != nil {
				return err
			}
			styles, err := loadStyles()
			if err != nil {
				return err
			}
			cl, _, err := newClient(ctx)
			if err != nil {
				return err
			}

			// Import (idempotent) before validating: tracks/BPM populate
			// asynchronously, so poll until the playlist is fully hydrated.
			pl, err := cl.ImportSpotifyPlaylist(ctx, course.Playlist.SpotifyID)
			if err != nil {
				return fmt.Errorf("import playlist: %w", err)
			}
			hydrated := pl
			ok := false
			for attempt := 0; attempt < 8; attempt++ {
				p, err := cl.SpotifyPlaylist(ctx, course.Playlist.SpotifyID)
				if err != nil {
					return fmt.Errorf("fetch playlist: %w", err)
				}
				// Durations are required for validation; BPM (Tempo) may lag a
				// little after a fresh import. Consider the playlist hydrated
				// once every track has a duration, and prefer BPM too.
				if playlistHydrated(p, pl.TrackCount) {
					hydrated = p
					ok = true
					if bpmComplete(p) {
						break
					}
				}
				if attempt < 7 {
					time.Sleep(2 * time.Second)
				}
			}
			if !ok {
				return fmt.Errorf("playlist track durations not available yet (Spotify indexing) — try again in a moment")
			}
			if missing := missingBPM(hydrated); missing > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "note: BPM not yet available for %d track(s); proceeding\n", missing)
			}

			tracks := map[int]spec.TrackInfo{}
			for i, tr := range hydrated.Tracks {
				tracks[i+1] = spec.TrackInfo{DurationSec: tr.DurationMs / 1000, Title: tr.Title}
			}
			tracks = spec.CrossfadeTracks(tracks, course.Playlist.Crossfade())

			if !skipCheck {
				reportAlignment(cmd, ctx, cl, course, hydrated)
			}

			if errs := spec.Validate(course, tracks, segmentTypeMap(), positionMap(), 5); len(errs) > 0 {
				for _, e := range errs {
					fmt.Fprintln(cmd.ErrOrStderr(), "validation error:", e)
				}
				return fmt.Errorf("%d validation error(s); aborting apply", len(errs))
			}
			userID, err := cl.Me(ctx)
			if err != nil {
				return fmt.Errorf("get current user: %w", err)
			}
			// Link via the IMPORT response's PlaylistID (the ITC playlist id).
			// `hydrated` comes from the Spotify-catalog GET, which has no ITC
			// PlaylistID — only `pl` (the import result) carries it.
			res, err := apply.Apply(ctx, cl, course, pl, styles, userID)
			if err != nil {
				return err
			}
			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				out, err := json.Marshal(map[string]any{
					"program_id":  res.ProgramID,
					"playlist_id": res.PlaylistID,
					"server_tss":  res.ServerTSS,
				})
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			}
			if res.ReplacedID != 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Replaced program %d with %d (playlist %d), server TSS %.1f\n",
					res.ReplacedID, res.ProgramID, res.PlaylistID, res.ServerTSS)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Created program %d (playlist %d), server TSS %.1f\n",
					res.ProgramID, res.PlaylistID, res.ServerTSS)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&skipCheck, "skip-check", false,
		"skip the music-alignment check before applying")
	return cmd
}

// reportAlignment prints how well the ride follows its music before it is
// created. It never blocks an apply: a course that cannot be analysed, or a
// playlist whose analysis is unavailable, still applies.
func reportAlignment(cmd *cobra.Command, ctx context.Context, cl *mowl.Client,
	course spec.Course, pl mowl.Playlist) {
	var tracks []simulate.Track
	for i, tr := range pl.Tracks {
		t := simulate.Track{Index: i + 1, Title: tr.Title, Artist: tr.Artist,
			BPM: tr.BPM, DurationSec: tr.DurationMs / 1000}
		secs, err := cl.AudioAnalysis(ctx, tr.SpotifyTrackID)
		if err != nil {
			return // analysis unavailable; say nothing rather than guess
		}
		for _, s := range secs {
			t.Sections = append(t.Sections, simulate.Section{Duration: s.Duration, Loudness: s.Loudness})
		}
		tracks = append(tracks, t)
	}
	rep := simulate.Run(simulate.Build(&course, tracks, course.Playlist.Crossfade()), 3)
	fmt.Fprintf(cmd.ErrOrStderr(),
		"check: %.0f%% of boundaries land on the music, loudness match %+.2f, longest flat run %ds\n",
		rep.Alignment.WithinTolerance*100, rep.Loudness, rep.Monotony)
	for _, s := range rep.Issues {
		fmt.Fprintln(cmd.ErrOrStderr(), "check:", s)
	}
}
