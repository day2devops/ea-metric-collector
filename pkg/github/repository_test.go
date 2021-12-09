package github

import (
	"flag"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Turn on trace to enable all variable logging code
	flag.Lookup("v").Value.Set("3")
}

func TestListRepositories(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchPages(
			mock.GetOrgsReposByOrg,
			[]github.Repository{
				{
					ID:        github.Int64(123),
					Name:      github.String("Repo-123"),
					UpdatedAt: &github.Timestamp{Time: time.Now()},
				},
				{
					ID:   github.Int64(124),
					Name: github.String("Repo-124"),
				},
			},
			[]github.Repository{},
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	repos, err := m.ListRepositories("testorg", nil)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(repos))

	assert.Equal(t, int64(123), repos[0].ID)
	assert.Equal(t, "testorg", repos[0].Org)
	assert.Equal(t, "Repo-123", repos[0].Name)
	assert.NotNil(t, repos[0].Changed)

	assert.Equal(t, int64(124), repos[1].ID)
	assert.Equal(t, "testorg", repos[1].Org)
	assert.Equal(t, "Repo-124", repos[1].Name)
	assert.Nil(t, repos[1].Changed)
}

func TestListRepositories_WithChangedAfterTimestamp(t *testing.T) {
	oneHourAgo := time.Now().Add(time.Hour * -1)
	twoHourAgo := time.Now().Add(time.Hour * -2)
	threeHourAgo := time.Now().Add(time.Hour * -3)
	fourHourAgo := time.Now().Add(time.Hour * -4)

	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetOrgsReposByOrg,
			// updated sort calls
			[]github.Repository{
				{ID: github.Int64(123), Name: github.String("Repo-123"), UpdatedAt: &github.Timestamp{Time: oneHourAgo}},
				{ID: github.Int64(124), Name: github.String("Repo-124"), UpdatedAt: &github.Timestamp{Time: twoHourAgo}},
				{ID: github.Int64(129), Name: github.String("Repo-129"), UpdatedAt: &github.Timestamp{Time: fourHourAgo}},
			},
			// pushed sort calls
			[]github.Repository{
				{ID: github.Int64(124), Name: github.String("Repo-124"), PushedAt: &github.Timestamp{Time: twoHourAgo}},
				{ID: github.Int64(125), Name: github.String("Repo-125"), PushedAt: &github.Timestamp{Time: twoHourAgo}},
				{ID: github.Int64(126), Name: github.String("Repo-126"), PushedAt: &github.Timestamp{Time: fourHourAgo}},
			},
			// created sort calls
			[]github.Repository{
				{ID: github.Int64(127), Name: github.String("Repo-127"), CreatedAt: &github.Timestamp{Time: fourHourAgo}},
				{ID: github.Int64(128), Name: github.String("Repo-128"), CreatedAt: &github.Timestamp{Time: fourHourAgo}},
			},
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	repos, err := m.ListRepositories("testorg", &threeHourAgo)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(repos))

	assert.Equal(t, int64(123), repos[0].ID)
	assert.Equal(t, "testorg", repos[0].Org)
	assert.Equal(t, "Repo-123", repos[0].Name)
	assert.Equal(t, oneHourAgo.GoString(), repos[0].Changed.GoString())

	assert.Equal(t, int64(124), repos[1].ID)
	assert.Equal(t, "testorg", repos[1].Org)
	assert.Equal(t, "Repo-124", repos[1].Name)
	assert.Equal(t, twoHourAgo.GoString(), repos[1].Changed.GoString())

	assert.Equal(t, int64(125), repos[2].ID)
	assert.Equal(t, "testorg", repos[2].Org)
	assert.Equal(t, "Repo-125", repos[2].Name)
	assert.Equal(t, twoHourAgo.GoString(), repos[2].Changed.GoString())
}

