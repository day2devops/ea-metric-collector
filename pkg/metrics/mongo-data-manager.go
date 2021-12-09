package metrics

import (
	"context"
	"time"

	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDataManager mongo based implementation of MetricDataManager
type MongoDataManager struct {
	User             string
	Pwd              string
	ConnectionString string
	client           *mongo.Client
}

// StoreMetrics Persist the supplied metrics
func (mdm MongoDataManager) StoreMetrics(metrics GitRepositoryMetric) error {
	glog.V(2).Infof("Writing metric data for repository %s", metrics.RepositoryName)
	client, err := mdm.connect()
	if err != nil {
		glog.Warning("Unable to connect to mongo", err)
	}

	collection := client.Database("devops_metrics").Collection("metrics")
	filter := bson.M{"org": metrics.Org, "repositoryName": metrics.RepositoryName}
	_, err = collection.ReplaceOne(
		context.Background(), filter, metrics, &options.ReplaceOptions{Upsert: &[]bool{true}[0]})
	return err
}

// ReadMetrics Read the metrics for supplied repository
func (mdm MongoDataManager) ReadMetrics(org string, repo string) (found bool, metric *GitRepositoryMetric, err error) {
	glog.V(2).Infof("Reading metric data for repository %s/%s from mongo", org, repo)
	client, err := mdm.connect()
	if err != nil {
		glog.Warning("Unable to connect to mongo", err)
	}

	metric = &GitRepositoryMetric{}
	collection := client.Database("devops_metrics").Collection("metrics")
	filter := bson.M{"org": org, "repositoryName": repo}
	if err = collection.FindOne(context.Background(), filter).Decode(metric); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil, nil
		}
		glog.Warningf("Unable to read stats from mongo for org %s: %v", org, err)
		return false, nil, err
	}
	found = true
	return
}

// DeleteMetrics Delete the metrics for the supplied repository
func (mdm MongoDataManager) DeleteMetrics(org string, repo string) error {
	glog.V(2).Infof("Deleting metric data for repository %s/%s from mongo", org, repo)
	return nil
}

// ListMetrics List the known repositories with metrics that match the supplied options
func (mdm MongoDataManager) ListMetrics(opts ListMetricOptions) ([]Key, error) {
	glog.V(2).Infof("Listing repository metrics found in mongo")
	client, err := mdm.connect()
	if err != nil {
		glog.Warning("Unable to connect to mongo", err)
	}

	var allKeys []Key

	client.Database("devops_metrics").Collection("metrics")
	// filter := bson.M{"org": opts.orgFilter.String(), "repositoryName": repo}

	return allKeys, nil
}

// StoreCacheStats store the statistics for overall cache statistics
func (mdm MongoDataManager) StoreCacheStats(org string, stats CacheStats) {
	glog.V(2).Infof("Writing cache stats to mongo for org %s", org)
	client, err := mdm.connect()
	if err != nil {
		glog.Warning("Unable to connect to mongo", err)
	}

	stats.Org = org
	collection := client.Database("devops_metrics").Collection("stats")
	filter := bson.M{"org": org}
	_, err = collection.ReplaceOne(
		context.Background(), filter, stats, &options.ReplaceOptions{Upsert: &[]bool{true}[0]})
	if err != nil {
		glog.Warning("Unable to store cache stats in mongo", err)
	}
}

// ReadCacheStats read the overall cache statistics
func (mdm MongoDataManager) ReadCacheStats(org string) (found bool, stats *CacheStats) {
	glog.V(2).Infof("Reading cache stats from mongo")
	client, err := mdm.connect()
	if err != nil {
		glog.Warning("Unable to connect to mongo", err)
	}

	collection := client.Database("devops_metrics").Collection("stats")
	filter := bson.M{"org": org}
	stats = &CacheStats{}
	if err = collection.FindOne(context.Background(), filter).Decode(stats); err != nil {
		glog.Warningf("Unable to read stats from mongo for org %s: %v", org, err)
		return false, nil
	}

	glog.V(3).Infof("Cache Stats found: %v", *stats)
	found = true
	return
}

// connect get connection to mongo database
func (mdm MongoDataManager) connect() (*mongo.Client, error) {
	if mdm.client == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		creds := options.Credential{
			AuthMechanism: "SCRAM-SHA-1",
			Username:      mdm.User,
			Password:      mdm.Pwd,
		}
		clientOps := options.Client().ApplyURI(mdm.ConnectionString)
		clientOps.Auth = &creds
		client, err := mongo.Connect(ctx, clientOps)
		if err != nil {
			return nil, err
		}
		err = client.Ping(ctx, nil)
		if err != nil {
			return nil, err
		}
		mdm.client = client
	}
	return mdm.client, nil
}
