// internal/cli/delete.go
package cli

import (
	"encoding/json"
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
			// Look up the program's category first so we can clean it up after.
			catID := 0
			if p, err := cl.Program(ctx, id); err == nil {
				catID = p.ProgramCategoryID
			}
			if err := cl.DeleteProgram(ctx, id); err != nil {
				return err
			}
			// Best-effort: remove the (now-empty) private category. The API
			// refuses to delete non-empty or non-owned categories, so ignore
			// errors here.
			deletedCat := false
			if catID > 0 {
				if err := cl.DeleteCategory(ctx, catID); err == nil {
					deletedCat = true
				}
			}
			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				out, err := json.Marshal(map[string]any{
					"deleted_program_id":  id,
					"deleted_category_id": catIDIf(deletedCat, catID),
				})
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted program %d\n", id)
			if deletedCat {
				fmt.Fprintf(cmd.OutOrStdout(), "Removed empty category %d\n", catID)
			}
			return nil
		},
	}
	return cmd
}

// catIDIf returns id when cond is true, else nil (for JSON output).
func catIDIf(cond bool, id int) any {
	if cond {
		return id
	}
	return nil
}
