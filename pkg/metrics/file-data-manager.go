package metrics

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/golang/glog"
)

// FileDataManager file based implementation of MetricDataManager
type FileDataManager struct {
	DataDir string
}

// StoreMetrics Persist the supplied metrics
func (fdm FileDataManager) StoreMetrics(metrics GitRepositoryMetric) error {
	filename := fdm.repositoryFileName(metrics.Org, metrics.RepositoryName)
	glog.V(2).Infof("Writing metric data for repository %s to file %s", metrics.RepositoryName, filename)
	return fdm.writeFile(filename, metrics)
}

// ReadMetrics Read the metrics for supplied repository
func (fdm FileDataManager) ReadMetrics(org string, repo string) (found bool, metric *GitRepositoryMetric, err error) {
	metric = &GitRepositoryMetric{}
	filename := fdm.repositoryFileName(org, repo)
	glog.V(2).Infof("Reading metric data for repository %s/%s from file %s", org, repo, filename)
	found, err = fdm.readFile(filename, metric)
	if !found || err != nil {
		metric = nil
	}
	return
}

// DeleteMetrics Delete the metrics for the supplied repository
func (fdm FileDataManager) DeleteMetrics(org string, repo string) error {
	filename := fdm.repositoryFileName(org, repo)
	glog.V(2).Infof("Deleting metric data for repository %s/%s from file %s", org, repo, filename)
	err := os.Remove(filename)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	return err
}

// ListMetrics List the known repositories with metrics that match the supplied options
func (fdm FileDataManager) ListMetrics(opts ListMetricOptions) ([]Key, error) {
	glog.V(2).Infof("Listing repository metrics found in %s", fdm.DataDir)
	files, err := ioutil.ReadDir(fdm.DataDir)
	if err != nil {
		return nil, err
	}

	matcher := regexp.MustCompile(`^org-(.+)\.repo-(.+)\.json$`)

	var allKeys []Key
	for _, file := range files {
		name := file.Name()
		matches := matcher.FindStringSubmatch(name)
		if matches == nil || matches[1] == "" || matches[2] == "" {
			glog.V(2).Infof("Filtered file: %s", name)
			continue
		}
		org := matches[1]
		repo := matches[2]

		if opts.orgFilter != nil && !opts.orgFilter.MatchString(org) {
			glog.V(2).Infof("Filtered entry, org value (%s) didn't match supplied filter(%s)", org, opts.orgFilter)
			continue
		}

		if opts.repoFilter != nil && !opts.repoFilter.MatchString(repo) {
			glog.V(2).Infof("Filtered entry, repo value (%s) didn't match supplied filter(%s)", repo, opts.repoFilter)
			continue
		}

		allKeys = append(allKeys, Key{Org: org, Name: repo})
	}

	return allKeys, nil
}

// StoreCacheStats store the statistics for overall cache statistics
func (fdm FileDataManager) StoreCacheStats(org string, stats CacheStats) {
	filename := fdm.cacheStatsFileName(org)
	glog.V(2).Infof("Writing cache stats to file %s", filename)
	err := fdm.writeFile(filename, stats)
	if err != nil {
		glog.Warningf("Problem writing cache statistics to file %s: %s", filename, err.Error())
	}
}

// ReadCacheStats read the overall cache statistics
func (fdm FileDataManager) ReadCacheStats(org string) (found bool, stats *CacheStats) {
	filename := fdm.cacheStatsFileName(org)
	stats = &CacheStats{}
	glog.V(2).Infof("Reading cache stats from file %s", filename)
	found, err := fdm.readFile(filename, stats)
	if !found || err != nil {
		stats = nil
		if err != nil {
			glog.Warningf("Problem reading cache statistics from file %s, treating as not found: %s", filename, err.Error())
		}
	}
	return
}

// builds the file name for the supplied repository
func (fdm FileDataManager) repositoryFileName(org string, repoName string) string {
	return filepath.Join(fdm.DataDir, "org-"+org+".repo-"+repoName+".json")
}

// builds the overall cache statistics
func (fdm FileDataManager) cacheStatsFileName(org string) string {
	return filepath.Join(fdm.DataDir, "org-"+org+".cache-stats.json")
}

// write file with supplied name using supplied struct as data
func (fdm FileDataManager) writeFile(filename string, s interface{}) error {
	// convert struct into json
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	// create base directory if necessary
	err = os.MkdirAll(fdm.DataDir, os.ModePerm)
	if err != nil {
		return err
	}

	// write data to file
	glog.V(3).Infof("Data being stored to file %s: %s", filename, data)
	return ioutil.WriteFile(filename, data, 0644)
}

// read file data into struct
func (fdm FileDataManager) readFile(filename string, s interface{}) (found bool, err error) {
	// Check for file existence
	found = false
	if _, err = os.Stat(filename); err != nil {
		glog.V(3).Infof("Result of looking for file %s: %s", filename, err.Error())
		if errors.Is(err, os.ErrNotExist) {
			err = nil
			return
		}
		return
	}

	// Read and parse file if exists
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	err = json.Unmarshal(file, s)
	if err != nil {
		return
	}

	found = true
	return
}
