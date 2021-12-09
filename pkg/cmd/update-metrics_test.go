package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogithub "github.com/google/go-github/v39/github"
	"github.com/nyarly/spies"
	"github.com/stretchr/testify/assert"

	"github.com/day2devops/ea-metric-extractor/pkg/github"
	"github.com/day2devops/ea-metric-extractor/pkg/metrics"
)

func TestGithubToken_EnvVariableSet(t *testing.T) {
	token := "authtokenval"
	os.Setenv("GITHUB_AUTH_TOKEN", token)
	defer os.Unsetenv("GITHUB_AUTH_TOKEN")

	oToken, err := githubToken()
	assert.Equal(t, token, oToken)
	assert.NoError(t, err)
}

func TestGithubToken_EnvVariableNotSet(t *testing.T) {
	oToken, err := githubToken()
	assert.Equal(t, "", oToken)
	assert.Error(t, err)
}

func TestDefaultDataDir_UserHomeNoError(t *testing.T) {
	userDir := "/home/user"

	userHomeDir := func() (string, error) {
		return userDir, nil
	}

	assert.Equal(t, filepath.Join(userDir, ".git-metrics"), defaultDataDir(userHomeDir))
}

func TestDefaultDataDir_UserHomeError(t *testing.T) {
	userHomeDir := func() (string, error) {
		return "", errors.New("no user home")
	}

	assert.Equal(t, filepath.Join(".", ".git-metrics"), defaultDataDir(userHomeDir))
}

func TestNewUpdateMetricsCmd_FlagDefaults(t *testing.T) {
	cmd, umc := newUpdateMetricsCmd()
	cmd.ParseFlags([]string{})

	assert.Equal(t, "https://github.com/", umc.baseURL)
	assert.True(t, strings.Contains(umc.dataDir, ".git-metrics"))
	assert.Equal(t, "day2devops", umc.org)
	assert.Equal(t, "", umc.repo)
	assert.False(t, umc.forceUpdate)
	assert.False(t, umc.forceEvalAll)
	assert.NotNil(t, umc.gitHubClientFactory)
	assert.NotNil(t, umc.processorFactory)
}

func TestNewUpdateMetricsCmd_FlagOverrides(t *testing.T) {
	cmd, umc := newUpdateMetricsCmd()
	cmd.ParseFlags([]string{
		"--baseURL", "https://mygithub.com/",
		"--org", "myorg",
		"--dataDir", "./.mydatadir",
		"--repo", "myrepo",
		"--forceUpdate",
		"--forceEvalAll",
	})

	assert.Equal(t, "https://mygithub.com/", umc.baseURL)
	assert.Equal(t, "./.mydatadir", umc.dataDir)
	assert.Equal(t, "myorg", umc.org)
	assert.Equal(t, "myrepo", umc.repo)
	assert.True(t, umc.forceUpdate)
	assert.True(t, umc.forceEvalAll)
	assert.NotNil(t, umc.gitHubClientFactory)
	assert.NotNil(t, umc.processorFactory)
}

func TestUpdateMetricsCmd_NoGitHubToken(t *testing.T) {
	cmd := UpdateMetricsCommand{}
	err := cmd.UpdateMetricsCmd()

	assert.Error(t, err)
	if err != nil {
		assert.Equal(t, "GitHub token not specified", err.Error())
	}
}

func TestUpdateMetricsCmd_GitHubClientError(t *testing.T) {
	// Define spy function for github client factory
	ghcSpy := &GitHubClientFactorySpy{Spy: spies.NewSpy()}
	ghcSpy.MatchMethod("NewGitHubClient", spies.AnyArgs, nil, errors.New("test error with gh client"))

	// Establish token in environment for test
	token := "authtokenval-githubclienterror"
	os.Setenv("GITHUB_AUTH_TOKEN", token)
	defer os.Unsetenv("GITHUB_AUTH_TOKEN")

	// Build and execute command
	cmd := UpdateMetricsCommand{
		gitHubClientFactory: ghcSpy,
	}
	err := cmd.UpdateMetricsCmd()

	assert.Error(t, err)
	if err != nil {
		assert.Equal(t, "test error with gh client", err.Error())
	}
}

