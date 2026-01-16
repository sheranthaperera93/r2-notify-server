package configurationRepository

import (
	"context"
	"errors"
	"r2-notify/logger"
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
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "ConfigurationRepository",
		Operation: "FindByAppAndUser",
		Message:   "Fetching configuration for userId: " + userId,
		UserId:    userId,
	})
	err := t.Db.Collection("configurations").FindOne(
		context.Background(),
		bson.M{"userId": userId},
	).Decode(&configuration)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "ConfigurationRepository",
			Operation: "FindByAppAndUser",
			Message:   "Failed to fetch configuration for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
		return models.Configuration{}, err
	}
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "ConfigurationRepository",
		Operation: "FindByAppAndUser",
		Message:   "Successfully fetched configuration for userId: " + userId,
		UserId:    userId,
	})
	return configuration, nil
}

// Create inserts a new configuration document into the "configurations"
// collection. It returns the inserted document's ObjectID if the operation
// is successful, or an error if the operation fails.
func (t *ConfigurationRepositoryImpl) Create(configuration models.Configuration) (primitive.ObjectID, error) {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "ConfigurationRepository",
		Operation: "Create",
		Message:   "Creating configuration for userId: " + configuration.UserId,
		UserId:    configuration.UserId,
	})
	result, err := t.Db.Collection("configurations").InsertOne(context.Background(), configuration)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "ConfigurationRepository",
			Operation: "Create",
			Message:   "Failed to create configuration for userId: " + configuration.UserId,
			Error:     err,
			UserId:    configuration.UserId,
		})
		return primitive.NilObjectID, err
	}
	id, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		convertErr := errors.New("failed to convert inserted ID to ObjectID")
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "ConfigurationRepository",
			Operation: "Create",
			Message:   "Failed to convert inserted ID for userId: " + configuration.UserId,
			Error:     convertErr,
			UserId:    configuration.UserId,
		})
		return primitive.NilObjectID, convertErr
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Repository",
		Component: "ConfigurationRepository",
		Operation: "Create",
		Message:   "Successfully created configuration for userId: " + configuration.UserId,
		UserId:    configuration.UserId,
	})
	return id, nil
}

// Update updates a configuration document in the "configurations" collection
// with the given models.Configuration document. It returns an error if the
// operation fails, or if no document is found to update.
func (t *ConfigurationRepositoryImpl) Update(configuration models.Configuration) error {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "ConfigurationRepository",
		Operation: "Update",
		Message:   "Updating configuration for userId: " + configuration.UserId,
		UserId:    configuration.UserId,
	})
	filter := bson.M{
		"userId": configuration.UserId,
	}
	update := bson.M{
		"$set": configuration,
	}
	result, err := t.Db.Collection("configurations").UpdateOne(context.Background(), filter, update)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "ConfigurationRepository",
			Operation: "Update",
			Message:   "Failed to update configuration for userId: " + configuration.UserId,
			Error:     err,
			UserId:    configuration.UserId,
		})
		return err
	}
	if result.MatchedCount == 0 {
		notFoundErr := errors.New("no document found to update")
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "ConfigurationRepository",
			Operation: "Update",
			Message:   "No configuration document found to update for userId: " + configuration.UserId,
			Error:     notFoundErr,
			UserId:    configuration.UserId,
		})
		return notFoundErr
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Repository",
		Component: "ConfigurationRepository",
		Operation: "Update",
		Message:   "Successfully updated configuration for userId: " + configuration.UserId,
		UserId:    configuration.UserId,
	})
	return nil
}

// Delete deletes a configuration document from the "configurations" collection
// for the given userId. It returns an error if the operation fails, or if no
// document is found to delete.
func (t *ConfigurationRepositoryImpl) Delete(userId string) error {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "ConfigurationRepository",
		Operation: "Delete",
		Message:   "Deleting configuration for userId: " + userId,
		UserId:    userId,
	})
	filter := bson.M{
		"userId": userId,
	}
	result, err := t.Db.Collection("configurations").DeleteOne(context.Background(), filter)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "ConfigurationRepository",
			Operation: "Delete",
			Message:   "Failed to delete configuration for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
		return err
	}
	if result.DeletedCount == 0 {
		notFoundErr := errors.New("no document found to delete")
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "ConfigurationRepository",
			Operation: "Delete",
			Message:   "No configuration document found to delete for userId: " + userId,
			Error:     notFoundErr,
			UserId:    userId,
		})
		return notFoundErr
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Repository",
		Component: "ConfigurationRepository",
		Operation: "Delete",
		Message:   "Successfully deleted configuration for userId: " + userId,
		UserId:    userId,
	})
	return nil
}
