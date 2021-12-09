package github

import (
	"context"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

// ClientCreator interface for representing github client creation functions
type ClientCreator interface {
	NewGitHubClient(baseURL string, token string) (*github.Client, error)
}

// ClientFactory factory implementation for ClientCreator interface
type ClientFactory struct {
}

// NewGitHubClient creates a client to access the GitHub API
func (ClientFactory) NewGitHubClient(baseURL string, token string) (*github.Client, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	return github.NewEnterpriseClient(baseURL, baseURL, tc)
}
