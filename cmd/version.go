package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaacov/kubectl-debug-queries/pkg/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("kubectl-debug-queries %s (commit: %s, built: %s)\n",
			version.Version, version.GitCommit, version.BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
