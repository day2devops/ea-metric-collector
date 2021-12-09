package metrics

import (
	"errors"
	"testing"
	"time"

	gogithub "github.com/google/go-github/v39/github"
	"github.com/nyarly/spies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/day2devops/ea-metric-extractor/pkg/github"
)

func TestNewProcessor(t *testing.T) {
	collector := github.RepositoryDataCollector{}
	dataMgr := FileDataManager{}

	rp := ProcessorFactory{}.NewProcessor(collector, dataMgr)

	assert.Equal(t, collector, rp.(Manager).DataCollector)
	assert.Equal(t, dataMgr, rp.(Manager).DataManager)
}

func TestRepositoriesForOrg(t *testing.T) {
	repos := []github.Repository{
		{ID: int64(123), Org: "testorg", Name: "test-repo1", Detail: &gogithub.Repository{}},
		{ID: int64(124), Org: "testorg", Name: "test-repo2", Detail: &gogithub.Repository{}},
	}
	dataCollectorSpy := &DataCollectorSpy{Spy: spies.NewSpy()}
	dataCollectorSpy.MatchMethod("ListRepositories", spies.AnyArgs, repos, nil)
	dataCollectorSpy.MatchMethod("GetRepository", spies.AnyArgs, &repos[0], nil)

	dataMgrSpy := &DataManagerSpy{Spy: spies.NewSpy()}
	dataMgrSpy.MatchMethod("StoreMetrics", spies.AnyArgs, nil)
	dataMgrSpy.MatchMethod("StoreCacheStats", spies.AnyArgs, nil)
	dataMgrSpy.MatchMethod("ListMetrics", spies.AnyArgs, []Key{
		{Org: "testorg", Name: "test-repo1"},
		{Org: "testorg", Name: "old-repo1"},
		{Org: "testorg", Name: "test-repo2"},
		{Org: "testorg", Name: "old-repo2"},
	}, nil)
	dataMgrSpy.MatchMethod("DeleteMetrics", spies.AnyArgs, nil)

	metricMgr := Manager{
		DataCollector: dataCollectorSpy,
		DataManager:   dataMgrSpy,
	}

	err := metricMgr.RepositoriesForOrg("testorg", Options{true, true})

	assert.NoError(t, err)

	assert.Equal(t, 1, len(dataCollectorSpy.CallsTo("ListRepositories")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("ListRepositories")[0].PassedArgs().String(0))

	assert.Equal(t, 2, len(dataCollectorSpy.CallsTo("GetRepository")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(0))
	assert.Equal(t, "test-repo1", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(1))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[1].PassedArgs().String(0))
	assert.Equal(t, "test-repo2", dataCollectorSpy.CallsTo("GetRepository")[1].PassedArgs().String(1))

	assert.Equal(t, 2, len(dataMgrSpy.CallsTo("StoreMetrics")))
	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("ReadCacheStats")))
	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("StoreCacheStats")))
	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("ListMetrics")))

	assert.Equal(t, 2, len(dataMgrSpy.CallsTo("DeleteMetrics")))
	assert.Equal(t, "testorg", dataMgrSpy.CallsTo("DeleteMetrics")[0].PassedArgs().String(0))
	assert.Equal(t, "old-repo1", dataMgrSpy.CallsTo("DeleteMetrics")[0].PassedArgs().String(1))
	assert.Equal(t, "testorg", dataMgrSpy.CallsTo("DeleteMetrics")[1].PassedArgs().String(0))
	assert.Equal(t, "old-repo2", dataMgrSpy.CallsTo("DeleteMetrics")[1].PassedArgs().String(1))
}

