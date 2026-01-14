package notificationRepository

import (
	"address-book-notification-service/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationRepository interface {
	FindAll(userId string) ([]models.Notification, error)
	FindById(id primitive.ObjectID, userId string) (models.Notification, error)
	Create(notification models.Notification) (primitive.ObjectID, error)
	MarkAsRead(clientId string) error
	MarkAppAsRead(clientId string, appId string) error
	MarkGroupAsRead(clientId string, appId string, groupKey string) error
	MarkNotificationAsRead(clientId string, notificationId string) error
	DeleteNotifications(clientId string) error
	DeleteAppNotifications(clientId string, appId string) error
	DeleteGroupNotifications(clientId string, appId string, groupKey string) error
	DeleteNotification(clientId string, notificationId string) error
}