func TestUpdateMetricsCmd_AllRepositories(t *testing.T) {
	// Define spy for github client factory
	ghcSpy := &GitHubClientFactorySpy{Spy: spies.NewSpy()}
	oGitHubClient := &gogithub.Client{}
	ghcSpy.MatchMethod("NewGitHubClient", spies.AnyArgs, oGitHubClient, nil)

	// Define spy for metrics processor and factory
	mpSpy := &MetricsProcessorSpy{Spy: spies.NewSpy()}
	mpSpy.MatchMethod("RepositoriesForOrg", spies.AnyArgs, nil)

	mpfSpy := &MetricsProcessorFactorySpy{Spy: spies.NewSpy()}
	mpfSpy.MatchMethod("NewProcessor", spies.AnyArgs, mpSpy)

	// Establish token in environment for test
	token := "authtokenval-allrepositories"
	os.Setenv("GITHUB_AUTH_TOKEN", token)
	defer os.Unsetenv("GITHUB_AUTH_TOKEN")

	// Build and execute command
	cmd := UpdateMetricsCommand{
		baseURL:             "https://testgithub.edwardjones.com/",
		org:                 "testorg",
		dataDir:             ".",
		forceUpdate:         false,
		forceEvalAll:        true,
		gitHubClientFactory: ghcSpy,
		processorFactory:    mpfSpy,
	}
	err := cmd.UpdateMetricsCmd()

	assert.NoError(t, err)

	assert.Equal(t, "https://testgithub.edwardjones.com/", ghcSpy.Calls()[0].PassedArgs().Get(0))
	assert.Equal(t, token, ghcSpy.Calls()[0].PassedArgs().Get(1))

	ghdc, ok := mpfSpy.Calls()[0].PassedArgs().Get(0).(github.RepositoryDataCollector)
	if ok {
		assert.Same(t, oGitHubClient, ghdc.GitHubClient)
	}
	fdm, ok := mpfSpy.Calls()[0].PassedArgs().Get(1).(metrics.FileDataManager)
	if ok {
		assert.Equal(t, ".", fdm.DataDir)
	}

	assert.Equal(t, 1, len(mpSpy.CallsTo("RepositoriesForOrg")))
	assert.Equal(t, "testorg", mpSpy.CallsTo("RepositoriesForOrg")[0].PassedArgs().String(0))
	assert.False(t, mpSpy.CallsTo("RepositoriesForOrg")[0].PassedArgs().Get(1).(metrics.Options).ForceMetricUpdate)
	assert.True(t, mpSpy.CallsTo("RepositoriesForOrg")[0].PassedArgs().Get(1).(metrics.Options).ForceAllRepoEval)
	assert.Equal(t, 0, len(mpSpy.CallsTo("Repository")))
}

