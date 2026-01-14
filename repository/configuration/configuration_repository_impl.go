package configurationRepository

import (
	"context"
	"errors"
	"r2-notify/models"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ConfigurationRepositoryImpl struct {
	Db *mongo.Database
}

// NewConfigurationRepositoryImpl creates a new instance of ConfigurationRepositoryImpl
// with the given mongo Db instance.
func NewConfigurationRepositoryImpl(Db *mongo.Database) ConfigurationRepository {
	return &ConfigurationRepositoryImpl{Db: Db}
}

// FindByAppAndUser retrieves a configuration document from the "configurations" collection
// for the given userId. It returns the configuration if found, or an error if the operation
// fails or no configuration is found for the specified userId.

func (t ConfigurationRepositoryImpl) FindByAppAndUser(userId string) (models.Configuration, error) {
	var configuration models.Configuration
	err := t.Db.Collection("configurations").FindOne(
		context.Background(),
		bson.M{"userId": userId},
	).Decode(&configuration)
	if err != nil {
		return models.Configuration{}, err
	}
	return configuration, nil
}

// Create inserts a new configuration document into the "configurations"
// collection. It returns the inserted document's ObjectID if the operation
// is successful, or an error if the operation fails.
func (t *ConfigurationRepositoryImpl) Create(configuration models.Configuration) (primitive.ObjectID, error) {
	result, err := t.Db.Collection("configurations").InsertOne(context.Background(), configuration)
	if err != nil {
		return primitive.NilObjectID, err
	}
	id, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("failed to convert inserted ID to ObjectID")
	}
	return id, nil
}

// Update updates a configuration document in the "configurations" collection
// with the given models.Configuration document. It returns an error if the
// operation fails, or if no document is found to update.
func (t *ConfigurationRepositoryImpl) Update(configuration models.Configuration) error {
	filter := bson.M{
		"userId": configuration.UserId,
	}
	update := bson.M{
		"$set": configuration,
	}
	result, err := t.Db.Collection("configurations").UpdateOne(context.Background(), filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("no document found to update")
	}
	return nil
}

// Delete deletes a configuration document from the "configurations" collection
// for the given userId. It returns an error if the operation fails, or if no
// document is found to delete.
func (t *ConfigurationRepositoryImpl) Delete(userId string) error {
	filter := bson.M{
		"userId": userId,
	}
	result, err := t.Db.Collection("configurations").DeleteOne(context.Background(), filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("no document found to delete")
	}
	return nil
}
