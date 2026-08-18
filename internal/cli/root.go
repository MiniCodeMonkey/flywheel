package cli

import "github.com/spf13/cobra"

// NewRootCmd builds the flywheel root command. Subcommands are attached by
// their own files' init-style registration helpers.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "flywheel",
		Short:         "Build MOWL spinning courses from Spotify playlists",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().Bool("json", false, "machine-readable JSON output")
	root.AddCommand(newInitCmd(), newAuthCmd(), newInspectCmd(), newPreviewCmd(),
		newApplyCmd(), newListCmd(), newDeleteCmd(), newLookupsCmd())
	return root
}

// Execute runs the root command.
func Execute() error { return NewRootCmd().Execute() }
