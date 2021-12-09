package cmd

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/day2devops/ea-metric-extractor/pkg/github"
	"github.com/day2devops/ea-metric-extractor/pkg/metrics"
)

const (
	updateMetricsExample = `  # Update the metrics for all GitHub repositories on default organization changed since last update
  git-what update-metrics
  
  # Update metrics for ALL GitHub repositories regardless of last update
  git-what update-metrics --forceUpdate

  # Update metrics for ALL GitHub repositories on specified organization
  git-what update-metrics --org <org>

  # Update the metrics for a particular GitHub repository on default organization
  git-what update-metrics --repo <repo>

  # Update the metrics for a particular GitHub repository on specified organization
  git-what update-metrics --org <org> --repo <repo>

  # Override the base github url and data directories
  git-what update-metrics --baseURL <baseURL> --dataDir <dataDir>
  `
)

// UpdateMetricsCommand the update metric command structure
type UpdateMetricsCommand struct {
	baseURL             string
	org                 string
	dataDir             string
	repo                string
	forceUpdate         bool
	forceEvalAll        bool
	mongo               bool
	gitHubClientFactory github.ClientCreator
	processorFactory    metrics.ProcessorCreator
}

// returns a new initialized instance of the update-metrics sub command
func newUpdateMetricsCmd() (*cobra.Command, *UpdateMetricsCommand) {
	umc := UpdateMetricsCommand{
		gitHubClientFactory: github.ClientFactory{},
		processorFactory:    metrics.ProcessorFactory{},
	}

	updateMetricsCmd := &cobra.Command{
		Use:     "update-metrics",
		Short:   "Update metrics for repositories.",
		Long:    `Update all metrics for repositories from GitHub.`,
		Example: updateMetricsExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return umc.UpdateMetricsCmd()
		},
	}

	updateMetricsCmd.Flags().StringVar(&umc.baseURL, "baseURL", "https://api.github.com/", "Override the default base url")
	updateMetricsCmd.Flags().StringVar(&umc.org, "org", "day2devops", "Override the default organization of repositories")
	updateMetricsCmd.Flags().StringVar(&umc.repo, "repo", "", "Restrict update to the supplied repository name")
	updateMetricsCmd.Flags().StringVar(&umc.dataDir, "dataDir", defaultDataDir(os.UserHomeDir), "Override the default data directory")
	updateMetricsCmd.Flags().BoolVar(&umc.forceUpdate, "forceUpdate", false, "Force updates of repositories regardless of last update timestamp")
	updateMetricsCmd.Flags().BoolVar(&umc.forceEvalAll, "forceEvalAll", false, "Force evaluation of all repositories regardless of cache statistics")
	updateMetricsCmd.Flags().BoolVar(&umc.mongo, "mongo", false, "Leverage mongodb for metric persistence")
	return updateMetricsCmd, &umc
}

// UpdateMetricsCmd performs the update-metrics sub command
func (umc UpdateMetricsCommand) UpdateMetricsCmd() error {
	// establish a client for the GitHub API interactions
	token, err := githubToken()
	if err != nil {
		return err
	}

	glog.V(2).Infof("Building github client with base url: %s, token: %s", umc.baseURL, token)

	client, err := umc.gitHubClientFactory.NewGitHubClient(umc.baseURL, token)
	if err != nil {
		return err
	}

	// build metric manager and execute based on command args
	glog.V(2).Infof("Using metric data directory: %s", umc.dataDir)

	var dataMgr metrics.DataManager
	dataMgr = metrics.FileDataManager{DataDir: umc.dataDir}
	if umc.mongo {
		user, pwd, conn, err := mongoConnectionInfo()
		if err != nil {
			return err
		}
		dataMgr = metrics.MongoDataManager{User: user, Pwd: pwd, ConnectionString: conn}
	}

	dataCollector := github.RepositoryDataCollector{GitHubClient: client}

	processor := umc.processorFactory.NewProcessor(dataCollector, dataMgr)

	if umc.repo == "" {
		return processor.RepositoriesForOrg(umc.org, metrics.Options{
			ForceMetricUpdate: umc.forceUpdate,
			ForceAllRepoEval:  umc.forceEvalAll || umc.forceUpdate,
		})
	}
	return processor.Repository(umc.org, umc.repo)
}

// retrieves authorization token for GitHub for process
func githubToken() (string, error) {
	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		return "", errors.New("GitHub token not specified")
	}
	return token, nil
}

// build the default directory to hold github metric data
func defaultDataDir(userHomeDir func() (string, error)) string {
	userDir, err := userHomeDir()
	if err != nil {
		glog.Warning("Unable to find user home directory, defaulting to current directory")
		userDir = "."
	}
	return filepath.Join(userDir, ".git-metrics")
}

// retrieves mongo authorization items
func mongoConnectionInfo() (user string, pwd string, connection string, err error) {
	user = os.Getenv("MONGO_USER")
	pwd = os.Getenv("MONGO_PWD")
	connection = os.Getenv("MONGO_CONN")
	if user == "" || pwd == "" || connection == "" {
		return "", "", "", errors.New("mongo user/pwd/connection not specified")
	}
	return
}
