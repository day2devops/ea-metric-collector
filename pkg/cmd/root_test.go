package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewGHWhatCmd(t *testing.T) {
	cmd := NewGHWhatCmd()
	assert.Equal(t, 2, len(cmd.Commands()))

	if found := findCommand(cmd.Commands(), "version"); !found {
		assert.Fail(t, "Version Command Not Found")
	}
	if found := findCommand(cmd.Commands(), "update-metrics"); !found {
		assert.Fail(t, "Update Metrics Command Not Found")
	}
}

func findCommand(commands []*cobra.Command, use string) bool {
	for _, c := range commands {
		if c.Use == use {
			return true
		}
	}
	return false
}
