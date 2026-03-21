package configurationService

import (
	"github.com/sheranthaperera93/r2-notify-server/internal/models"

	"github.com/sheranthaperera93/r2-notify-server/internal/data"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ConfigurationService interface {
	FindByAppAndUser(userId string) (configuration data.Configuration, err error)
	Create(configuration models.Configuration) (primitive.ObjectID, error)
	Update(configuration models.Configuration) error
	Delete(userId string) error
}