func TestUpdateMetricsCmd_AllRepositories_Error(t *testing.T) {
	// Define spy for github client factory
	ghcSpy := &GitHubClientFactorySpy{Spy: spies.NewSpy()}
	oGitHubClient := &gogithub.Client{}
	ghcSpy.MatchMethod("NewGitHubClient", spies.AnyArgs, oGitHubClient, nil)

	// Define spy for metrics processor and factory
	mpSpy := &MetricsProcessorSpy{Spy: spies.NewSpy()}
	mpSpy.MatchMethod("RepositoriesForOrg", spies.AnyArgs, errors.New("test error"))

	mpfSpy := &MetricsProcessorFactorySpy{Spy: spies.NewSpy()}
	mpfSpy.MatchMethod("NewProcessor", spies.AnyArgs, mpSpy)

	// Establish token in environment for test
	token := "authtokenval-allrepositories-error"
	os.Setenv("GITHUB_AUTH_TOKEN", token)
	defer os.Unsetenv("GITHUB_AUTH_TOKEN")

	// Build and execute command
	cmd := UpdateMetricsCommand{
		baseURL:             "https://testgithub.edwardjones.com/",
		org:                 "testorg",
		dataDir:             ".",
		forceUpdate:         true,
		gitHubClientFactory: ghcSpy,
		processorFactory:    mpfSpy,
	}
	err := cmd.UpdateMetricsCmd()

	assert.Error(t, err)

	assert.Equal(t, "https://testgithub.edwardjones.com/", ghcSpy.Calls()[0].PassedArgs().Get(0))
	assert.Equal(t, token, ghcSpy.Calls()[0].PassedArgs().Get(1))

	ghdc, ok := mpfSpy.Calls()[0].PassedArgs().Get(0).(github.RepositoryDataCollector)
	if ok {
		assert.Same(t, oGitHubClient, ghdc.GitHubClient)
	}
	fdm, ok := mpfSpy.Calls()[0].PassedArgs().Get(1).(metrics.FileDataManager)
	if ok {
		assert.Equal(t, ".", fdm.DataDir)
	}

	assert.Equal(t, 1, len(mpSpy.CallsTo("RepositoriesForOrg")))
	assert.Equal(t, "testorg", mpSpy.CallsTo("RepositoriesForOrg")[0].PassedArgs().String(0))
	assert.True(t, mpSpy.CallsTo("RepositoriesForOrg")[0].PassedArgs().Get(1).(metrics.Options).ForceMetricUpdate)
	assert.True(t, mpSpy.CallsTo("RepositoriesForOrg")[0].PassedArgs().Get(1).(metrics.Options).ForceAllRepoEval)
	assert.Equal(t, 0, len(mpSpy.CallsTo("Repository")))
}

func TestUpdateMetricsCmd_SpecificRepository(t *testing.T) {
	// Define spy for github client factory
	ghcSpy := &GitHubClientFactorySpy{Spy: spies.NewSpy()}
	oGitHubClient := &gogithub.Client{}
	ghcSpy.MatchMethod("NewGitHubClient", spies.AnyArgs, oGitHubClient, nil)

	// Define spy for metrics processor and factory
	mpSpy := &MetricsProcessorSpy{Spy: spies.NewSpy()}
	mpSpy.MatchMethod("Repository", spies.AnyArgs, nil)

	mpfSpy := &MetricsProcessorFactorySpy{Spy: spies.NewSpy()}
	mpfSpy.MatchMethod("NewProcessor", spies.AnyArgs, mpSpy)

	// Establish token in environment for test
	token := "authtokenval-specificrepository"
	os.Setenv("GITHUB_AUTH_TOKEN", token)
	defer os.Unsetenv("GITHUB_AUTH_TOKEN")

	// Build and execute command
	cmd := UpdateMetricsCommand{
		baseURL:             "https://testgithub.edwardjones.com/",
		org:                 "testorg",
		dataDir:             ".",
		repo:                "test-repo",
		gitHubClientFactory: ghcSpy,
		processorFactory:    mpfSpy,
	}
	err := cmd.UpdateMetricsCmd()

	assert.NoError(t, err)

	assert.Equal(t, "https://testgithub.edwardjones.com/", ghcSpy.Calls()[0].PassedArgs().Get(0))
	assert.Equal(t, token, ghcSpy.Calls()[0].PassedArgs().Get(1))

	ghdc, ok := mpfSpy.Calls()[0].PassedArgs().Get(0).(github.RepositoryDataCollector)
	if ok {
		assert.Same(t, oGitHubClient, ghdc.GitHubClient)
	}
	fdm, ok := mpfSpy.Calls()[0].PassedArgs().Get(1).(metrics.FileDataManager)
	if ok {
		assert.Equal(t, ".", fdm.DataDir)
	}

	assert.Equal(t, 0, len(mpSpy.CallsTo("RepositoriesForOrg")))
	assert.Equal(t, 1, len(mpSpy.CallsTo("Repository")))
	assert.Equal(t, "testorg", mpSpy.CallsTo("Repository")[0].PassedArgs().String(0))
	assert.Equal(t, "test-repo", mpSpy.CallsTo("Repository")[0].PassedArgs().String(1))
}

