package metrics

import (
	"regexp"
	"time"

	"github.com/golang/glog"

	"github.com/day2devops/ea-metric-extractor/pkg/github"
)

// Options to use when updating repository metrics
type Options struct {
	ForceAllRepoEval  bool
	ForceMetricUpdate bool
}

// Processor defines methods for metric management
type Processor interface {
	RepositoriesForOrg(orgNa string, options Options) error
	Repository(orgNa string, repoNa string) error
}

// ProcessorCreator interface for creation of repository processors
type ProcessorCreator interface {
	NewProcessor(collector github.DataCollector, dataMgr DataManager) Processor
}

// ProcessorFactory factory implementation for implementing repository processor creator interface
type ProcessorFactory struct {
}

// NewProcessor construct instance of Processor
func (ProcessorFactory) NewProcessor(collector github.DataCollector, dataMgr DataManager) Processor {
	return Manager{
		DataCollector: collector,
		DataManager:   dataMgr,
	}
}

// Manager used to manage git hub metrics
type Manager struct {
	DataCollector github.DataCollector
	DataManager   DataManager
}

// RepositoriesForOrg process all repositories for an organization
func (m Manager) RepositoriesForOrg(orgNa string, options Options) error {
	glog.Infof("Updating metrics for repositories in org %s", orgNa)

	// read statistcs for metric data
	now := time.Now().UTC()
	found, stats := m.DataManager.ReadCacheStats(orgNa)
	if !found {
		stats = &CacheStats{}
	}

	// process the necessary repositories
	changedAfter := stats.UpdatedAt
	if options.ForceAllRepoEval {
		changedAfter = nil
	}

	repositories, err := m.DataCollector.ListRepositories(orgNa, changedAfter)
	if err != nil {
		return err
	}

	activeOrgRepos := make(map[string]bool)
	for _, r := range repositories {
		if options.ForceAllRepoEval {
			activeOrgRepos[r.Name] = true
		}
		if !options.ForceMetricUpdate && m.skipRepositoryNotUpdated(r) {
			glog.V(2).Infof("Skipping metric updates for repository: %s/%s", r.Org, r.Name)
			continue
		}
		if err = m.Repository(orgNa, r.Name); err != nil {
			return err
		}
	}

	// when evaluating all repositories, look for repositories in cache to cleanup
	if options.ForceAllRepoEval {
		if err = m.cleanOldRepositories(orgNa, activeOrgRepos); err != nil {
			return err
		}
	}

	// write statistics for metric data
	stats.UpdatedAt = &now
	m.DataManager.StoreCacheStats(orgNa, *stats)
	return nil
}

// Repository handles metric gathering for the given repository
func (m Manager) Repository(orgNa string, repoNa string) error {
	// Get the core repository details
	glog.Infof("Updating metrics for repository: %s/%s", orgNa, repoNa)
	repository, err := m.DataCollector.GetRepository(orgNa, repoNa)
	if err != nil {
		return err
	}

	// Extract metrics and store them
	repoMetrics := newGitRepositoryMetric(repository)
	err = m.DataManager.StoreMetrics(repoMetrics)
	return err
}

// Determine if repository should be skipped since it hasn't been updated
func (m Manager) skipRepositoryNotUpdated(r github.Repository) bool {
	// Don't skip if updated timestamp not available for comparison
	if r.Changed == nil {
		glog.V(3).Infof("NOT Skipping: Update timestamp not found for repository %s", r.Name)
		return false
	}

	// Attempt to read metrics...not skipping if error or not found
	found, metrics, err := m.DataManager.ReadMetrics(r.Org, r.Name)
	if !found || err != nil {
		glog.V(3).Infof("NOT Skipping: Metrics found: %t", found)
		if err != nil {
			glog.V(3).Infof("NOT Skipping: Error reading metrics (%s)", err.Error())
		}
		return false
	}

	// Skip when repository updated before the cached as of timestamp
	return r.Changed.Before(*metrics.AsOf)
}

// Find repositories that have metric data that no longer exist and delete them
func (m Manager) cleanOldRepositories(org string, activeOrgRepos map[string]bool) error {
	// Pull list of repositories that exist within the cache for organization
	orgFilter, err := regexp.Compile("^" + org + "$")
	if err != nil {
		return err
	}
	cacheKeys, err := m.DataManager.ListMetrics(ListMetricOptions{orgFilter: orgFilter})
	if err != nil {
		return err
	}

	// Check each of the keys against the active map, when not found, delete
	for _, cacheKey := range cacheKeys {
		if _, found := activeOrgRepos[cacheKey.Name]; !found {
			glog.Infof("Deleting metrics for repository: %s/%s", org, cacheKey.Name)
			if err = m.DataManager.DeleteMetrics(org, cacheKey.Name); err != nil {
				return err
			}
		}
	}
	return nil
}
