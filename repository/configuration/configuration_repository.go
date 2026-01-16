package configurationRepository

import (
	"r2-notify-server/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ConfigurationRepository interface {
	FindByAppAndUser(userId string) (configurations models.Configuration, err error)
	Create(configuration models.Configuration) (primitive.ObjectID, error)
	Update(configuration models.Configuration) error
	Delete(userId string) error
}