func TestRepositoriesForOrg_SkipSinceNotUpdated(t *testing.T) {
	oneHourAgo := time.Now().Add(time.Hour * -1)
	twoHourAgo := time.Now().Add(time.Hour * -2)
	repos := []github.Repository{
		{ID: int64(123), Org: "testorg", Name: "test-repo1", Changed: &oneHourAgo, Detail: &gogithub.Repository{}},
		{ID: int64(124), Org: "testorg", Name: "test-repo2", Changed: &twoHourAgo, Detail: &gogithub.Repository{}},
		{ID: int64(125), Org: "testorg", Name: "test-repo3", Changed: &twoHourAgo, Detail: &gogithub.Repository{}},
		{ID: int64(126), Org: "testorg", Name: "test-repo4", Detail: &gogithub.Repository{}},
		{ID: int64(127), Org: "testorg", Name: "test-repo5", Changed: &twoHourAgo, Detail: &gogithub.Repository{}},
	}
	dataCollectorSpy := &DataCollectorSpy{Spy: spies.NewSpy()}
	dataCollectorSpy.MatchMethod("ListRepositories", spies.AnyArgs, repos, nil)
	dataCollectorSpy.MatchMethod("GetRepository", spies.AnyArgs, &repos[1], nil)

	dataMgrSpy := &DataManagerSpy{Spy: spies.NewSpy()}
	dataMgrSpy.MatchMethod("StoreMetrics", spies.AnyArgs, nil)

	repo1Metrics := GitRepositoryMetric{AsOf: &twoHourAgo}
	dataMgrSpy.MatchMethod("ReadMetrics", func(args mock.Arguments) bool {
		return args.String(1) == "test-repo1"
	}, true, &repo1Metrics, nil)

	repo2Metrics := GitRepositoryMetric{AsOf: &oneHourAgo}
	dataMgrSpy.MatchMethod("ReadMetrics", func(args mock.Arguments) bool {
		return args.String(1) == "test-repo2"
	}, true, &repo2Metrics, nil)

	dataMgrSpy.MatchMethod("ReadMetrics", func(args mock.Arguments) bool {
		return args.String(1) == "test-repo3"
	}, false, nil, nil)

	dataMgrSpy.MatchMethod("ReadMetrics", func(args mock.Arguments) bool {
		return args.String(1) == "test-repo5"
	}, false, nil, errors.New("repo read error"))

	metricMgr := Manager{
		DataCollector: dataCollectorSpy,
		DataManager:   dataMgrSpy,
	}

	err := metricMgr.RepositoriesForOrg("testorg", Options{})

	assert.NoError(t, err)

	assert.Equal(t, 1, len(dataCollectorSpy.CallsTo("ListRepositories")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("ListRepositories")[0].PassedArgs().String(0))

	assert.Equal(t, 4, len(dataCollectorSpy.CallsTo("GetRepository")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(0))
	assert.Equal(t, "test-repo1", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(1))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[1].PassedArgs().String(0))
	assert.Equal(t, "test-repo3", dataCollectorSpy.CallsTo("GetRepository")[1].PassedArgs().String(1))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[2].PassedArgs().String(0))
	assert.Equal(t, "test-repo4", dataCollectorSpy.CallsTo("GetRepository")[2].PassedArgs().String(1))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[3].PassedArgs().String(0))
	assert.Equal(t, "test-repo5", dataCollectorSpy.CallsTo("GetRepository")[3].PassedArgs().String(1))

	assert.Equal(t, 4, len(dataMgrSpy.CallsTo("StoreMetrics")))
	assert.Equal(t, 4, len(dataMgrSpy.CallsTo("ReadMetrics")))
	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("ListMetrics")))
	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("DeleteMetrics")))
	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("ReadCacheStats")))
	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("StoreCacheStats")))
}

func TestRepositoriesForOrg_ListRepositoriesError(t *testing.T) {
	dataCollectorSpy := &DataCollectorSpy{Spy: spies.NewSpy()}
	dataCollectorSpy.MatchMethod("ListRepositories", spies.AnyArgs, nil, errors.New("list repo error"))

	dataMgrSpy := &DataManagerSpy{Spy: spies.NewSpy()}
	dataMgrSpy.MatchMethod("StoreMetrics", spies.AnyArgs, nil)

	metricMgr := Manager{
		DataCollector: dataCollectorSpy,
		DataManager:   dataMgrSpy,
	}

	err := metricMgr.RepositoriesForOrg("testorg", Options{})

	assert.Error(t, err)

	assert.Equal(t, 1, len(dataCollectorSpy.CallsTo("ListRepositories")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("ListRepositories")[0].PassedArgs().String(0))

	assert.Equal(t, 0, len(dataCollectorSpy.CallsTo("GetRepository")))

	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("StoreMetrics")))
	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("ListMetrics")))
	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("DeleteMetrics")))
	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("ReadCacheStats")))
	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("StoreCacheStats")))
}

