package main

import (
	"flag"
	"os"

	"github.com/golang/glog"

	"github.com/day2devops/ea-metric-extractor/pkg/cmd"
)

func main() {
	// Parse flags (Required for enabling the glog flag options)
	flag.Parse()

	// Execute command
	if err := cmd.NewGHWhatCmd().Execute(); err != nil {
		glog.Error(err)
		os.Exit(-1)
	}
}
