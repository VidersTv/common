package mongo

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/common/instance"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var ErrNoDocuments = mongo.ErrNoDocuments

func New(ctx context.Context, opt SetupOptions) (instance.Mongo, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(opt.URI).SetDirect(opt.Direct))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	database := client.Database(opt.Database)

	logrus.Info("mongo, ok")

	return &MongoInst{
		client: client,
		db:     database,
	}, nil
}

type SetupOptions struct {
	URI      string
	Database string
	Direct   bool
}

type (
	Pipeline       = mongo.Pipeline
	WriteModel     = mongo.WriteModel
	InsertOneModel = mongo.InsertOneModel
	UpdateOneModel = mongo.UpdateOneModel
	IndexModel     = mongo.IndexModel
)
