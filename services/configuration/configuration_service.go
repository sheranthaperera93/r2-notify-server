package configurationService

import (
	"address-book-notification-service/data"
	"address-book-notification-service/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ConfigurationService interface {
	FindByAppAndUser(userId string) (configuration data.Configuration, err error)
	Create(configuration models.Configuration) (primitive.ObjectID, error)
	Update(configuration models.Configuration) error
	Delete(userId string) error
}
