// internal/cli/apply.go
package cli

import (
	"fmt"
	"os"

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
			tracks, err := trackInfo(ctx, course, 0)
			if err != nil {
				return err
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
			res, err := apply.Apply(ctx, cl, course, styles, userID)
			if err != nil {
				return err
			}
			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				fmt.Fprintf(cmd.OutOrStdout(), "{\"program_id\":%d,\"playlist_id\":%d,\"server_tss\":%g}\n",
					res.ProgramID, res.PlaylistID, res.ServerTSS)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created program %d (playlist %d), server TSS %.1f\n",
				res.ProgramID, res.PlaylistID, res.ServerTSS)
			return nil
		},
	}
	return cmd
}
