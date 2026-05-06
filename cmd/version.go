package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// BuildInfo carries build metadata populated at link time.
type BuildInfo struct {
	GoVersion string
	Version   string
	Commit    string
	Date      string
}

func (b BuildInfo) String() string {
	return fmt.Sprintf("mahjong %s built with %s from %s on %s",
		b.Version, b.GoVersion, b.Commit, b.Date)
}

var buildInfo BuildInfo

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print the version, commit hash, and build date of mahjong.",
	Args:  cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		_, err := fmt.Fprintln(c.OutOrStdout(), buildInfo.String())
		return err
	},
}
