package metrics

import (
	"strings"
	"time"

	"github.com/golang/glog"
	gogithub "github.com/google/go-github/v39/github"

	"github.com/day2devops/ea-metric-extractor/pkg/github"
)

// GitRepositoryMetric defines structure for tracking GH metrics
type GitRepositoryMetric struct {
	ID             int64               `json:"id" bson:"id"`
	Org            string              `json:"org" bson:"org"`
	RepositoryName string              `json:"repositoryName" bson:"repositoryName"`
	Portfolio      string              `json:"portfolio" bson:"portfolio"`
	Product        string              `json:"product" bson:"product"`
	Team           string              `json:"team" bson:"team"`
	Created        *time.Time          `json:"created" bson:"created"`
	Updated        *time.Time          `json:"updated" bson:"updated"`
	Pushed         *time.Time          `json:"pushed" bson:"pushed"`
	DefaultBranch  string              `json:"defaultBranch" bson:"defaultBranch"`
	Squashable     bool                `json:"squashable" bson:"squashable"`
	Rebaseable     bool                `json:"rebaseable" bson:"rebaseable"`
	Protected      bool                `json:"protected" bson:"protected"`
	BranchCount    int                 `json:"branchCount" bson:"branchCount"`
	ReleaseCount   int                 `json:"releaseCount" bson:"releaseCount"`
	CommitCount    int                 `json:"commitCount" bson:"commitCount"`
	CodeByteCount  int                 `json:"codeByteCount" bson:"codeByteCount"`
	Languages      map[string]int      `json:"languages" bson:"languages"`
	PullRequests   []PullRequestMetric `json:"pullRequests" bson:"pullRequests"`
	Build          BuildMetric         `json:"build" bson:"build"`
	CodeQuality    CodeQualityMetric   `json:"codeQuality" bson:"codeQuality"`
	AsOf           *time.Time          `json:"asOf" bson:"asOf"`
}

// PullRequestMetric defines structure for pull request metrics
type PullRequestMetric struct {
	Number      int64      `json:"number" bson:"number"`
	Status      string     `json:"status" bson:"status"`
	CreatedAt   *time.Time `json:"createdAt" bson:"createdAt"`
	ClosedAt    *time.Time `json:"closedAt" bson:"closedAt"`
	MergedAt    *time.Time `json:"mergedAt" bson:"mergedAt"`
	MinutesOpen float64    `json:"minutesOpen" bson:"minutesOpen"`
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

	metrics.Portfolio = parseTopic(r, "portfolio-")
	metrics.Product = parseTopic(r, "product-")
	metrics.Team = parseTopic(r, "team-")
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

	// Process pull request data
	metrics.PullRequests = mapPullRequests(r.PullRequests)

	// Process language data
	metrics.Languages = r.Languages
	for _, cnt := range metrics.Languages {
		metrics.CodeByteCount += cnt
	}

	// Process commits
	for _, c := range r.Contributors {
		metrics.CommitCount += c.GetTotal()
	}

	// Mock data for NOW on other metrics
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

// map pull request metrics
func mapPullRequests(prs []*gogithub.PullRequest) []PullRequestMetric {
	var prMetrics []PullRequestMetric
	for _, pr := range prs {
		prMetric := PullRequestMetric{Number: *pr.ID}
		prMetric.Status = extractString(pr.State)
		prMetric.CreatedAt = pr.CreatedAt
		prMetric.ClosedAt = pr.ClosedAt
		prMetric.MergedAt = pr.MergedAt

		compTS := pr.MergedAt
		if compTS == nil {
			compTS = pr.ClosedAt
		}
		if compTS == nil {
			now := time.Now().UTC()
			compTS = &now
		}
		diff := compTS.Sub(*pr.CreatedAt).Minutes()
		prMetric.MinutesOpen = diff

		prMetrics = append(prMetrics, prMetric)
	}
	return prMetrics
}

// parse the topic value if found using the supplied prefix
func parseTopic(r *github.Repository, prefix string) string {
	for _, topic := range r.Topics {
		if strings.HasPrefix(topic, prefix) {
			return strings.TrimPrefix(topic, prefix)
		}
	}
	return ""
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
