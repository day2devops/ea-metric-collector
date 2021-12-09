package metrics

import (
	"testing"
	"time"

	gogithub "github.com/google/go-github/v39/github"
	"github.com/stretchr/testify/assert"

	"github.com/day2devops/ea-metric-extractor/pkg/github"
)

func Test_newGitRepositoryMetric(t *testing.T) {
	r := github.Repository{
		ID:     int64(123),
		Org:    "test-org",
		Name:   "test-repo",
		Topics: []string{"portfolio-myport", "product-myprod", "team-myteam"},
		Detail: &gogithub.Repository{
			ID:               gogithub.Int64(123),
			Name:             gogithub.String("test-repo"),
			CreatedAt:        &gogithub.Timestamp{Time: time.Now()},
			UpdatedAt:        &gogithub.Timestamp{Time: time.Now()},
			PushedAt:         &gogithub.Timestamp{Time: time.Now()},
			DefaultBranch:    gogithub.String("main"),
			AllowSquashMerge: gogithub.Bool(true),
			AllowRebaseMerge: gogithub.Bool(false),
		},
	}

	metrics := newGitRepositoryMetric(&r)

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(123), metrics.ID)
	assert.Equal(t, "test-org", metrics.Org)
	assert.Equal(t, "test-repo", metrics.RepositoryName)
	assert.Equal(t, "myport", metrics.Portfolio)
	assert.Equal(t, "myprod", metrics.Product)
	assert.Equal(t, "myteam", metrics.Team)
	assert.Equal(t, r.Detail.CreatedAt.Time, *metrics.Created)
	assert.Equal(t, r.Detail.UpdatedAt.Time, *metrics.Updated)
	assert.Equal(t, r.Detail.PushedAt.Time, *metrics.Pushed)
	assert.Equal(t, "main", metrics.DefaultBranch)
	assert.True(t, metrics.Squashable)
	assert.False(t, metrics.Rebaseable)
	assert.False(t, metrics.Protected)
	assert.Equal(t, 0, metrics.BranchCount)
	assert.Equal(t, 0, metrics.ReleaseCount)
	assert.NotNil(t, metrics.AsOf)
}

func Test_newGitRepositoryMetric_MissingGitHubData(t *testing.T) {
	r := github.Repository{
		ID:   int64(123),
		Org:  "test-org",
		Name: "test-repo",
		Detail: &gogithub.Repository{
			ID:   gogithub.Int64(123),
			Name: gogithub.String("test-repo"),
		},
	}

	metrics := newGitRepositoryMetric(&r)

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(123), metrics.ID)
	assert.Equal(t, "test-org", metrics.Org)
	assert.Equal(t, "test-repo", metrics.RepositoryName)
	assert.Equal(t, "", metrics.Portfolio)
	assert.Equal(t, "", metrics.Product)
	assert.Equal(t, "", metrics.Team)
	assert.Nil(t, metrics.Created)
	assert.Nil(t, metrics.Updated)
	assert.Nil(t, metrics.Pushed)
	assert.Equal(t, "", metrics.DefaultBranch)
	assert.False(t, metrics.Squashable)
	assert.False(t, metrics.Rebaseable)
	assert.False(t, metrics.Protected)
	assert.Equal(t, 0, metrics.BranchCount)
	assert.Equal(t, 0, metrics.ReleaseCount)
	assert.NotNil(t, metrics.AsOf)
}

func Test_newGitRepositoryMetric_WithBranchInfo(t *testing.T) {
	r := github.Repository{
		ID:   int64(123),
		Org:  "test-org",
		Name: "test-repo",
		Detail: &gogithub.Repository{
			ID:            gogithub.Int64(123),
			Name:          gogithub.String("test-repo"),
			DefaultBranch: gogithub.String("main"),
		},
		Branches: []*gogithub.Branch{
			{
				Name:      gogithub.String("branch-1"),
				Protected: gogithub.Bool(false),
			},
			{
				Name:      gogithub.String("main"),
				Protected: gogithub.Bool(true),
			},
			{
				Name:      gogithub.String("branch-2"),
				Protected: gogithub.Bool(false),
			},
		},
	}

	metrics := newGitRepositoryMetric(&r)

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(123), metrics.ID)
	assert.Equal(t, "test-org", metrics.Org)
	assert.Equal(t, "test-repo", metrics.RepositoryName)
	assert.Nil(t, metrics.Created)
	assert.Nil(t, metrics.Updated)
	assert.Nil(t, metrics.Pushed)
	assert.Equal(t, "main", metrics.DefaultBranch)
	assert.False(t, metrics.Squashable)
	assert.False(t, metrics.Rebaseable)
	assert.True(t, metrics.Protected)
	assert.Equal(t, 3, metrics.BranchCount)
	assert.Equal(t, 0, metrics.ReleaseCount)
	assert.NotNil(t, metrics.AsOf)
}

func Test_newGitRepositoryMetric_WithBranchInfo_NoDefaultBranch(t *testing.T) {
	r := github.Repository{
		ID:   int64(123),
		Org:  "test-org",
		Name: "test-repo",
		Detail: &gogithub.Repository{
			ID:   gogithub.Int64(123),
			Name: gogithub.String("test-repo"),
		},
		Branches: []*gogithub.Branch{
			{
				Name:      gogithub.String("branch-1"),
				Protected: gogithub.Bool(false),
			},
			{
				Name:      gogithub.String("main"),
				Protected: gogithub.Bool(true),
			},
			{
				Name:      gogithub.String("branch-2"),
				Protected: gogithub.Bool(false),
			},
		},
	}

	metrics := newGitRepositoryMetric(&r)

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(123), metrics.ID)
	assert.Equal(t, "test-org", metrics.Org)
	assert.Equal(t, "test-repo", metrics.RepositoryName)
	assert.Nil(t, metrics.Created)
	assert.Nil(t, metrics.Updated)
	assert.Nil(t, metrics.Pushed)
	assert.Equal(t, "", metrics.DefaultBranch)
	assert.False(t, metrics.Squashable)
	assert.False(t, metrics.Rebaseable)
	assert.False(t, metrics.Protected)
	assert.Equal(t, 3, metrics.BranchCount)
	assert.Equal(t, 0, metrics.ReleaseCount)
	assert.NotNil(t, metrics.AsOf)
}

func Test_newGitRepositoryMetric_WithReleaseInfo(t *testing.T) {
	r := github.Repository{
		ID:   int64(123),
		Org:  "test-org",
		Name: "test-repo",
		Detail: &gogithub.Repository{
			ID:            gogithub.Int64(123),
			Name:          gogithub.String("test-repo"),
			DefaultBranch: gogithub.String("main"),
		},
		Releases: []*gogithub.RepositoryRelease{
			{Name: gogithub.String("release-1")},
			{Name: gogithub.String("release-2")},
			{Name: gogithub.String("release-3")},
		},
	}

	metrics := newGitRepositoryMetric(&r)

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(123), metrics.ID)
	assert.Equal(t, "test-org", metrics.Org)
	assert.Equal(t, "test-repo", metrics.RepositoryName)
	assert.Nil(t, metrics.Created)
	assert.Nil(t, metrics.Updated)
	assert.Nil(t, metrics.Pushed)
	assert.Equal(t, "main", metrics.DefaultBranch)
	assert.False(t, metrics.Squashable)
	assert.False(t, metrics.Rebaseable)
	assert.False(t, metrics.Protected)
	assert.Equal(t, 0, metrics.BranchCount)
	assert.Equal(t, 3, metrics.ReleaseCount)
	assert.NotNil(t, metrics.AsOf)
}
