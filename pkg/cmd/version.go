package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/day2devops/ea-metric-extractor/pkg/version"
)

var (
	versionExample = `  # Print the current installed git-what package version
  git-what version`
	short = false
)

// newVersionCmd returns a new initialized instance of the version sub command
func newVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Print the current git-what package version.",
		Long:    `Print the current installed git-what package version.`,
		Example: versionExample,
		RunE:    VersionCmd,
	}

	versionCmd.Flags().BoolVar(&short, "short", false, "Provide just the short version.")

	return versionCmd
}

// VersionCmd performs the version sub command
func VersionCmd(cmd *cobra.Command, args []string) error {
	gwVersion := version.Get()
	if short {
		fmt.Fprintln(cmd.OutOrStdout(), gwVersion.GitVersion)
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "git-what version: %s\n", fmt.Sprintf("%#v", gwVersion))
	return nil
}
