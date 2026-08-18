// internal/cli/delete.go
package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <programID>",
		Short: "Delete a MOWL program",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid programID %q: %w", args[0], err)
			}
			ctx := cmd.Context()
			cl, _, err := newClient(ctx)
			if err != nil {
				return err
			}
			if err := cl.DeleteProgram(ctx, id); err != nil {
				return err
			}
			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				fmt.Fprintf(cmd.OutOrStdout(), "{\"deleted\":%d}\n", id)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted program %d\n", id)
			return nil
		},
	}
	return cmd
}
