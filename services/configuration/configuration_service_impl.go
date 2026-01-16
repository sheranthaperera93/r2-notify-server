package configurationService

import (
	"errors"
	"r2-notify/data"
	"r2-notify/logger"
	"r2-notify/models"
	configurationRepository "r2-notify/repository/configuration"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ConfigurationServiceImpl struct {
	ConfigurationRepository configurationRepository.ConfigurationRepository
	Validate                *validator.Validate
}

// NewConfigurationServiceImpl returns a new instance of ConfigurationService, which is used to manage application configurations of users.
// The first parameter is the ConfigurationRepository, which is used to interact with the database to store and retrieve the configurations.
// The second parameter is an instance of validator.Validate, which is used to validate the configuration struct before saving to or retrieving from the database.
// If the second parameter is nil, the function will return an error.
func NewConfigurationServiceImpl(configurationRepository configurationRepository.ConfigurationRepository, validate *validator.Validate) (service ConfigurationService, err error) {
	if validate == nil {
		return nil, errors.New("validator instance cannot be nil")
	}
	return &ConfigurationServiceImpl{
		ConfigurationRepository: configurationRepository,
		Validate:                validate,
	}, err
}

// FindByAppAndUser retrieves the configuration for a specific user based on their user ID.
// It returns a data.Configuration object containing the user's configuration details,
// including the configuration ID, user ID, and notification enablement status.
// If no configuration is found or an error occurs during the retrieval, an error is returned.
func (t ConfigurationServiceImpl) FindByAppAndUser(userId string) (data.Configuration, error) {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Service",
		Component: "ConfigurationService",
		Operation: "FindByAppAndUser",
		Message:   "Fetching configuration for userId: " + userId,
		UserId:    userId,
	})
	result, err := t.ConfigurationRepository.FindByAppAndUser(userId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Service",
			Component: "ConfigurationService",
			Operation: "FindByAppAndUser",
			Message:   "Failed to fetch configuration for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
		return data.Configuration{}, err
	}

	configuration := data.Configuration{
		Id:                 result.Id.Hex(),
		UserID:             result.UserId,
		EnableNotification: result.EnableNotifications,
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Service",
		Component: "ConfigurationService",
		Operation: "FindByAppAndUser",
		Message:   "Successfully fetched configuration for userId: " + userId,
		UserId:    userId,
	})
	return configuration, nil
}

// Create creates a new configuration for the user identified by the configuration's UserId field.
// It returns the ObjectID of the newly created configuration document, or an error if the creation fails.
func (t *ConfigurationServiceImpl) Create(configuration models.Configuration) (primitive.ObjectID, error) {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Service",
		Component: "ConfigurationService",
		Operation: "Create",
		Message:   "Creating configuration for userId: " + configuration.UserId,
		UserId:    configuration.UserId,
	})
	recordId, err := t.ConfigurationRepository.Create(configuration)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Service",
			Component: "ConfigurationService",
			Operation: "Create",
			Message:   "Failed to create configuration for userId: " + configuration.UserId,
			Error:     err,
			UserId:    configuration.UserId,
		})
		return primitive.NilObjectID, err
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Service",
		Component: "ConfigurationService",
		Operation: "Create",
		Message:   "Successfully created configuration for userId: " + configuration.UserId,
		UserId:    configuration.UserId,
	})
	return recordId, nil
}

// Update updates the configuration for a user identified by the configuration's UserId field.
// It returns an error if the update fails.
func (t *ConfigurationServiceImpl) Update(configuration models.Configuration) error {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Service",
		Component: "ConfigurationService",
		Operation: "Update",
		Message:   "Updating configuration for userId: " + configuration.UserId,
		UserId:    configuration.UserId,
	})
	err := t.ConfigurationRepository.Update(configuration)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Service",
			Component: "ConfigurationService",
			Operation: "Update",
			Message:   "Failed to update configuration for userId: " + configuration.UserId,
			Error:     err,
			UserId:    configuration.UserId,
		})
		return err
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Service",
		Component: "ConfigurationService",
		Operation: "Update",
		Message:   "Successfully updated configuration for userId: " + configuration.UserId,
		UserId:    configuration.UserId,
	})
	return nil
}

// Delete deletes the configuration for a user identified by the configuration's UserId field.
// It returns an error if the deletion fails.
func (t *ConfigurationServiceImpl) Delete(userId string) error {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Service",
		Component: "ConfigurationService",
		Operation: "Delete",
		Message:   "Deleting configuration for userId: " + userId,
		UserId:    userId,
	})
	err := t.ConfigurationRepository.Delete(userId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Service",
			Component: "ConfigurationService",
			Operation: "Delete",
			Message:   "Failed to delete configuration for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
		return err
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Service",
		Component: "ConfigurationService",
		Operation: "Delete",
		Message:   "Successfully deleted configuration for userId: " + userId,
		UserId:    userId,
	})
	return nil
}
