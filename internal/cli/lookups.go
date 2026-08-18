// internal/cli/lookups.go
package cli

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/minicodemonkey/flywheel/internal/mowl"
	"github.com/spf13/cobra"
)

func newLookupsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lookups",
		Short: "Dump MOWL segment categories, activity types, and flywheel's alias maps",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cl, _, err := newClient(ctx)
			if err != nil {
				return err
			}
			segCats, err := cl.SegmentCategories(ctx)
			if err != nil {
				return err
			}
			actTypes, err := cl.ActivityTypes(ctx)
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				b, err := json.MarshalIndent(struct {
					SegmentCategories []mowl.SegmentCategory `json:"segment_categories"`
					ActivityTypes     []mowl.ActivityType    `json:"activity_types"`
					SegmentTypeAlias  map[string]int         `json:"segment_type_alias"`
					PositionAlias     map[string]int         `json:"position_alias"`
				}{segCats, actTypes, mowl.SegmentTypeAlias, mowl.PositionAlias}, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Segment categories:")
			for _, s := range segCats {
				fmt.Fprintf(out, "  %4d  %-20s warmup=%v recovery=%v cooldown=%v\n",
					s.SegmentCategoryID, s.Name, s.IsWarmup, s.IsActiveRecovery, s.IsCooldown)
			}
			fmt.Fprintln(out, "Activity types:")
			for _, a := range actTypes {
				fmt.Fprintf(out, "  %4d  %s\n", a.ActivityTypeID, a.ActivityTypeName)
			}
			fmt.Fprintln(out, "flywheel segment type aliases:")
			for _, k := range sortedKeys(mowl.SegmentTypeAlias) {
				fmt.Fprintf(out, "  %-12s -> %d\n", k, mowl.SegmentTypeAlias[k])
			}
			fmt.Fprintln(out, "flywheel position aliases:")
			for _, k := range sortedKeys(mowl.PositionAlias) {
				fmt.Fprintf(out, "  %-12s -> %d\n", k, mowl.PositionAlias[k])
			}
			return nil
		},
	}
	return cmd
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
