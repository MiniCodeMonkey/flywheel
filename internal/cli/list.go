// internal/cli/list.go
package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your MOWL programs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cl, _, err := newClient(ctx)
			if err != nil {
				return err
			}
			userID, err := cl.Me(ctx)
			if err != nil {
				return fmt.Errorf("get current user: %w", err)
			}
			progs, err := cl.MyPrograms(ctx, userID)
			if err != nil {
				return err
			}
			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				b, err := json.MarshalIndent(progs, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}
			for _, p := range progs {
				fmt.Fprintf(cmd.OutOrStdout(), "%6d  %-40s  %d segments  %ds\n",
					p.ProgramID, p.Name, p.SegmentCount, p.TotalDuration)
			}
			return nil
		},
	}
	return cmd
}