func TestListRepositories_MultiplePages(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchPages(
			mock.GetOrgsReposByOrg,
			[]github.Repository{
				{
					ID:   github.Int64(123),
					Name: github.String("Repo-123"),
				},
				{
					ID:   github.Int64(124),
					Name: github.String("Repo-124"),
				},
			},
			[]github.Repository{
				{
					ID:   github.Int64(234),
					Name: github.String("Repo-234"),
				},
				{
					ID:   github.Int64(235),
					Name: github.String("Repo-235"),
				},
			},
			[]github.Repository{},
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	repos, err := m.ListRepositories("testorg", nil)

	assert.NoError(t, err)
	assert.Equal(t, 4, len(repos))

	assert.Equal(t, int64(123), repos[0].ID)
	assert.Equal(t, "testorg", repos[0].Org)
	assert.Equal(t, "Repo-123", repos[0].Name)

	assert.Equal(t, int64(124), repos[1].ID)
	assert.Equal(t, "testorg", repos[1].Org)
	assert.Equal(t, "Repo-124", repos[1].Name)

	assert.Equal(t, int64(234), repos[2].ID)
	assert.Equal(t, "testorg", repos[2].Org)
	assert.Equal(t, "Repo-234", repos[2].Name)

	assert.Equal(t, int64(235), repos[3].ID)
	assert.Equal(t, "testorg", repos[3].Org)
	assert.Equal(t, "Repo-235", repos[3].Name)
}

func TestListRepositories_ExceedPageLimit(t *testing.T) {
	var repositoryPages []interface{}
	for i := 1; i < 1005; i++ {
		repositoryPage := []github.Repository{
			{ID: github.Int64(int64(i)), Name: github.String("repo")},
		}
		repositoryPages = append(repositoryPages, repositoryPage)
	}
	repositoryPage := []github.Repository{}
	repositoryPages = append(repositoryPages, repositoryPage)

	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchPages(
			mock.GetOrgsReposByOrg,
			repositoryPages...,
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	repos, err := m.ListRepositories("testorg", nil)

	assert.Error(t, err)
	assert.Nil(t, repos)
}

func TestListRepositories_APIError(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.GetOrgsReposByOrg,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(
					w,
					"github api error",
					http.StatusInternalServerError,
				)
			}),
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	repos, err := m.ListRepositories("testorg", nil)

	assert.Error(t, err)
	assert.Nil(t, repos)
}

func TestGetRepository(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchPages(
			mock.GetReposByOwnerByRepo,
			github.Repository{
				ID:        github.Int64(123),
				Name:      github.String("testrepo"),
				UpdatedAt: &github.Timestamp{Time: time.Now()},
			},
		),
		mock.WithRequestMatchPages(
			mock.GetReposBranchesByOwnerByRepo,
			[]github.Branch{
				{
					Name: github.String("branch-1"),
				},
				{
					Name: github.String("branch-2"),
				},
			},
			[]github.Branch{},
		),
		mock.WithRequestMatchPages(
			mock.GetReposReleasesByOwnerByRepo,
			[]github.RepositoryRelease{
				{
					Name: github.String("release-1"),
				},
				{
					Name: github.String("release-2"),
				},
			},
			[]github.RepositoryRelease{},
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	repo, err := m.GetRepository("testorg", "testrepo")

	assert.NoError(t, err)
	assert.NotNil(t, repo)
	assert.Equal(t, int64(123), repo.ID)
	assert.Equal(t, "testorg", repo.Org)
	assert.Equal(t, "testrepo", repo.Name)
	assert.NotNil(t, repo.Changed)
	assert.NotNil(t, repo.Detail)
	assert.Equal(t, 2, len(repo.Branches))
	assert.Equal(t, "branch-1", *repo.Branches[0].Name)
	assert.Equal(t, "branch-2", *repo.Branches[1].Name)
	assert.Equal(t, 2, len(repo.Releases))
	assert.Equal(t, "release-1", *repo.Releases[0].Name)
	assert.Equal(t, "release-2", *repo.Releases[1].Name)
}

func TestGetRepository_RepositoryAPIError(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.GetReposByOwnerByRepo,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(
					w,
					"github api error",
					http.StatusInternalServerError,
				)
			}),
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	repo, err := m.GetRepository("testorg", "testrepo")

	assert.Error(t, err)
	assert.Nil(t, repo)
}

func TestGetRepository_BranchAPIError(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchPages(
			mock.GetReposByOwnerByRepo,
			github.Repository{
				ID:   github.Int64(123),
				Name: github.String("testrepo"),
			},
		),
		mock.WithRequestMatchHandler(
			mock.GetReposBranchesByOwnerByRepo,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(
					w,
					"github api error",
					http.StatusInternalServerError,
				)
			}),
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	repo, err := m.GetRepository("testorg", "testrepo")

	assert.Error(t, err)
	assert.Nil(t, repo)
}

func TestGetRepository_ReleaseAPIError(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchPages(
			mock.GetReposByOwnerByRepo,
			github.Repository{
				ID:   github.Int64(123),
				Name: github.String("testrepo"),
			},
		),
		mock.WithRequestMatchPages(
			mock.GetReposBranchesByOwnerByRepo,
			[]github.Branch{
				{
					Name: github.String("branch-1"),
				},
				{
					Name: github.String("branch-2"),
				},
			},
			[]github.Branch{},
		),
		mock.WithRequestMatchHandler(
			mock.GetReposReleasesByOwnerByRepo,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(
					w,
					"github api error",
					http.StatusInternalServerError,
				)
			}),
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	repo, err := m.GetRepository("testorg", "testrepo")

	assert.Error(t, err)
	assert.Nil(t, repo)
}

