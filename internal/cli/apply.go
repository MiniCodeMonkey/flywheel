// internal/cli/apply.go
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/minicodemonkey/flywheel/internal/apply"
	"github.com/minicodemonkey/flywheel/internal/spec"
	"github.com/spf13/cobra"
)

func newApplyCmd() *cobra.Command {
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
			for attempt := 0; attempt < 6; attempt++ {
				p, err := cl.SpotifyPlaylist(ctx, course.Playlist.SpotifyID)
				if err != nil {
					return fmt.Errorf("fetch playlist: %w", err)
				}
				if len(p.Tracks) >= pl.TrackCount && len(p.Tracks) > 0 {
					hydrated = p
					ok = true
					break
				}
				time.Sleep(2 * time.Second)
			}
			if !ok {
				return fmt.Errorf("playlist tracks not available yet (Spotify indexing) — try again in a moment")
			}

			tracks := map[int]spec.TrackInfo{}
			for i, tr := range hydrated.Tracks {
				tracks[i+1] = spec.TrackInfo{DurationSec: tr.DurationMs / 1000, Title: tr.Title}
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
			res, err := apply.Apply(ctx, cl, course, hydrated, styles, userID)
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
			fmt.Fprintf(cmd.OutOrStdout(), "Created program %d (playlist %d), server TSS %.1f\n",
				res.ProgramID, res.PlaylistID, res.ServerTSS)
			return nil
		},
	}
	return cmd
}
