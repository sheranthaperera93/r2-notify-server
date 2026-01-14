package configurationService

import (
	"errors"
	"r2-notify/data"
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
	result, err := t.ConfigurationRepository.FindByAppAndUser(userId)
	if err != nil {
		return data.Configuration{}, err
	}

	configuration := data.Configuration{
		Id:                 result.Id.Hex(),
		UserID:             result.UserId,
		EnableNotification: result.EnableNotifications,
	}
	return configuration, nil
}

// Create creates a new configuration for the user identified by the configuration's UserId field.
// It returns the ObjectID of the newly created configuration document, or an error if the creation fails.
func (t *ConfigurationServiceImpl) Create(configuration models.Configuration) (primitive.ObjectID, error) {
	recordId, err := t.ConfigurationRepository.Create(configuration)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return recordId, nil
}

// Update updates the configuration for a user identified by the configuration's UserId field.
// It returns an error if the update fails.
func (t *ConfigurationServiceImpl) Update(configuration models.Configuration) error {
	err := t.ConfigurationRepository.Update(configuration)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes the configuration for a user identified by the configuration's UserId field.
// It returns an error if the deletion fails.
func (t *ConfigurationServiceImpl) Delete(userId string) error {
	err := t.ConfigurationRepository.Delete(userId)
	if err != nil {
		return err
	}
	return nil
}