func TestGetBranches_MultiplePages(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchPages(
			mock.GetReposBranchesByOwnerByRepo,
			[]github.Branch{
				{
					Name: github.String("branch-1"),
				},
				{
					Name: github.String("branch-2"),
				},
			},
			[]github.Branch{
				{
					Name: github.String("branch-3"),
				},
				{
					Name: github.String("branch-4"),
				},
			},
			[]github.Branch{},
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	branches, err := m.GetBranches("testorg", "testrepo")

	assert.NoError(t, err)
	assert.Equal(t, 4, len(branches))

	assert.Equal(t, "branch-1", *branches[0].Name)
	assert.Equal(t, "branch-2", *branches[1].Name)
	assert.Equal(t, "branch-3", *branches[2].Name)
	assert.Equal(t, "branch-4", *branches[3].Name)
}

func TestGetBranches_ExceedPageSanityCheck(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchPages(
			mock.GetReposBranchesByOwnerByRepo,
			[]github.Branch{{Name: github.String("branch-1")}},
			[]github.Branch{{Name: github.String("branch-2")}},
			[]github.Branch{{Name: github.String("branch-3")}},
			[]github.Branch{{Name: github.String("branch-4")}},
			[]github.Branch{{Name: github.String("branch-5")}},
			[]github.Branch{{Name: github.String("branch-6")}},
			[]github.Branch{{Name: github.String("branch-7")}},
			[]github.Branch{{Name: github.String("branch-8")}},
			[]github.Branch{{Name: github.String("branch-9")}},
			[]github.Branch{{Name: github.String("branch-10")}},
			[]github.Branch{{Name: github.String("branch-11")}},
			[]github.Branch{{Name: github.String("branch-12")}},
			[]github.Branch{{Name: github.String("branch-13")}},
			[]github.Branch{},
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	branches, err := m.GetBranches("testorg", "testrepo")

	assert.NoError(t, err)
	assert.Equal(t, 10, len(branches))

	assert.Equal(t, "branch-1", *branches[0].Name)
	assert.Equal(t, "branch-2", *branches[1].Name)
	assert.Equal(t, "branch-3", *branches[2].Name)
	assert.Equal(t, "branch-4", *branches[3].Name)
	assert.Equal(t, "branch-5", *branches[4].Name)
	assert.Equal(t, "branch-6", *branches[5].Name)
	assert.Equal(t, "branch-7", *branches[6].Name)
	assert.Equal(t, "branch-8", *branches[7].Name)
	assert.Equal(t, "branch-9", *branches[8].Name)
	assert.Equal(t, "branch-10", *branches[9].Name)
}

func TestGetBranches_APIError(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.GetReposBranchesByOwnerByRepo,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(
					w,
					"github api error",
					http.StatusInternalServerError,
				)
			}),
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	branches, err := m.GetBranches("testorg", "testrepo")

	assert.Error(t, err)
	assert.Nil(t, branches)
}

