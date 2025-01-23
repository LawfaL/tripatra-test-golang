package config

import (
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func ConnectDB(conf *Config) *mongo.Client {
	mongoconn := options.Client().ApplyURI(conf.DBUri)
	Mongoclient, Err := mongo.Connect(ctx, mongoconn)

	if Err != nil {
		panic(Err)
	}

	if err := Mongoclient.Ping(ctx, readpref.Primary()); err != nil {
		panic(err)
	}

	fmt.Println("MongoDB successfully connected...")
	return Mongoclient
}
