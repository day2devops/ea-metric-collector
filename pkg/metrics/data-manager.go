package metrics

import (
	"regexp"
	"time"
)

// CacheStats represents data about the overall cache of repository metrics
type CacheStats struct {
	Org       string     `json:"org"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

// ListMetricOptions represents options for retrieving lists of repositories with metrics
type ListMetricOptions struct {
	orgFilter  *regexp.Regexp
	repoFilter *regexp.Regexp
}

// Key represents the key fields for a repository metric
type Key struct {
	Org  string
	Name string
}

// DataManager Interface for implementing a metric persistence layer
type DataManager interface {
	StoreMetrics(metrics GitRepositoryMetric) error
	ReadMetrics(org string, repo string) (found bool, metric *GitRepositoryMetric, err error)
	DeleteMetrics(org string, repo string) error
	ListMetrics(options ListMetricOptions) ([]Key, error)
	StoreCacheStats(org string, stats CacheStats)
	ReadCacheStats(org string) (found bool, stats *CacheStats)
}