func TestRepositoriesForOrg_ListMetricsError(t *testing.T) {
	repos := []github.Repository{
		{ID: int64(123), Org: "testorg", Name: "test-repo1", Detail: &gogithub.Repository{}},
		{ID: int64(124), Org: "testorg", Name: "test-repo2", Detail: &gogithub.Repository{}},
	}
	dataCollectorSpy := &DataCollectorSpy{Spy: spies.NewSpy()}
	dataCollectorSpy.MatchMethod("ListRepositories", spies.AnyArgs, repos, nil)
	dataCollectorSpy.MatchMethod("GetRepository", spies.AnyArgs, &repos[0], nil)

	dataMgrSpy := &DataManagerSpy{Spy: spies.NewSpy()}
	dataMgrSpy.MatchMethod("StoreMetrics", spies.AnyArgs, nil)
	dataMgrSpy.MatchMethod("StoreCacheStats", spies.AnyArgs, nil)
	dataMgrSpy.MatchMethod("ListMetrics", spies.AnyArgs, nil, errors.New("list metric error"))
	dataMgrSpy.MatchMethod("DeleteMetrics", spies.AnyArgs, nil)

	metricMgr := Manager{
		DataCollector: dataCollectorSpy,
		DataManager:   dataMgrSpy,
	}

	err := metricMgr.RepositoriesForOrg("testorg", Options{true, true})

	assert.Error(t, err)
	assert.Equal(t, "list metric error", err.Error())

	assert.Equal(t, 1, len(dataCollectorSpy.CallsTo("ListRepositories")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("ListRepositories")[0].PassedArgs().String(0))

	assert.Equal(t, 2, len(dataCollectorSpy.CallsTo("GetRepository")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(0))
	assert.Equal(t, "test-repo1", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(1))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[1].PassedArgs().String(0))
	assert.Equal(t, "test-repo2", dataCollectorSpy.CallsTo("GetRepository")[1].PassedArgs().String(1))

	assert.Equal(t, 2, len(dataMgrSpy.CallsTo("StoreMetrics")))
	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("ReadCacheStats")))
	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("StoreCacheStats")))
	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("ListMetrics")))
	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("DeleteMetrics")))
}

func TestRepositoriesForOrg_DeleteMetricsError(t *testing.T) {
	repos := []github.Repository{
		{ID: int64(123), Org: "testorg", Name: "test-repo1", Detail: &gogithub.Repository{}},
		{ID: int64(124), Org: "testorg", Name: "test-repo2", Detail: &gogithub.Repository{}},
	}
	dataCollectorSpy := &DataCollectorSpy{Spy: spies.NewSpy()}
	dataCollectorSpy.MatchMethod("ListRepositories", spies.AnyArgs, repos, nil)
	dataCollectorSpy.MatchMethod("GetRepository", spies.AnyArgs, &repos[0], nil)

	dataMgrSpy := &DataManagerSpy{Spy: spies.NewSpy()}
	dataMgrSpy.MatchMethod("StoreMetrics", spies.AnyArgs, nil)
	dataMgrSpy.MatchMethod("StoreCacheStats", spies.AnyArgs, nil)
	dataMgrSpy.MatchMethod("ListMetrics", spies.AnyArgs, []Key{
		{Org: "testorg", Name: "test-repo1"},
		{Org: "testorg", Name: "old-repo1"},
		{Org: "testorg", Name: "test-repo2"},
		{Org: "testorg", Name: "old-repo2"},
	}, nil)
	dataMgrSpy.MatchMethod("DeleteMetrics", spies.AnyArgs, errors.New("delete metric error"))

	metricMgr := Manager{
		DataCollector: dataCollectorSpy,
		DataManager:   dataMgrSpy,
	}

	err := metricMgr.RepositoriesForOrg("testorg", Options{true, true})

	assert.Error(t, err)
	assert.Equal(t, "delete metric error", err.Error())

	assert.Equal(t, 1, len(dataCollectorSpy.CallsTo("ListRepositories")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("ListRepositories")[0].PassedArgs().String(0))

	assert.Equal(t, 2, len(dataCollectorSpy.CallsTo("GetRepository")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(0))
	assert.Equal(t, "test-repo1", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(1))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[1].PassedArgs().String(0))
	assert.Equal(t, "test-repo2", dataCollectorSpy.CallsTo("GetRepository")[1].PassedArgs().String(1))

	assert.Equal(t, 2, len(dataMgrSpy.CallsTo("StoreMetrics")))
	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("ReadCacheStats")))
	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("StoreCacheStats")))
	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("ListMetrics")))
	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("DeleteMetrics")))
}

