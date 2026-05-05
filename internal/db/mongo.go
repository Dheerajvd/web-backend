package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Mongo struct {
	Client *mongo.Client
	DB     *mongo.Database
}

func ConnectMongo(ctx context.Context, uri, database string) (*Mongo, func(context.Context) error, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, fmt.Errorf("mongo connect: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, fmt.Errorf("mongo ping: %w", err)
	}

	m := &Mongo{Client: client, DB: client.Database(database)}
	cleanup := func(ctx context.Context) error {
		return client.Disconnect(ctx)
	}
	return m, cleanup, nil
}
