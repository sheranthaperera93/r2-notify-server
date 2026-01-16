package notificationService

import (
	"errors"
	"r2-notify-server/data"
	"r2-notify-server/logger"
	"r2-notify-server/models"
	notificationRepository "r2-notify-server/repository/notification"

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
	logger.Log.Debug(logger.LogPayload{
		Component: "Notification Service",
		Operation: "FindAll",
		Message:   "Fetching all notifications for userId: " + userId,
		UserId:    userId,
	})
	result, err := t.NotificationRepository.FindAll(userId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Notification Service",
			Operation: "FindAll",
			Message:   "Failed to fetch notifications for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
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
		logger.Log.Debug(logger.LogPayload{
			Component: "Notification Service",
			Operation: "FindAll",
			Message:   "No notifications found for userId: " + userId,
			UserId:    userId,
		})
		return []data.Notification{}, nil
	}
	logger.Log.Info(logger.LogPayload{
		Component: "Notification Service",
		Operation: "FindAll",
		Message:   "Successfully fetched notifications for userId: " + userId,
		UserId:    userId,
	})
	return notifications, nil
}

// FindById retrieves a notification by its ID and user ID from the data store.
// It returns the notification as a data.Notification struct. If the notification
// is not found or an error occurs during the retrieval, it returns an empty
// notification and the corresponding error.
func (t *NotificationServiceImpl) FindById(id primitive.ObjectID, userId string) (notification data.Notification, err error) {
	logger.Log.Debug(logger.LogPayload{
		Component: "Notification Service",
		Operation: "FindById",
		Message:   "Fetching notification by ID for userId: " + userId,
		UserId:    userId,
	})
	notificationModel, err := t.NotificationRepository.FindById(id, userId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Notification Service",
			Operation: "FindById",
			Message:   "Failed to fetch notification for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
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
	logger.Log.Info(logger.LogPayload{
		Component: "Notification Service",
		Operation: "FindById",
		Message:   "Successfully fetched notification for userId: " + userId,
		UserId:    userId,
	})
	return notification, nil
}

// Create creates a notification in the data store. It returns the newly created
// notification's ID and an error if any. If an error occurs during the creation,
// the error is returned.
func (t *NotificationServiceImpl) Create(notification models.Notification) (primitive.ObjectID, error) {
	logger.Log.Debug(logger.LogPayload{
		Component: "Notification Service",
		Operation: "Create",
		Message:   "Creating notification for userId: " + notification.UserId,
		UserId:    notification.UserId,
	})
	recordId, err := t.NotificationRepository.Create(notification)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Notification Service",
			Operation: "Create",
			Message:   "Failed to create notification for userId: " + notification.UserId,
			Error:     err,
			UserId:    notification.UserId,
		})
		return primitive.NilObjectID, err
	}
	logger.Log.Info(logger.LogPayload{
		Component: "Notification Service",
		Operation: "Create",
		Message:   "Successfully created notification for userId: " + notification.UserId,
		UserId:    notification.UserId,
	})
	return recordId, nil
}

// MarkAppAsRead marks all notifications of a given application as read for a user
// given by the user ID. If an error occurs during the operation, the error is
// returned.
func (t *NotificationServiceImpl) MarkAppAsRead(userId string, appId string) (err error) {
	logger.Log.Debug(logger.LogPayload{
		Component: "Notification Service",
		Operation: "MarkAppAsRead",
		Message:   "Marking app notifications as read for userId: " + userId + ", appId: " + appId,
		UserId:    userId,
		AppId:     appId,
	})
	err = t.NotificationRepository.MarkAppAsRead(userId, appId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Notification Service",
			Operation: "MarkAppAsRead",
			Message:   "Failed to mark app notifications as read for userId: " + userId + ", appId: " + appId,
			Error:     err,
			UserId:    userId,
			AppId:     appId,
		})
	}
	return err
}

// DeleteAppNotifications deletes all notifications of a given application for a user
// given by the user ID. If an error occurs during the operation, the error is
// returned.
func (t *NotificationServiceImpl) DeleteAppNotifications(userId string, appId string) (err error) {
	logger.Log.Debug(logger.LogPayload{
		Component: "Notification Service",
		Operation: "DeleteAppNotifications",
		Message:   "Deleting app notifications for userId: " + userId + ", appId: " + appId,
		UserId:    userId,
		AppId:     appId,
	})
	err = t.NotificationRepository.DeleteAppNotifications(userId, appId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Notification Service",
			Operation: "DeleteAppNotifications",
			Message:   "Failed to delete app notifications for userId: " + userId + ", appId: " + appId,
			Error:     err,
			UserId:    userId,
			AppId:     appId,
		})
	}
	return err
}

