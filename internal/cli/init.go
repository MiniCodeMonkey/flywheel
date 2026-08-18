package cli

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/minicodemonkey/flywheel/internal/config"
	"github.com/spf13/cobra"
)

//go:embed embedded/styles.yaml
var defaultStyles []byte

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Write a starter styles.yaml into the config dir",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := config.Dir()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return err
			}
			p := filepath.Join(dir, "styles.yaml")
			if _, err := os.Stat(p); err == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "styles.yaml already exists; leaving it")
				return nil
			}
			return os.WriteFile(p, defaultStyles, 0o600)
		},
	}
}
