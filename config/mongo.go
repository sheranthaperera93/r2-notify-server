package config

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func MongoConnection() *mongo.Database {
	host := LoadConfig().MongoHost
	port := LoadConfig().MongoPort
	dbName := LoadConfig().MongoDBName
	username := LoadConfig().MongoUserName
	password := LoadConfig().MongoPassword
	mongoRetryWrites := LoadConfig().mongoRetryWrites
	mongoSsl := LoadConfig().mongoSsl

	fmt.Println("Mongo Configurations", host, port, dbName, username, password, mongoRetryWrites, mongoSsl)
	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/?ssl=%s&retrywrites=%s",
		username,
		password,
		host,
		port,
		mongoSsl,
		mongoRetryWrites,
	)

	fmt.Println("Mongo Connection URI:", uri)

	clientOptions := options.Client().ApplyURI(uri).SetDirect(true)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("MongoDB connection error: %v", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB ping error: %v", err)
	}

	log.Default().Printf("Connected to MongoDB at %s:%d, using database: %s", host, port, dbName)
	return client.Database(dbName)
}
