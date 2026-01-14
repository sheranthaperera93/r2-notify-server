package notificationService

import (
	"address-book-notification-service/data"
	"address-book-notification-service/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationService interface {
	FindAll(userId string) (notifications []data.Notification, err error)
	FindById(id primitive.ObjectID, userId string) (notification data.Notification, err error)
	Create(notification models.Notification) (primitive.ObjectID, error)
	MarkAsRead(userId string) error
	MarkAppAsRead(userId string, appId string) error
	MarkGroupAsRead(userId string, appId string, groupKey string) error
	MarkNotificationAsRead(userId string, notificationId string) error
	DeleteNotifications(userId string) error
	DeleteAppNotifications(userId string, appId string) error
	DeleteGroupNotifications(userId string, appId string, groupKey string) error
	DeleteNotification(userId string, notificationId string) error
}
