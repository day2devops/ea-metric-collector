package metrics

import (
	"time"

	"github.com/golang/glog"
	gogithub "github.com/google/go-github/v39/github"

	"github.com/day2devops/ea-metric-extractor/pkg/github"
)

// GitRepositoryMetric defines structure for tracking GH metrics
type GitRepositoryMetric struct {
	ID             int64      `json:"id"`
	Org            string     `json:"org"`
	RepositoryName string     `json:"repositoryName"`
	Created        *time.Time `json:"created"`
	Updated        *time.Time `json:"updated"`
	Pushed         *time.Time `json:"pushed"`
	DefaultBranch  string     `json:"defaultBranch"`
	Squashable     bool       `json:"squashable"`
	Rebaseable     bool       `json:"rebaseable"`
	Protected      bool       `json:"protected"`
	BranchCount    int        `json:"branchCount"`
	ReleaseCount   int        `json:"releaseCount"`
	AsOf           *time.Time `json:"asOf"`
}

// newGitRepositoryMetric extract desired metrics for the supplied repository
func newGitRepositoryMetric(r *github.Repository) GitRepositoryMetric {
	// Populate the base metrics from the repository object
	glog.V(3).Infof("Extracting metric data from repository %+v", r)
	metrics := GitRepositoryMetric{
		ID:             r.ID,
		Org:            r.Org,
		RepositoryName: r.Name,
	}

	metrics.Created = extractTime(r.Detail.CreatedAt)
	metrics.Updated = extractTime(r.Detail.UpdatedAt)
	metrics.Pushed = extractTime(r.Detail.PushedAt)
	metrics.DefaultBranch = extractString(r.Detail.DefaultBranch)
	metrics.Squashable = extractBool(r.Detail.AllowSquashMerge)
	metrics.Rebaseable = extractBool(r.Detail.AllowRebaseMerge)

	// Process branch information if found
	metrics.BranchCount = len(r.Branches)
	metrics.Protected = defaultBranchProtected(r, metrics.DefaultBranch)

	// Process release information if found
	metrics.ReleaseCount = len(r.Releases)

	// Add as of timestamp and return
	now := time.Now().UTC()
	metrics.AsOf = &now
	return metrics
}

// determine if the default branch is protected
func defaultBranchProtected(r *github.Repository, defaultBr string) bool {
	if r.Branches == nil {
		return false
	}
	for _, branch := range r.Branches {
		if extractString(branch.Name) == defaultBr {
			return extractBool(branch.Protected)
		}
	}
	return false
}

// extract time reference if supplied, otherwise default to nil
func extractTime(t *gogithub.Timestamp) *time.Time {
	if t != nil {
		return &t.Time
	}
	return nil
}

// extract string reference if supplied, otherwise default to empty string
func extractString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// extract bool reference if supplied, otherwise default to false
func extractBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}
