package main

import (
	"context"
	"log"
	"time"

	"github.com/sheranthaperera93/r2-notify-server/internal/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	db := config.MongoConnection()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := migrateUsers(ctx, db); err != nil {
		log.Fatalf("users migration failed: %v", err)
	}
	if err := migrateTokens(ctx, db); err != nil {
		log.Fatalf("tokens migration failed: %v", err)
	}
	if err := migrateNotifications(ctx, db); err != nil {
		log.Fatalf("notifications migration failed: %v", err)
	}
	if err := migrateConfigurations(ctx, db); err != nil {
		log.Fatalf("configurations migration failed: %v", err)
	}

	log.Println("All indexes applied successfully")
}

func migrateUsers(ctx context.Context, db *mongo.Database) error {
	col := db.Collection("users")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("users_email_unique"),
		},
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("users_username_unique"),
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return err
	}
	log.Println("users: indexes applied")
	return nil
}

func migrateTokens(ctx context.Context, db *mongo.Database) error {
	collections := []struct {
		name  string
		field string
		ttl   int32
	}{
		{"refresh_tokens", "expires_at", 0},
		{"verification_tokens", "expires_at", 0},
		{"password_reset_tokens", "expires_at", 0},
	}

	for _, c := range collections {
		col := db.Collection(c.name)

		// Unique index on token value
		_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true).SetName(c.name + "_token_unique"),
		})
		if err != nil {
			return err
		}

		// TTL index so MongoDB auto-expires documents
		_, err = col.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.D{{Key: c.field, Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0).SetName(c.name + "_ttl"),
		})
		if err != nil {
			return err
		}

		log.Printf("%s: indexes applied", c.name)
	}
	return nil
}

func migrateNotifications(ctx context.Context, db *mongo.Database) error {
	col := db.Collection("notifications")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetName("notifications_userId"),
		},
		{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "readStatus", Value: 1}},
			Options: options.Index().SetName("notifications_userId_readStatus"),
		},
		{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "appId", Value: 1}},
			Options: options.Index().SetName("notifications_userId_appId"),
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return err
	}
	log.Println("notifications: indexes applied")
	return nil
}

func migrateConfigurations(ctx context.Context, db *mongo.Database) error {
	col := db.Collection("configurations")

	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "userId", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("configurations_userId_unique"),
	})
	if err != nil {
		return err
	}
	log.Println("configurations: indexes applied")
	return nil
}
