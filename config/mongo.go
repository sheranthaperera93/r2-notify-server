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
	mongoRetryWrites := LoadConfig().MongoRetryWrites
	mongoSsl := LoadConfig().MongoSsl
	mongoSchema := LoadConfig().MongoSchema

	log.Printf("Mongo Configurations: host=%s, port=%d, dbName=%s, username=%s, password=***, mongoRetryWrites=%t, mongoSsl=%t", host, port, dbName, username, mongoRetryWrites, mongoSsl)
	uri := ""
	isDirect := false
	if mongoSchema == "mongodb+srv" {
		isDirect = false
		uri = fmt.Sprintf(
			"%s://%s:%s@%s/?ssl=%t&retrywrites=%t",
			mongoSchema,
			username,
			password,
			host,
			mongoSsl,
			mongoRetryWrites,
		)
	} else {
		isDirect = true
		uri = fmt.Sprintf(
			"%s://%s:%s@%s:%d/?ssl=%t&retrywrites=%t",
			mongoSchema,
			username,
			password,
			host,
			port,
			mongoSsl,
			mongoRetryWrites,
		)
	}

	clientOptions := options.Client().ApplyURI(uri).SetDirect(isDirect)
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

	log.Printf("Connected to MongoDB at %s:%d, using database: %s", host, port, dbName)
	return client.Database(dbName)
}
