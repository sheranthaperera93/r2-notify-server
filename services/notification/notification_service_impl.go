package notificationService

import (
	"errors"
	"r2-notify/data"
	"r2-notify/models"
	notificationRepository "r2-notify/repository/notification"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationServiceImpl struct {
	NotificationRepository notificationRepository.NotificationRepository
	Validate               *validator.Validate
}

// NewNotificationServiceImpl returns a new instance of NotificationService
// with the provided NotificationRepository and validator.Validate instance.
// If the validator instance is nil, an error is returned.
func NewNotificationServiceImpl(notificationRepository notificationRepository.NotificationRepository, validate *validator.Validate) (service NotificationService, err error) {
	if validate == nil {
		return nil, errors.New("validator instance cannot be nil")
	}
	return &NotificationServiceImpl{
		NotificationRepository: notificationRepository,
		Validate:               validate,
	}, err
}

// FindAll returns a list of notifications for the given user ID. If no
// notifications are found for the user, an empty list is returned with a nil
// error. If an error occurs while fetching the notifications, the error is
// returned.
func (t NotificationServiceImpl) FindAll(userId string) (notifications []data.Notification, err error) {
	result, err := t.NotificationRepository.FindAll(userId)
	if err != nil {
		return nil, err
	}

	for _, value := range result {
		notification := data.Notification{
			Id:         value.Id.Hex(),
			AppId:      value.AppId,
			GroupKey:   value.GroupKey,
			Message:    value.Message,
			ReadStatus: value.ReadStatus,
			UserID:     value.UserId,
			Status:     value.Status,
			CreatedAt:  value.CreatedAt,
			UpdatedAt:  value.UpdatedAt,
		}
		notifications = append(notifications, notification)
	}
	if len(notifications) == 0 {
		return []data.Notification{}, nil
	}
	return notifications, nil
}

// FindById retrieves a notification by its ID and user ID from the data store.
// It returns the notification as a data.Notification struct. If the notification
// is not found or an error occurs during the retrieval, it returns an empty
// notification and the corresponding error.
func (t *NotificationServiceImpl) FindById(id primitive.ObjectID, userId string) (notification data.Notification, err error) {
	notificationModel, err := t.NotificationRepository.FindById(id, userId)
	if err != nil {
		return data.Notification{}, err
	}

	notification = data.Notification{
		Id:         notificationModel.Id.Hex(),
		AppId:      notification.AppId,
		GroupKey:   notificationModel.GroupKey,
		Message:    notificationModel.Message,
		ReadStatus: notificationModel.ReadStatus,
		UserID:     notificationModel.UserId,
		Status:     notificationModel.Status,
		CreatedAt:  notificationModel.CreatedAt,
		UpdatedAt:  notificationModel.UpdatedAt,
	}

	return notification, nil
}

// Create creates a notification in the data store. It returns the newly created
// notification's ID and an error if any. If an error occurs during the creation,
// the error is returned.
func (t *NotificationServiceImpl) Create(notification models.Notification) (primitive.ObjectID, error) {
	recordId, err := t.NotificationRepository.Create(notification)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return recordId, nil
}

// MarkAppAsRead marks all notifications of a given application as read for a user
// given by the user ID. If an error occurs during the operation, the error is
// returned.
func (t *NotificationServiceImpl) MarkAppAsRead(userId string, appId string) (err error) {
	t.NotificationRepository.MarkAppAsRead(userId, appId)
	return nil
}

// DeleteAppNotifications deletes all notifications of a given application for a user
// given by the user ID. If an error occurs during the operation, the error is
// returned.
func (t *NotificationServiceImpl) DeleteAppNotifications(userId string, appId string) (err error) {
	t.NotificationRepository.DeleteAppNotifications(userId, appId)
	return nil
}

// MarkGroupAsRead marks all notifications of a given application and group key
// as read for a user given by the user ID. If an error occurs during the
// operation, the error is returned.
func (t *NotificationServiceImpl) MarkGroupAsRead(userId string, appId string, groupKey string) (err error) {
	t.NotificationRepository.MarkGroupAsRead(userId, appId, groupKey)
	return nil
}

// DeleteGroupNotifications deletes all notifications of a given application and group key
// for a user given by the user ID. If an error occurs during the operation, the error is returned.
func (t *NotificationServiceImpl) DeleteGroupNotifications(userId string, appId string, groupKey string) (err error) {
	t.NotificationRepository.DeleteGroupNotifications(userId, appId, groupKey)
	return nil
}

// MarkNotificationAsRead marks a specific notification as read for a user given by the user ID
// and notification ID. If an error occurs during the operation, the error is returned.
func (t *NotificationServiceImpl) MarkNotificationAsRead(userId string, notificationId string) (err error) {
	t.NotificationRepository.MarkNotificationAsRead(userId, notificationId)
	return nil
}

// DeleteNotification deletes a specific notification for a user given by the user ID
// and notification ID. If an error occurs during the operation, the error is returned.
func (t *NotificationServiceImpl) DeleteNotification(userId string, notificationId string) (err error) {
	t.NotificationRepository.DeleteNotification(userId, notificationId)
	return nil
}

// DeleteAllNotifications deletes all notifications for a given user ID.
// If an error occurs during the operation, the error is returned.
func (t *NotificationServiceImpl) DeleteNotifications(userId string) (err error) {
	t.NotificationRepository.DeleteNotifications(userId)
	return nil
}

// MarkAsRead marks all notifications for a given user ID as read. If an error
// occurs during the operation, the error is returned.
func (t *NotificationServiceImpl) MarkAsRead(userId string) (err error) {
	t.NotificationRepository.MarkAsRead(userId)
	return nil
}
