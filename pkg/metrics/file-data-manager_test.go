package metrics

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStoreAndReadAndDeleteMetrics(t *testing.T) {
	o := "testorg"
	r := "test-repo"
	m := GitRepositoryMetric{
		ID:             int64(123),
		Org:            o,
		RepositoryName: r,
	}

	dataMgr := FileDataManager{
		DataDir: ".",
	}
	err := dataMgr.StoreMetrics(m)

	assert.NoError(t, err)

	path := filepath.Join(".", "org-testorg.repo-test-repo.json")
	defer os.Remove(path)

	_, err = os.Stat(path)
	assert.NoError(t, err, "expected file is missing or has error")
	if err == nil {
		found, afterMetrics, err := dataMgr.ReadMetrics(o, r)

		assert.True(t, found)
		assert.NoError(t, err)
		assert.Equal(t, m.ID, afterMetrics.ID)
		assert.Equal(t, m.RepositoryName, afterMetrics.RepositoryName)

		err = dataMgr.DeleteMetrics(o, r)

		assert.NoError(t, err)

		_, err = os.Stat(path)
		assert.True(t, errors.Is(err, os.ErrNotExist))
	}
}

func TestReadMetrics_NotFound(t *testing.T) {
	dataMgr := FileDataManager{DataDir: "."}
	found, metrics, err := dataMgr.ReadMetrics("testorg", "test-read-repo")

	assert.False(t, found)
	assert.Nil(t, metrics)
	assert.NoError(t, err)
}

func TestDeleteMetrics_NotFound(t *testing.T) {
	dataMgr := FileDataManager{DataDir: "."}
	err := dataMgr.DeleteMetrics("testorg", "test-del-repo")
	assert.NoError(t, err)
}

func TestListRepositories(t *testing.T) {
	// Create test files for test cases
	testfiles := []string{
		"cache-stats.json", "org-test-1.repo-test-repo-1.json", "org-test-1.repo-test-repo-2.json",
		"org-test-2.repo-test-repo-3.json", "org-test-2.repo-test-repo-4.json", "org-test-2.repo-test-repo-5.json",
		"org-test-2.json", "org-test-2.badrepo.json",
		"org-ej.repo-repo1.json", "org-ej.repo-repo2.repoext.json",
	}
	for _, testfile := range testfiles {
		ioutil.WriteFile(filepath.Join(".", testfile), []byte{'t', 'e', 's', 't'}, 0644)
	}
	defer func() {
		for _, testfile := range testfiles {
			os.Remove(filepath.Join(".", testfile))
		}
	}()

	tests := []struct {
		name     string
		options  ListMetricOptions
		expected []Key
	}{
		{
			"no filters", ListMetricOptions{}, []Key{
				{Org: "test-1", Name: "test-repo-1"},
				{Org: "test-1", Name: "test-repo-2"},
				{Org: "test-2", Name: "test-repo-3"},
				{Org: "test-2", Name: "test-repo-4"},
				{Org: "test-2", Name: "test-repo-5"},
				{Org: "ej", Name: "repo1"},
				{Org: "ej", Name: "repo2.repoext"},
			},
		},
		{
			"org filter only", ListMetricOptions{
				orgFilter: regexp.MustCompile("^test-"),
			},
			[]Key{
				{Org: "test-1", Name: "test-repo-1"},
				{Org: "test-1", Name: "test-repo-2"},
				{Org: "test-2", Name: "test-repo-3"},
				{Org: "test-2", Name: "test-repo-4"},
				{Org: "test-2", Name: "test-repo-5"},
			},
		},
		{
			"repo filter only", ListMetricOptions{
				repoFilter: regexp.MustCompile("1$"),
			},
			[]Key{
				{Org: "test-1", Name: "test-repo-1"},
				{Org: "ej", Name: "repo1"},
			},
		},
		{
			"org and repo filter", ListMetricOptions{
				orgFilter:  regexp.MustCompile("^ej$"),
				repoFilter: regexp.MustCompile("1$"),
			},
			[]Key{
				{Org: "ej", Name: "repo1"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			dataMgr := FileDataManager{DataDir: "."}
			keys, err := dataMgr.ListMetrics(tt.options)

			assert.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(keys))
			for _, expectedKey := range tt.expected {
				assert.True(t, func() bool {
					match := false
					for _, key := range keys {
						if key.Name == expectedKey.Name && key.Org == expectedKey.Org {
							match = true
							break
						}
					}
					return match
				}(), "Missing "+expectedKey.Name)
			}
		})
	}
}

func TestStoreAndReadCacheStats(t *testing.T) {
	now := time.Now().UTC()
	s := CacheStats{
		UpdatedAt: &now,
	}

	dataMgr := FileDataManager{
		DataDir: ".",
	}
	dataMgr.StoreCacheStats("test", s)

	path := filepath.Join(".", "org-test.cache-stats.json")
	defer os.Remove(path)

	_, err := os.Stat(path)
	assert.NoError(t, err, "expected file is missing or has error")
	if err == nil {
		found, afterStats := dataMgr.ReadCacheStats("test")

		assert.True(t, found)
		assert.NoError(t, err)
		assert.Equal(t, s.UpdatedAt, afterStats.UpdatedAt)
	}
}

func TestReadCacheStats_NotFound(t *testing.T) {
	dataMgr := FileDataManager{
		DataDir: ".",
	}

	found, stats := dataMgr.ReadCacheStats("test")

	assert.False(t, found)
	assert.Nil(t, stats)
}
