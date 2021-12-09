package metrics

import (
	"time"

	"github.com/golang/glog"
	gogithub "github.com/google/go-github/v39/github"

	"github.com/day2devops/ea-metric-extractor/pkg/github"
)

// GitRepositoryMetric defines structure for tracking GH metrics
type GitRepositoryMetric struct {
	ID             int64             `json:"id" bson:"id"`
	Org            string            `json:"org" bson:"org"`
	RepositoryName string            `json:"repositoryName" bson:"repositoryName"`
	Created        *time.Time        `json:"created" bson:"created"`
	Updated        *time.Time        `json:"updated" bson:"updated"`
	Pushed         *time.Time        `json:"pushed" bson:"pushed"`
	DefaultBranch  string            `json:"defaultBranch" bson:"defaultBranch"`
	Squashable     bool              `json:"squashable" bson:"squashable"`
	Rebaseable     bool              `json:"rebaseable" bson:"rebaseable"`
	Protected      bool              `json:"protected" bson:"protected"`
	BranchCount    int               `json:"branchCount" bson:"branchCount"`
	ReleaseCount   int               `json:"releaseCount" bson:"releaseCount"`
	CommitCount    int               `json:"commitCount" bson:"commitCount"`
	CodeLineCount  int               `json:"codeLineCount" bson:"codeLineCount"`
	PullRequest    PullRequestMetric `json:"pullRequest" bson:"pullRequest"`
	Build          BuildMetric       `json:"build" bson:"build"`
	CodeQuality    CodeQualityMetric `json:"codeQuality" bson:"codeQuality"`
	AsOf           *time.Time        `json:"asOf" bson:"asOf"`
}

// PullRequestMetric defines structure for pull request metrics
type PullRequestMetric struct {
	CreatedTodayCount       int     `json:"createdTodayCount" bson:"createdTodayCount"`
	CreatedWeekCount        int     `json:"createdWeekCount" bson:"createdWeekCount"`
	CreatedMonthCount       int     `json:"createdMonthCount" bson:"createdMonthCount"`
	MergedTodayCount        int     `json:"mergedTodayCount" bson:"mergedTodayCount"`
	MergedWeekCount         int     `json:"mergedWeekCount" bson:"mergedWeekCount"`
	MergedMonthCount        int     `json:"mergedMonthCount" bson:"mergedMonthCount"`
	AvgMinutesOpenLastMonth float32 `json:"avgMinutesOpenLastMonth" bson:"avgMinutesOpenLastMonth"`
}

// BuildMetric defines structure for build metrics
type BuildMetric struct {
	BuildsTodayCount         int     `json:"buildsTodayCount" bson:"buildsTodayCount"`
	BuildsWeekCount          int     `json:"buildsWeekCount" bson:"buildsWeekCount"`
	BuildsMonthCount         int     `json:"buildsMonthCount" bson:"buildsMonthCount"`
	AvgBuildMinutesLastMonth float32 `json:"avgBuildMinutesLastMonth" bson:"avgBuildMinutesLastMonth"`
}

// CodeQualityMetric defines structure for code quality metrics
type CodeQualityMetric struct {
	BlockerCount    int     `json:"blockerCount" bson:"blockerCount"`
	CriticalCount   int     `json:"criticalCount" bson:"criticalCount"`
	MajorCount      int     `json:"majorCount" bson:"majorCount"`
	IssueCount      int     `json:"issueCount" bson:"issueCount"`
	TestCount       int     `json:"testCount" bson:"testCount"`
	TestErrorCount  int     `json:"testErrorCount" bson:"testErrorCount"`
	TestFailCount   int     `json:"testFailCount" bson:"testFailCount"`
	TestCoveragePct float32 `json:"testCoveragePct" bson:"testCoveragePct"`
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

	// Mock data for NOW on other metrics
	metrics.CommitCount = 101
	metrics.CodeLineCount = 5000
	metrics.PullRequest = PullRequestMetric{
		CreatedTodayCount:       3,
		CreatedWeekCount:        7,
		CreatedMonthCount:       20,
		MergedTodayCount:        1,
		MergedWeekCount:         3,
		MergedMonthCount:        10,
		AvgMinutesOpenLastMonth: 380.25,
	}
	metrics.Build = BuildMetric{
		BuildsTodayCount:         10,
		BuildsWeekCount:          25,
		BuildsMonthCount:         200,
		AvgBuildMinutesLastMonth: 2.5,
	}
	metrics.CodeQuality = CodeQualityMetric{
		BlockerCount:    0,
		CriticalCount:   0,
		MajorCount:      0,
		IssueCount:      4,
		TestCount:       25,
		TestErrorCount:  0,
		TestFailCount:   0,
		TestCoveragePct: 83.4,
	}

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