func TestRepositoriesForOrg_GetRepositoryError(t *testing.T) {
	repos := []github.Repository{
		{ID: int64(123), Org: "testorg", Name: "test-repo1", Detail: &gogithub.Repository{}},
		{ID: int64(124), Org: "testorg", Name: "test-repo2", Detail: &gogithub.Repository{}},
	}
	dataCollectorSpy := &DataCollectorSpy{Spy: spies.NewSpy()}
	dataCollectorSpy.MatchMethod("ListRepositories", spies.AnyArgs, repos, nil)
	dataCollectorSpy.MatchMethod("GetRepository", spies.AnyArgs, nil, errors.New("get repo error"))

	dataMgrSpy := &DataManagerSpy{Spy: spies.NewSpy()}
	dataMgrSpy.MatchMethod("StoreMetrics", spies.AnyArgs, nil)

	metricMgr := Manager{
		DataCollector: dataCollectorSpy,
		DataManager:   dataMgrSpy,
	}

	err := metricMgr.RepositoriesForOrg("testorg", Options{})

	assert.Error(t, err)

	assert.Equal(t, 1, len(dataCollectorSpy.CallsTo("ListRepositories")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("ListRepositories")[0].PassedArgs().String(0))

	assert.Equal(t, 1, len(dataCollectorSpy.CallsTo("GetRepository")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(0))
	assert.Equal(t, "test-repo1", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(1))

	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("StoreMetrics")))
	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("ListMetrics")))
	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("DeleteMetrics")))
	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("ReadCacheStats")))
	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("StoreCacheStats")))
}

func TestRepository(t *testing.T) {
	repo := &github.Repository{
		ID:     int64(123),
		Org:    "testorg",
		Name:   "testrepo",
		Detail: &gogithub.Repository{},
	}

	dataCollectorSpy := &DataCollectorSpy{Spy: spies.NewSpy()}
	dataCollectorSpy.MatchMethod("GetRepository", spies.AnyArgs, repo, nil)

	dataMgrSpy := &DataManagerSpy{Spy: spies.NewSpy()}
	dataMgrSpy.MatchMethod("StoreMetrics", spies.AnyArgs, nil)

	metricMgr := Manager{
		DataCollector: dataCollectorSpy,
		DataManager:   dataMgrSpy,
	}

	err := metricMgr.Repository("testorg", "testrepo")

	assert.NoError(t, err)

	assert.Equal(t, 1, len(dataCollectorSpy.CallsTo("GetRepository")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(0))
	assert.Equal(t, "testrepo", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(1))

	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("StoreMetrics")))
}

