package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewGitHubClient(t *testing.T) {
	client, err := ClientFactory{}.NewGitHubClient("baseurlval", "authtokenval")
	assert.NotNil(t, client)
	assert.NoError(t, err)
}