func TestGetReleases_MultiplePages(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchPages(
			mock.GetReposReleasesByOwnerByRepo,
			[]github.RepositoryRelease{
				{
					Name: github.String("release-1"),
				},
				{
					Name: github.String("release-2"),
				},
			},
			[]github.RepositoryRelease{
				{
					Name: github.String("release-3"),
				},
				{
					Name: github.String("release-4"),
				},
			},
			[]github.RepositoryRelease{},
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	releases, err := m.GetReleases("testorg", "testrepo")

	assert.NoError(t, err)
	assert.Equal(t, 4, len(releases))

	assert.Equal(t, "release-1", *releases[0].Name)
	assert.Equal(t, "release-2", *releases[1].Name)
	assert.Equal(t, "release-3", *releases[2].Name)
	assert.Equal(t, "release-4", *releases[3].Name)
}

func TestGetReleases_ExceedPageSanityCheck(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchPages(
			mock.GetReposReleasesByOwnerByRepo,
			[]github.RepositoryRelease{{Name: github.String("release-1")}},
			[]github.RepositoryRelease{{Name: github.String("release-2")}},
			[]github.RepositoryRelease{{Name: github.String("release-3")}},
			[]github.RepositoryRelease{{Name: github.String("release-4")}},
			[]github.RepositoryRelease{{Name: github.String("release-5")}},
			[]github.RepositoryRelease{{Name: github.String("release-6")}},
			[]github.RepositoryRelease{{Name: github.String("release-7")}},
			[]github.RepositoryRelease{{Name: github.String("release-8")}},
			[]github.RepositoryRelease{{Name: github.String("release-9")}},
			[]github.RepositoryRelease{{Name: github.String("release-10")}},
			[]github.RepositoryRelease{{Name: github.String("release-11")}},
			[]github.RepositoryRelease{{Name: github.String("release-12")}},
			[]github.RepositoryRelease{{Name: github.String("release-13")}},
			[]github.RepositoryRelease{},
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	releases, err := m.GetReleases("testorg", "testrepo")

	assert.NoError(t, err)
	assert.Equal(t, 10, len(releases))

	assert.Equal(t, "release-1", *releases[0].Name)
	assert.Equal(t, "release-2", *releases[1].Name)
	assert.Equal(t, "release-3", *releases[2].Name)
	assert.Equal(t, "release-4", *releases[3].Name)
	assert.Equal(t, "release-5", *releases[4].Name)
	assert.Equal(t, "release-6", *releases[5].Name)
	assert.Equal(t, "release-7", *releases[6].Name)
	assert.Equal(t, "release-8", *releases[7].Name)
	assert.Equal(t, "release-9", *releases[8].Name)
	assert.Equal(t, "release-10", *releases[9].Name)
}

func TestGetReleases_APIError(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.GetReposReleasesByOwnerByRepo,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(
					w,
					"github api error",
					http.StatusInternalServerError,
				)
			}),
		),
	)

	c := github.NewClient(mockedHTTPClient)
	m := RepositoryDataCollector{GitHubClient: c}

	releases, err := m.GetReleases("testorg", "testrepo")

	assert.Error(t, err)
	assert.Nil(t, releases)
}

func TestExtractLastChangeTS(t *testing.T) {
	oneHourAgo := time.Now().Add(time.Hour * -1)
	twoHourAgo := time.Now().Add(time.Hour * -2)
	threeHourAgo := time.Now().Add(time.Hour * -3)

	tests := []struct {
		name      string
		createdAt *time.Time
		pushedAt  *time.Time
		updatedAt *time.Time
		expected  *time.Time
	}{
		{"all nil", nil, nil, nil, nil},
		{"created wins", &oneHourAgo, &twoHourAgo, &threeHourAgo, &oneHourAgo},
		{"update wins", &twoHourAgo, &threeHourAgo, &oneHourAgo, &oneHourAgo},
		{"pushed wins", &threeHourAgo, &oneHourAgo, &twoHourAgo, &oneHourAgo},
		{"missing one value", nil, &twoHourAgo, &threeHourAgo, &twoHourAgo},
		{"missing several values", nil, nil, &threeHourAgo, &threeHourAgo},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			repo := github.Repository{}
			if tt.createdAt != nil {
				repo.CreatedAt = &github.Timestamp{Time: *tt.createdAt}
			}
			if tt.pushedAt != nil {
				repo.PushedAt = &github.Timestamp{Time: *tt.pushedAt}
			}
			if tt.updatedAt != nil {
				repo.UpdatedAt = &github.Timestamp{Time: *tt.updatedAt}
			}

			changed := extractLastChangeTS(&repo)

			assert.Equal(t, tt.expected, changed)
		})
	}
}

func TestDedupAndMerge(t *testing.T) {
	var repos []Repository

	// append array empty
	repos = dedupAndMerge(repos, []Repository{})
	assert.Equal(t, 0, len(repos))

	// none existing
	repos = dedupAndMerge(repos, []Repository{
		{ID: int64(123)},
		{ID: int64(124)},
	})
	assert.Equal(t, 2, len(repos))
	assert.Equal(t, int64(123), repos[0].ID)
	assert.Equal(t, int64(124), repos[1].ID)

	// some existing
	repos = dedupAndMerge(repos, []Repository{
		{ID: int64(234)},
		{ID: int64(124)},
		{ID: int64(235)},
	})
	assert.Equal(t, 4, len(repos))
	assert.Equal(t, int64(123), repos[0].ID)
	assert.Equal(t, int64(124), repos[1].ID)
	assert.Equal(t, int64(234), repos[2].ID)
	assert.Equal(t, int64(235), repos[3].ID)

	// all existing
	repos = dedupAndMerge(repos, []Repository{
		{ID: int64(234)},
		{ID: int64(123)},
		{ID: int64(235)},
	})
	assert.Equal(t, 4, len(repos))
	assert.Equal(t, int64(123), repos[0].ID)
	assert.Equal(t, int64(124), repos[1].ID)
	assert.Equal(t, int64(234), repos[2].ID)
	assert.Equal(t, int64(235), repos[3].ID)
}