func TestUpdateMetricsCmd_SpecificRepository_Error(t *testing.T) {
	// Define spy for github client factory
	ghcSpy := &GitHubClientFactorySpy{Spy: spies.NewSpy()}
	oGitHubClient := &gogithub.Client{}
	ghcSpy.MatchMethod("NewGitHubClient", spies.AnyArgs, oGitHubClient, nil)

	// Define spy for metrics processor and factory
	mpSpy := &MetricsProcessorSpy{Spy: spies.NewSpy()}
	mpSpy.MatchMethod("Repository", spies.AnyArgs, errors.New("test error"))

	mpfSpy := &MetricsProcessorFactorySpy{Spy: spies.NewSpy()}
	mpfSpy.MatchMethod("NewProcessor", spies.AnyArgs, mpSpy)

	// Establish token in environment for test
	token := "authtokenval-specificrepository-error"
	os.Setenv("GITHUB_AUTH_TOKEN", token)
	defer os.Unsetenv("GITHUB_AUTH_TOKEN")

	// Build and execute command
	cmd := UpdateMetricsCommand{
		baseURL:             "https://testgithub.edwardjones.com/",
		org:                 "testorg",
		dataDir:             ".",
		repo:                "test-repo",
		gitHubClientFactory: ghcSpy,
		processorFactory:    mpfSpy,
	}
	err := cmd.UpdateMetricsCmd()

	assert.Error(t, err)

	assert.Equal(t, "https://testgithub.edwardjones.com/", ghcSpy.Calls()[0].PassedArgs().Get(0))
	assert.Equal(t, token, ghcSpy.Calls()[0].PassedArgs().Get(1))

	ghdc, ok := mpfSpy.Calls()[0].PassedArgs().Get(0).(github.RepositoryDataCollector)
	if ok {
		assert.Same(t, oGitHubClient, ghdc.GitHubClient)
	}
	fdm, ok := mpfSpy.Calls()[0].PassedArgs().Get(1).(metrics.FileDataManager)
	if ok {
		assert.Equal(t, ".", fdm.DataDir)
	}

	assert.Equal(t, 0, len(mpSpy.CallsTo("RepositoriesForOrg")))
	assert.Equal(t, 1, len(mpSpy.CallsTo("Repository")))
	assert.Equal(t, "testorg", mpSpy.CallsTo("Repository")[0].PassedArgs().String(0))
	assert.Equal(t, "test-repo", mpSpy.CallsTo("Repository")[0].PassedArgs().String(1))
}

type MetricsProcessorSpy struct {
	*spies.Spy
	metrics.Processor
}

func (mps *MetricsProcessorSpy) RepositoriesForOrg(org string, options metrics.Options) error {
	res := mps.Called(org, options)
	return res.Error(0)
}

func (mps *MetricsProcessorSpy) Repository(org string, repo string) error {
	res := mps.Called(org, repo)
	return res.Error(0)
}

type MetricsProcessorFactorySpy struct {
	*spies.Spy
	metrics.ProcessorCreator
}

func (mpfs *MetricsProcessorFactorySpy) NewProcessor(collector github.DataCollector, dataMgr metrics.DataManager) metrics.Processor {
	res := mpfs.Called(collector, dataMgr)
	return res.Get(0).(metrics.Processor)
}

type GitHubClientFactorySpy struct {
	*spies.Spy
	github.ClientCreator
}

func (ghcfs *GitHubClientFactorySpy) NewGitHubClient(baseURL string, token string) (*gogithub.Client, error) {
	res := ghcfs.Called(baseURL, token)
	client := res.Get(0)
	if client == nil {
		return nil, res.Error(1)
	}
	return client.(*gogithub.Client), res.Error(1)
}
