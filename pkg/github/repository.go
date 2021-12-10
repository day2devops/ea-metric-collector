package github

import (
	"context"
	"errors"
	"time"

	"github.com/golang/glog"
	gogithub "github.com/google/go-github/v39/github"
	"golang.org/x/sync/errgroup"
)

// Repository represents the minimal repository identifiers
type Repository struct {
	ID           int64
	Org          string
	Name         string
	Topics       []string
	Changed      *time.Time
	Detail       *gogithub.Repository
	Branches     []*gogithub.Branch
	Releases     []*gogithub.RepositoryRelease
	PullRequests []*gogithub.PullRequest
	Languages    map[string]int
}

// DataCollector defines methods for repository management
type DataCollector interface {
	ListRepositories(org string, changedAfter *time.Time) ([]Repository, error)
	GetRepository(org string, name string) (*Repository, error)
	GetBranches(org string, repo string) ([]*gogithub.Branch, error)
	GetReleases(org string, repo string) ([]*gogithub.RepositoryRelease, error)
}

// RepositoryDataCollector used to collect data from git hub repositories
type RepositoryDataCollector struct {
	GitHubClient *gogithub.Client
}

// ListRepositories retrieves the set of repositories for an organization
func (m RepositoryDataCollector) ListRepositories(org string, changedAfter *time.Time) ([]Repository, error) {
	// if no change after timestamp supplied, just return all repos using created sort
	if changedAfter == nil {
		glog.Infof("Collecting all repositories for org %s", org)
		return m.listByOrgAndSort(org, "created", nil)
	}

	// changed after timestamp has been supplied.  GitHub API can list repositories by created, updated, or pushed
	// timestamps but unfortunately can't list by the lastest of those timestamps.  To simulate that, we will merge
	// the results of getting repositories using each of those sort options.
	glog.Infof("Collecting repositories for org %s UPDATED after %s", org, changedAfter)
	repos, err := m.listByOrgAndSort(org, "updated", changedAfter)
	if err != nil {
		return nil, err
	}

	glog.Infof("Collecting repositories for org %s PUSHED after %s", org, changedAfter)
	pushedRepos, err := m.listByOrgAndSort(org, "pushed", changedAfter)
	if err != nil {
		return nil, err
	}
	repos = dedupAndMerge(repos, pushedRepos)

	glog.Infof("Collecting repositories for org %s CREATED after %s", org, changedAfter)
	createdRepos, err := m.listByOrgAndSort(org, "created", changedAfter)
	if err != nil {
		return nil, err
	}
	repos = dedupAndMerge(repos, createdRepos)

	return repos, nil
}

// GetRepository retrieves the repository information by organization/name
func (m RepositoryDataCollector) GetRepository(org string, name string) (*Repository, error) {
	// retrieve data from the github using 3 goroutines to...
	//  a. pull the base repository data
	//  b. pull branch information
	//  c. pull release information
	grp, ctx := errgroup.WithContext(context.Background())

	var ghRepo *gogithub.Repository
	grp.Go(func() error {
		r, _, err := m.GitHubClient.Repositories.Get(ctx, org, name)
		if err == nil {
			ghRepo = r
		}
		return err
	})

	var branches []*gogithub.Branch
	grp.Go(func() error {
		b, err := m.GetBranches(org, name)
		if err == nil {
			branches = b
		}
		return err
	})

	var releases []*gogithub.RepositoryRelease
	grp.Go(func() error {
		r, err := m.GetReleases(org, name)
		if err == nil {
			releases = r
		}
		return err
	})

	var pullRequests []*gogithub.PullRequest
	grp.Go(func() error {
		p, err := m.GetPullRequests(org, name)
		if err == nil {
			pullRequests = p
		}
		return err
	})

	var languages map[string]int
	grp.Go(func() error {
		l, err := m.GetLanguages(org, name)
		if err == nil {
			languages = l
		}
		return err
	})

	var topics []string
	grp.Go(func() error {
		t, err := m.GetTopics(org, name)
		if err == nil {
			topics = t
		}
		return err
	})

	if err := grp.Wait(); err != nil {
		return nil, err
	}

	// Build repository output
	return &Repository{
		ID:           *ghRepo.ID,
		Org:          org,
		Name:         name,
		Topics:       topics,
		Changed:      extractLastChangeTS(ghRepo),
		Detail:       ghRepo,
		Branches:     branches,
		Releases:     releases,
		PullRequests: pullRequests,
		Languages:    languages,
	}, nil
}

