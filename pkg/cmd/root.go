package cmd

import (
	goflag "flag"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/day2devops/ea-metric-extractor/pkg/version"
)

// NewGHWhatCmd creates a new root command for git-what
func NewGHWhatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "git-what",
		Short:        "CLI to Get Metrics from Github",
		SilenceUsage: true,
		Version:      version.Get().GitVersion,
	}

	cmd.AddCommand(newVersionCmd())

	metricCmd, _ := newUpdateMetricsCmd()
	cmd.AddCommand(metricCmd)

	// Add flags from glog to valid flag set
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	return cmd
}