func TestRepository_RepoManagerError(t *testing.T) {
	dataCollectorSpy := &DataCollectorSpy{Spy: spies.NewSpy()}
	dataCollectorSpy.MatchMethod("GetRepository", spies.AnyArgs, nil, errors.New("get repo error"))

	dataMgrSpy := &DataManagerSpy{Spy: spies.NewSpy()}
	dataMgrSpy.MatchMethod("StoreMetrics", spies.AnyArgs, nil)

	metricMgr := Manager{
		DataCollector: dataCollectorSpy,
		DataManager:   dataMgrSpy,
	}

	err := metricMgr.Repository("testorg", "testrepo")

	assert.Error(t, err)

	assert.Equal(t, 1, len(dataCollectorSpy.CallsTo("GetRepository")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(0))
	assert.Equal(t, "testrepo", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(1))

	assert.Equal(t, 0, len(dataMgrSpy.CallsTo("StoreMetrics")))
}

func TestRepository_DataManagerError(t *testing.T) {
	repo := &github.Repository{
		ID:     int64(123),
		Org:    "testorg",
		Name:   "testrepo",
		Detail: &gogithub.Repository{},
	}

	dataCollectorSpy := &DataCollectorSpy{Spy: spies.NewSpy()}
	dataCollectorSpy.MatchMethod("GetRepository", spies.AnyArgs, repo, nil)

	dataMgrSpy := &DataManagerSpy{Spy: spies.NewSpy()}
	dataMgrSpy.MatchMethod("StoreMetrics", spies.AnyArgs, errors.New("store metric error"))

	metricMgr := Manager{
		DataCollector: dataCollectorSpy,
		DataManager:   dataMgrSpy,
	}

	err := metricMgr.Repository("testorg", "testrepo")

	assert.Error(t, err)

	assert.Equal(t, 1, len(dataCollectorSpy.CallsTo("GetRepository")))
	assert.Equal(t, "testorg", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(0))
	assert.Equal(t, "testrepo", dataCollectorSpy.CallsTo("GetRepository")[0].PassedArgs().String(1))

	assert.Equal(t, 1, len(dataMgrSpy.CallsTo("StoreMetrics")))
}

type DataCollectorSpy struct {
	*spies.Spy
	github.DataCollector
}

func (rdcs *DataCollectorSpy) ListRepositories(org string, changedSince *time.Time) ([]github.Repository, error) {
	res := rdcs.Called(org, changedSince)
	repos := res.Get(0)
	if repos == nil {
		return nil, res.Error(1)
	}
	return repos.([]github.Repository), res.Error(1)
}

func (rdcs *DataCollectorSpy) GetRepository(org string, name string) (*github.Repository, error) {
	res := rdcs.Called(org, name)
	repo := res.Get(0)
	if repo == nil {
		return nil, res.Error(1)
	}
	return repo.(*github.Repository), res.Error(1)
}

type DataManagerSpy struct {
	*spies.Spy
	DataManager
}

func (dms *DataManagerSpy) StoreMetrics(metrics GitRepositoryMetric) error {
	res := dms.Called(metrics)
	return res.Error(0)
}

func (dms *DataManagerSpy) ReadMetrics(org string, repo string) (bool, *GitRepositoryMetric, error) {
	res := dms.Called(org, repo)
	metrics := res.Get(1)
	if metrics == nil {
		return res.Bool(0), nil, res.Error(2)
	}
	return res.Bool(0), metrics.(*GitRepositoryMetric), res.Error(2)
}

func (dms *DataManagerSpy) DeleteMetrics(org string, repo string) error {
	res := dms.Called(org, repo)
	return res.Error(0)
}

func (dms *DataManagerSpy) ListMetrics(options ListMetricOptions) ([]Key, error) {
	res := dms.Called(options)
	repos := res.Get(0)
	if repos == nil {
		return nil, res.Error(1)
	}
	return repos.([]Key), res.Error(1)
}

func (dms *DataManagerSpy) StoreCacheStats(org string, stats CacheStats) {
	dms.Called(org, stats)
}

func (dms *DataManagerSpy) ReadCacheStats(org string) (bool, *CacheStats) {
	res := dms.Called(org)
	stats := res.Get(1)
	if stats == nil {
		return res.Bool(0), nil
	}
	return res.Bool(0), stats.(*CacheStats)
}