// MarkGroupAsRead marks all notifications of a given application and group key
// as read for a user given by the user ID. If an error occurs during the
// operation, the error is returned.
func (t *NotificationServiceImpl) MarkGroupAsRead(userId string, appId string, groupKey string) (err error) {
	logger.Log.Debug(logger.LogPayload{
		Component: "Notification Service",
		Operation: "MarkGroupAsRead",
		Message:   "Marking group notifications as read for userId: " + userId + ", appId: " + appId + ", groupKey: " + groupKey,
		UserId:    userId,
		AppId:     appId,
	})
	err = t.NotificationRepository.MarkGroupAsRead(userId, appId, groupKey)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Notification Service",
			Operation: "MarkGroupAsRead",
			Message:   "Failed to mark group notifications as read for userId: " + userId + ", appId: " + appId + ", groupKey: " + groupKey,
			Error:     err,
			UserId:    userId,
			AppId:     appId,
		})
	}
	return err
}

// DeleteGroupNotifications deletes all notifications of a given application and group key
// for a user given by the user ID. If an error occurs during the operation, the error is returned.
func (t *NotificationServiceImpl) DeleteGroupNotifications(userId string, appId string, groupKey string) (err error) {
	logger.Log.Debug(logger.LogPayload{
		Component: "Notification Service",
		Operation: "DeleteGroupNotifications",
		Message:   "Deleting group notifications for userId: " + userId + ", appId: " + appId + ", groupKey: " + groupKey,
		UserId:    userId,
		AppId:     appId,
	})
	err = t.NotificationRepository.DeleteGroupNotifications(userId, appId, groupKey)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Notification Service",
			Operation: "DeleteGroupNotifications",
			Message:   "Failed to delete group notifications for userId: " + userId + ", appId: " + appId + ", groupKey: " + groupKey,
			Error:     err,
			UserId:    userId,
			AppId:     appId,
		})
	}
	return err
}

// MarkNotificationAsRead marks a specific notification as read for a user given by the user ID
// and notification ID. If an error occurs during the operation, the error is returned.
func (t *NotificationServiceImpl) MarkNotificationAsRead(userId string, notificationId string) (err error) {
	logger.Log.Debug(logger.LogPayload{
		Component: "Notification Service",
		Operation: "MarkNotificationAsRead",
		Message:   "Marking notification as read for userId: " + userId,
		UserId:    userId,
	})
	err = t.NotificationRepository.MarkNotificationAsRead(userId, notificationId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Notification Service",
			Operation: "MarkNotificationAsRead",
			Message:   "Failed to mark notification as read for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
	}
	return err
}

// DeleteNotification deletes a specific notification for a user given by the user ID
// and notification ID. If an error occurs during the operation, the error is returned.
func (t *NotificationServiceImpl) DeleteNotification(userId string, notificationId string) (err error) {
	logger.Log.Debug(logger.LogPayload{
		Component: "Notification Service",
		Operation: "DeleteNotification",
		Message:   "Deleting notification for userId: " + userId,
		UserId:    userId,
	})
	err = t.NotificationRepository.DeleteNotification(userId, notificationId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Notification Service",
			Operation: "DeleteNotification",
			Message:   "Failed to delete notification for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
	}
	return err
}

// DeleteAllNotifications deletes all notifications for a given user ID.
// If an error occurs during the operation, the error is returned.
func (t *NotificationServiceImpl) DeleteNotifications(userId string) (err error) {
	logger.Log.Debug(logger.LogPayload{
		Component: "Notification Service",
		Operation: "DeleteNotifications",
		Message:   "Deleting all notifications for userId: " + userId,
		UserId:    userId,
	})
	err = t.NotificationRepository.DeleteNotifications(userId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Notification Service",
			Operation: "DeleteNotifications",
			Message:   "Failed to delete notifications for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
	}
	return err
}

// MarkAsRead marks all notifications for a given user ID as read. If an error
// occurs during the operation, the error is returned.
func (t *NotificationServiceImpl) MarkAsRead(userId string) (err error) {
	logger.Log.Debug(logger.LogPayload{
		Component: "Notification Service",
		Operation: "MarkAsRead",
		Message:   "Marking all notifications as read for userId: " + userId,
		UserId:    userId,
	})
	err = t.NotificationRepository.MarkAsRead(userId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Notification Service",
			Operation: "MarkAsRead",
			Message:   "Failed to mark all notifications as read for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
	}
	return err
}
