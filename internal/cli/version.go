package cli

import (
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Stamped at release-build time via -ldflags -X. A `go build` or `go install`
// leaves these empty and the values come from the Go toolchain's build info
// instead, so both paths report something meaningful.
var (
	buildVersion string
	buildCommit  string
)

// buildInfo reports the version and VCS revision of the running binary. A
// binary built from a working tree reports "(devel)" with the revision of
// HEAD, so an installed binary that has fallen behind the source tree is
// identifiable at a glance.
func buildInfo() (version, revision string, modified bool) {
	version, revision = "unknown", "unknown"
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	defer func() {
		if buildVersion != "" {
			version = buildVersion
		}
		if buildCommit != "" {
			revision = buildCommit
		}
	}()
	if bi.Main.Version != "" {
		version = bi.Main.Version
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	return
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the flywheel version and the revision it was built from",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			version, revision, modified := buildInfo()
			if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
				b, err := json.MarshalIndent(struct {
					Version  string `json:"version"`
					Revision string `json:"revision"`
					Modified bool   `json:"modified"`
				}{version, revision, modified}, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}
			dirty := ""
			if modified {
				dirty = " (uncommitted changes)"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "flywheel %s\nrevision %s%s\n", version, revision, dirty)
			return nil
		},
	}
}
