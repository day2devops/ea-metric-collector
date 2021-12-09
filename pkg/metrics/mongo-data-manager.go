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
	collection, err := mdm.collection("metrics")
	if err != nil {
		return err
	}

	filter := bson.M{"org": metrics.Org, "repositoryName": metrics.RepositoryName}
	_, err = collection.ReplaceOne(
		context.Background(), filter, metrics, &options.ReplaceOptions{Upsert: &[]bool{true}[0]})
	return err
}

// ReadMetrics Read the metrics for supplied repository
func (mdm MongoDataManager) ReadMetrics(org string, repo string) (found bool, metric *GitRepositoryMetric, err error) {
	glog.V(2).Infof("Reading metric data for repository %s/%s from mongo", org, repo)
	collection, err := mdm.collection("metrics")
	if err != nil {
		return false, nil, err
	}

	metric = &GitRepositoryMetric{}
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
	_, err := mdm.collection("metrics")
	if err != nil {
		return nil, err
	}

	var allKeys []Key
	// filter := bson.M{"org": opts.orgFilter.String(), "repositoryName": repo}

	return allKeys, nil
}

// StoreCacheStats store the statistics for overall cache statistics
func (mdm MongoDataManager) StoreCacheStats(org string, stats CacheStats) {
	glog.V(2).Infof("Writing cache stats to mongo for org %s", org)
	collection, err := mdm.collection("stats")
	if err != nil {
		glog.Warning("Unable to connect to mongo collection: ", err)
		return
	}

	stats.Org = org
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
	collection, err := mdm.collection("stats")
	if err != nil {
		glog.Warning("Unable to connect to mongo collection: ", err)
		return false, nil
	}

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

// get connection to collection with supplied name
func (mdm MongoDataManager) collection(collection string) (*mongo.Collection, error) {
	client, err := mdm.connect()
	if err != nil {
		return nil, err
	}
	return client.Database("devops_metrics").Collection(collection), nil
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
		clientOps := options.Client().ApplyURI(mdm.ConnectionString).SetAuth(creds)
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
