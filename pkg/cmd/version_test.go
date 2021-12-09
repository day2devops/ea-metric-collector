package cmd

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionCmd(t *testing.T) {
	// Build and execute command
	cmd := newVersionCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	err := cmd.Execute()

	// Assert results
	assert.NoError(t, err)
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	assert.True(t, strings.HasPrefix(s, "git-what version:"))
}

func TestVersionCmd_ShortFlag(t *testing.T) {
	// Build and execute command
	cmd := newVersionCmd()
	cmd.SetArgs([]string{"--short", "true"})
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	err := cmd.Execute()

	// Assert results
	assert.NoError(t, err)
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	assert.False(t, strings.HasPrefix(s, "git-what version:"))
}
