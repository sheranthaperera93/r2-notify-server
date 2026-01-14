package configurationService

import (
	"r2-notify/data"
	"r2-notify/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ConfigurationService interface {
	FindByAppAndUser(userId string) (configuration data.Configuration, err error)
	Create(configuration models.Configuration) (primitive.ObjectID, error)
	Update(configuration models.Configuration) error
	Delete(userId string) error
}
