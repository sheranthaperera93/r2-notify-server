package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Configuration struct {
	Id                  primitive.ObjectID `bson:"_id,omitempty"`
	UserId              string             `bson:"userId"`
	EnableNotifications bool               `bson:"enableNotifications"`
}
