package mongo

import (
	"context"

	"github.com/viderstv/common/instance"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoInst struct {
	client *mongo.Client
	db     *mongo.Database
}

func (i *MongoInst) Collection(name instance.CollectionName) *mongo.Collection {
	return i.db.Collection(string(name))
}

func (i *MongoInst) Ping(ctx context.Context) error {
	return i.db.Client().Ping(ctx, nil)
}

func (i *MongoInst) RawClient() *mongo.Client {
	return i.client
}

func (i *MongoInst) RawDatabase() *mongo.Database {
	return i.db
}
