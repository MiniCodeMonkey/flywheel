// internal/cli/preview.go
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/minicodemonkey/flywheel/internal/mowl"
	"github.com/minicodemonkey/flywheel/internal/plan"
	"github.com/minicodemonkey/flywheel/internal/spec"
	"github.com/spf13/cobra"
)

func newPreviewCmd() *cobra.Command {
	var offlineSecs int
	cmd := &cobra.Command{
		Use:   "preview <course.yaml>",
		Short: "Validate and render a course without writing to MOWL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			var tracks map[int]spec.TrackInfo
			if offlineSecs > 0 {
				tracks, err = trackInfo(cmd.Context(), nil, course, offlineSecs)
			} else {
				var cl *mowl.Client
				cl, _, err = newClient(cmd.Context())
				if err != nil {
					return err
				}
				tracks, err = trackInfo(cmd.Context(), cl, course, 0)
			}
			if err != nil {
				return err
			}
			errs := spec.Validate(course, tracks,
				segmentTypeMap(), positionMap(), 5)
			pv := plan.BuildPreview(course, tracks, styles, errs)
			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				b, _ := pv.JSON()
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
			} else {
				fmt.Fprint(cmd.OutOrStdout(), pv.Text(course.Targets))
			}
			if len(errs) > 0 {
				return fmt.Errorf("%d validation error(s)", len(errs))
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&offlineSecs, "offline-track-seconds", 0, "assign every referenced track this duration; skips the network (testing/inspection)")
	return cmd
}

// trackInfo returns track index → info, either from the offline override or by
// reading the playlist from MOWL via the given client. cl may be nil when
// offline > 0, since no network call is made in that path.
func trackInfo(ctx context.Context, cl *mowl.Client, c spec.Course, offline int) (map[int]spec.TrackInfo, error) {
	out := map[int]spec.TrackInfo{}
	if offline > 0 {
		for _, s := range c.Segments {
			for _, idx := range s.Tracks {
				out[idx] = spec.TrackInfo{DurationSec: offline}
			}
		}
		return out, nil
	}
	pl, err := cl.SpotifyPlaylist(ctx, c.Playlist.SpotifyID)
	if err != nil {
		return nil, err
	}
	for i, tr := range pl.Tracks {
		out[i+1] = spec.TrackInfo{DurationSec: tr.DurationMs / 1000, Title: tr.Title}
	}
	return out, nil
}
