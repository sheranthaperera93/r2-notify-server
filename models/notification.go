package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Notification struct {
	Id         primitive.ObjectID `bson:"_id,omitempty"`
	AppId      string             `bson:"appId"`
	UserId     string             `bson:"userId"`
	GroupKey   string             `bson:"groupKey"`
	Message    string             `bson:"message"`
	Status     string             `bson:"status"`
	ReadStatus bool               `bson:"readStatus"`
	CreatedAt  time.Time          `bson:"createdAt"`
	UpdatedAt  time.Time          `bson:"updatedAt"`
}