// GetBranches retrieves branch information by organization/repo
func (m RepositoryDataCollector) GetBranches(org string, repo string) ([]*gogithub.Branch, error) {
	// build context and options for branch call...maximum of 100
	// branches per page so going with that for now
	ctx := context.Background()
	opt := &gogithub.BranchListOptions{
		ListOptions: gogithub.ListOptions{PerPage: 100},
	}

	// process all pages until finished
	var loopCnt = 0
	var allBranches []*gogithub.Branch
	for {
		// sanity check to stop looking for branches if we hit 1000 (10 pages)
		loopCnt++
		if loopCnt > 10 {
			glog.Warningf("Repository has more than 1000 branches: %s/%s", org, repo)
			break
		}

		glog.V(2).Infof("Collecting branches for %s/%s, count per page = %d, page number = %d", org, repo, opt.ListOptions.PerPage, opt.Page)
		branches, resp, err := m.GitHubClient.Repositories.ListBranches(ctx, org, repo, opt)
		if err != nil {
			return nil, err
		}

		if glog.V(3) {
			glog.Infof("Branches found: %d", len(branches))
			for _, branch := range branches {
				glog.Infof("Branch found: %s", *branch.Name)
			}
		}

		allBranches = append(allBranches, branches...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return allBranches, nil
}

// GetReleases retrieves release information by organization/repo
func (m RepositoryDataCollector) GetReleases(org string, repo string) ([]*gogithub.RepositoryRelease, error) {
	// build context and options for release call...maximum of 100
	// releases per page so going with that for now
	ctx := context.Background()
	opt := &gogithub.ListOptions{PerPage: 100}

	// process all pages until finished
	var loopCnt = 0
	var allReleases []*gogithub.RepositoryRelease
	for {
		// sanity check to stop looking for branches if we hit 1000 (10 pages)
		loopCnt++
		if loopCnt > 10 {
			glog.Warningf("Repository has more than 1000 releases: %s/%s", org, repo)
			break
		}

		glog.V(2).Infof("Collecting releases for %s/%s, count per page = %d, page number = %d", org, repo, opt.PerPage, opt.Page)
		releases, resp, err := m.GitHubClient.Repositories.ListReleases(ctx, org, repo, opt)
		if err != nil {
			return nil, err
		}

		if glog.V(3) {
			glog.Infof("Releases found: %d", len(releases))
			for _, release := range releases {
				glog.Infof("Release found: %s", *release.Name)
			}
		}

		allReleases = append(allReleases, releases...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return allReleases, nil
}

// GetPullRequests retrieves pull requests by organization/repo
func (m RepositoryDataCollector) GetPullRequests(org string, repo string) ([]*gogithub.PullRequest, error) {
	// build context and options for release call...maximum of 100
	// pull requests per page so going with that for now
	ctx := context.Background()
	opt := &gogithub.PullRequestListOptions{
		State:       "all",
		ListOptions: gogithub.ListOptions{PerPage: 100},
	}

	// process all pages until finished
	var loopCnt = 0
	var allPullRequests []*gogithub.PullRequest
	for {
		// sanity check to stop looking for requests if we hit 1000 (10 pages)
		loopCnt++
		if loopCnt > 10 {
			glog.Warningf("Repository has more than 1000 pull requests: %s/%s", org, repo)
			break
		}

		glog.V(2).Infof("Collecting pull requests for %s/%s, count per page = %d, page number = %d", org, repo, opt.PerPage, opt.Page)
		pullRequests, resp, err := m.GitHubClient.PullRequests.List(ctx, org, repo, opt)
		if err != nil {
			return nil, err
		}

		if glog.V(3) {
			glog.Infof("Pull requests found: %d", len(pullRequests))
			for _, pr := range pullRequests {
				glog.Infof("Pull request found: %s", *pr.Title)
			}
		}

		allPullRequests = append(allPullRequests, pullRequests...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return allPullRequests, nil
}

// GetLanguages retrieves language usage by organization/repo
func (m RepositoryDataCollector) GetLanguages(org string, repo string) (map[string]int, error) {
	ctx := context.Background()
	glog.V(2).Infof("Collecting languages for %s/%s", org, repo)
	languages, _, err := m.GitHubClient.Repositories.ListLanguages(ctx, org, repo)
	if err != nil {
		return nil, err
	}
	return languages, nil
}

// GetTopics retrieves topics by organization/repo
func (m RepositoryDataCollector) GetTopics(org string, repo string) ([]string, error) {
	ctx := context.Background()
	glog.V(2).Infof("Collecting topics for %s/%s", org, repo)
	topics, _, err := m.GitHubClient.Repositories.ListAllTopics(ctx, org, repo)
	if err != nil {
		return nil, err
	}
	return topics, nil
}

// find repositories change after supplied time using supplied sort (updated, pushed, or created)
func (m RepositoryDataCollector) listByOrgAndSort(org string, sort string, changedAfter *time.Time) ([]Repository, error) {
	// build context and options for repository call...maximum of 100
	// repositories per page so going with that for now
	ctx := context.Background()
	opt := &gogithub.RepositoryListByOrgOptions{
		Sort:        sort,
		Direction:   "desc",
		ListOptions: gogithub.ListOptions{Page: 1, PerPage: 100},
	}

	// process all pages until finished
	var loopCnt = 0
	var allRepos []Repository
	for {
		// sanity check the paging loop to prevent infinite loop and spamming of github api
		loopCnt++
		if loopCnt > 1000 {
			return nil, errors.New("repository loop exceeded sanity check")
		}

		if !glog.V(2) {
			glog.Infof("...")
		}
		glog.V(2).Infof("Collecting repositories using sort %s, count per page = %d, page number = %d", sort, opt.ListOptions.PerPage, opt.Page)

		repos, resp, err := m.GitHubClient.Repositories.ListByOrg(ctx, org, opt)
		if err != nil {
			return nil, err
		}

		if glog.V(2) {
			glog.Infof("Repositories found: %d", len(repos))
			for _, repo := range repos {
				glog.Infof("Repository found: %s, changed %s", *repo.Name, extractLastChangeTS(repo))
			}
		}

		for _, repository := range repos {
			// done if change after time supplied and this is repo was last changed before it
			changed := extractLastChangeTS(repository)
			if changedAfter != nil && changed != nil && changed.Before(*changedAfter) {
				return allRepos, nil
			}

			allRepos = append(allRepos, Repository{
				ID:      *repository.ID,
				Org:     org,
				Name:    *repository.Name,
				Changed: changed,
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return allRepos, nil
}

// extract the latest modification timestamp for the repository
func extractLastChangeTS(r *gogithub.Repository) *time.Time {
	return latest(
		extractTime(r.CreatedAt),
		extractTime(r.PushedAt),
		extractTime(r.UpdatedAt))
}

// select the latest of the supplied times
func latest(times ...*time.Time) *time.Time {
	var latest *time.Time
	for _, t := range times {
		if latest == nil {
			latest = t
			continue
		}
		if t == nil || latest.After(*t) {
			continue
		}
		latest = t
	}
	return latest
}

// extract time reference if supplied, otherwise default to nil
func extractTime(t *gogithub.Timestamp) *time.Time {
	if t != nil {
		return &t.Time
	}
	return nil
}

// append additions (based on repository ID) not already in the original slice
func dedupAndMerge(orig []Repository, additions []Repository) []Repository {
	repos := orig
	for _, a := range additions {
		add := true
		for _, r := range repos {
			if a.ID == r.ID {
				add = false
				break
			}
		}

		if add {
			repos = append(repos, a)
		}
	}
	return repos
}
