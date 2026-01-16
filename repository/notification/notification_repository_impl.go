package notificationRepository

import (
	"context"
	"errors"
	"fmt"
	"r2-notify/logger"
	"r2-notify/models"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type NotificationRepositoryImpl struct {
	Db *mongo.Database
}

// NewNotificationRepositoryImpl returns a new instance of NotificationRepositoryImpl.
// It takes a pointer to a mongo.Database as an argument, which is used to interact with the database.
// The returned NotificationRepositoryImpl is safe to use concurrently.
func NewNotificationRepositoryImpl(Db *mongo.Database) NotificationRepository {
	return &NotificationRepositoryImpl{Db: Db}
}

// FindAll finds all unread notifications for a given user.
// The notifications are retrieved from the database, and the function returns a slice of Notification
// objects. If an error occurs during the retrieval process, the function returns an error.
func (t NotificationRepositoryImpl) FindAll(userId string) (notifications []models.Notification, err error) {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "FindAll",
		Message:   "Fetching all unread notifications for userId: " + userId,
		UserId:    userId,
	})
	cursor, err := t.Db.Collection("notifications").Find(context.Background(), bson.M{"userId": userId, "readStatus": false})
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "FindAll",
			Message:   "Failed to fetch notifications for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
		return nil, err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var notification models.Notification
		if err := cursor.Decode(&notification); err != nil {
			logger.Log.Error(logger.LogPayload{
				Service:   "Repository",
				Component: "NotificationRepository",
				Operation: "FindAll",
				Message:   "Failed to decode notification for userId: " + userId,
				Error:     err,
				UserId:    userId,
			})
			return nil, err
		}
		notifications = append(notifications, notification)
	}

	if err := cursor.Err(); err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "FindAll",
			Message:   "Cursor error while fetching notifications for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
		return nil, err
	}
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "FindAll",
		Message:   "Successfully fetched notifications for userId: " + userId,
		UserId:    userId,
	})
	return notifications, nil
}

// FindById retrieves a notification document from the database using the specified notificationId and userId.
// It returns the notification if found, or an error if the notification is not found or if there is an issue with the database query.
func (t NotificationRepositoryImpl) FindById(notificationId primitive.ObjectID, userId string) (notification models.Notification, err error) {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "FindById",
		Message:   "Fetching notification by ID for userId: " + userId,
		UserId:    userId,
	})
	result := t.Db.Collection("notifications").FindOne(context.Background(), bson.M{"_id": notificationId, "userId": userId})
	if err := result.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			notFoundErr := errors.New("notification not found")
			logger.Log.Error(logger.LogPayload{
				Service:   "Repository",
				Component: "NotificationRepository",
				Operation: "FindById",
				Message:   "Notification not found for userId: " + userId,
				Error:     notFoundErr,
				UserId:    userId,
			})
			return models.Notification{}, notFoundErr
		}
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "FindById",
			Message:   "Error fetching notification for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
		return models.Notification{}, err
	}
	if err := result.Decode(&notification); err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "FindById",
			Message:   "Failed to decode notification for userId: " + userId,
			Error:     err,
			UserId:    userId,
		})
		return models.Notification{}, err
	}
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "FindById",
		Message:   "Successfully fetched notification for userId: " + userId,
		UserId:    userId,
	})
	return notification, nil
}

// Create creates a new notification document in the database and returns the ID of the newly created document, or an error if the creation fails.
func (t *NotificationRepositoryImpl) Create(notification models.Notification) (primitive.ObjectID, error) {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "Create",
		Message:   "Creating notification for userId: " + notification.UserId,
		UserId:    notification.UserId,
	})
	result, err := t.Db.Collection("notifications").InsertOne(context.Background(), notification)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "Create",
			Message:   "Failed to create notification for userId: " + notification.UserId,
			Error:     err,
			UserId:    notification.UserId,
		})
		return primitive.NilObjectID, err
	}
	id, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		convertErr := errors.New("failed to convert inserted ID to ObjectID")
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "Create",
			Message:   "Failed to convert inserted ID for userId: " + notification.UserId,
			Error:     convertErr,
			UserId:    notification.UserId,
		})
		return primitive.NilObjectID, convertErr
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "Create",
		Message:   "Successfully created notification for userId: " + notification.UserId,
		UserId:    notification.UserId,
	})
	return id, nil
}

// MarkAsRead marks all unread notifications for a given user as read.
// It trims and removes any double quotes from the clientId,
// and then updates all relevant notifications in the database with the current time and sets the readStatus to true.
// It returns an error if there is an issue with the database query.
func (t *NotificationRepositoryImpl) MarkAsRead(clientId string) error {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "MarkAsRead",
		Message:   "Marking all notifications as read for userId: " + clientId,
		UserId:    clientId,
	})
	updatedResults, err := t.Db.Collection("notifications").UpdateMany(context.Background(), bson.M{"userId": clientId}, bson.M{"$set": bson.M{"readStatus": true, "updatedAt": primitive.NewDateTimeFromTime(time.Now())}})
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "MarkAsRead",
			Message:   "Failed to mark notifications as read for userId: " + clientId,
			Error:     err,
			UserId:    clientId,
		})
		return err
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "MarkAsRead",
		Message:   "Marked notifications as read for userId: " + clientId + " | Matched: " + fmt.Sprintf("%d", updatedResults.MatchedCount) + " Modified: " + fmt.Sprintf("%d", updatedResults.ModifiedCount),
		UserId:    clientId,
	})
	return nil
}

// MarkAppAsRead marks all unread notifications for a given user and appId as read.
func (t *NotificationRepositoryImpl) MarkAppAsRead(clientId string, appId string) error {
	appId = strings.TrimSpace(appId)
	appId = strings.Trim(appId, `"'`)
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "MarkAppAsRead",
		Message:   "Marking app notifications as read for userId: " + clientId + ", appId: " + appId,
		UserId:    clientId,
		AppId:     appId,
	})
	updatedResults, err := t.Db.Collection("notifications").UpdateMany(context.Background(), bson.M{"userId": clientId, "appId": appId}, bson.M{"$set": bson.M{"readStatus": true, "updatedAt": primitive.NewDateTimeFromTime(time.Now())}})
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "MarkAppAsRead",
			Message:   "Failed to mark app notifications as read for userId: " + clientId + ", appId: " + appId,
			Error:     err,
			UserId:    clientId,
			AppId:     appId,
		})
		return err
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "MarkAppAsRead",
		Message:   "Marked app notifications as read for userId: " + clientId + ", appId: " + appId + " | Matched: " + fmt.Sprintf("%d", updatedResults.MatchedCount) + " Modified: " + fmt.Sprintf("%d", updatedResults.ModifiedCount),
		UserId:    clientId,
		AppId:     appId,
	})
	return nil
}

// MarkGroupAsRead marks all unread notifications for a given user, appId and groupKey as read.
// It trims the appId and groupKey of any whitespace and removes any double quotes from the strings.
// It then updates the relevant notifications in the database with the current time and sets the readStatus to true.
func (t *NotificationRepositoryImpl) MarkGroupAsRead(clientId string, appId string, groupKey string) error {
	appId = strings.TrimSpace(appId)
	groupKey = strings.TrimSpace(groupKey)
	appId = strings.Trim(appId, `"'`)
	groupKey = strings.Trim(groupKey, `"'`)
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "MarkGroupAsRead",
		Message:   "Marking group notifications as read for userId: " + clientId + ", appId: " + appId + ", groupKey: " + groupKey,
		UserId:    clientId,
		AppId:     appId,
	})
	updatedResults, err := t.Db.Collection("notifications").UpdateMany(context.Background(), bson.M{"userId": clientId, "appId": appId, "groupKey": groupKey}, bson.M{"$set": bson.M{"readStatus": true, "updatedAt": primitive.NewDateTimeFromTime(time.Now())}})
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "MarkGroupAsRead",
			Message:   "Failed to mark group notifications as read for userId: " + clientId + ", appId: " + appId + ", groupKey: " + groupKey,
			Error:     err,
			UserId:    clientId,
			AppId:     appId,
		})
		return err
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "MarkGroupAsRead",
		Message:   "Marked group notifications as read for userId: " + clientId + ", appId: " + appId + ", groupKey: " + groupKey + " | Matched: " + fmt.Sprintf("%d", updatedResults.MatchedCount) + " Modified: " + fmt.Sprintf("%d", updatedResults.ModifiedCount),
		UserId:    clientId,
		AppId:     appId,
	})
	return nil
}

// MarkNotificationAsRead marks a notification as read for a given user.
// It takes a clientId and a notificationId as arguments, trims and removes any double quotes from the strings,
// converts the notificationId to an ObjectID, and then updates the relevant notification in the database with the current time and sets the readStatus to true.
// It returns an error if the notification is not found or if there is an issue with the database query.
func (t *NotificationRepositoryImpl) MarkNotificationAsRead(clientId string, notificationId string) error {
	notificationId = strings.TrimSpace(notificationId)
	notificationId = strings.Trim(notificationId, `"'`)
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "MarkNotificationAsRead",
		Message:   "Marking notification as read for userId: " + clientId,
		UserId:    clientId,
	})
	objID, err := primitive.ObjectIDFromHex(notificationId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "MarkNotificationAsRead",
			Message:   "Failed to convert notification ID for userId: " + clientId,
			Error:     err,
			UserId:    clientId,
		})
		return err
	}
	updatedResults, err := t.Db.Collection("notifications").UpdateByID(context.Background(), objID, bson.M{"$set": bson.M{"readStatus": true, "updatedAt": primitive.NewDateTimeFromTime(time.Now())}})
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "MarkNotificationAsRead",
			Message:   "Failed to mark notification as read for userId: " + clientId,
			Error:     err,
			UserId:    clientId,
		})
		return err
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "MarkNotificationAsRead",
		Message:   "Marked notification as read for userId: " + clientId + " | Matched: " + fmt.Sprintf("%d", updatedResults.MatchedCount) + " Modified: " + fmt.Sprintf("%d", updatedResults.ModifiedCount),
		UserId:    clientId,
	})
	return nil
}

// DeleteAllNotifications deletes all notifications for a given user.
// It trims and removes any double quotes from the clientId,
// and then deletes all relevant notifications in the database.
// It returns an error if there is an issue with the database query.
func (t *NotificationRepositoryImpl) DeleteNotifications(clientId string) error {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "DeleteNotifications",
		Message:   "Deleting all notifications for userId: " + clientId,
		UserId:    clientId,
	})
	deleteResult, err := t.Db.Collection("notifications").DeleteMany(context.Background(), bson.M{"userId": clientId})
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "DeleteNotifications",
			Message:   "Failed to delete notifications for userId: " + clientId,
			Error:     err,
			UserId:    clientId,
		})
		return err
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "DeleteNotifications",
		Message:   "Deleted notifications for userId: " + clientId + " | Deleted: " + fmt.Sprintf("%d", deleteResult.DeletedCount),
		UserId:    clientId,
	})
	return nil
}

// DeleteAppNotifications deletes all notifications for a given user and appId.
func (t *NotificationRepositoryImpl) DeleteAppNotifications(clientId string, appId string) error {
	appId = strings.TrimSpace(appId)
	appId = strings.Trim(appId, `"'`)
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "DeleteAppNotifications",
		Message:   "Deleting app notifications for userId: " + clientId + ", appId: " + appId,
		UserId:    clientId,
		AppId:     appId,
	})
	deleteResult, err := t.Db.Collection("notifications").DeleteMany(context.Background(), bson.M{"userId": clientId, "appId": appId})
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "DeleteAppNotifications",
			Message:   "Failed to delete app notifications for userId: " + clientId + ", appId: " + appId,
			Error:     err,
			UserId:    clientId,
			AppId:     appId,
		})
		return err
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "DeleteAppNotifications",
		Message:   "Deleted app notifications for userId: " + clientId + ", appId: " + appId + " | Deleted: " + fmt.Sprintf("%d", deleteResult.DeletedCount),
		UserId:    clientId,
		AppId:     appId,
	})
	return nil
}

// DeleteGroupNotifications deletes all notifications for a given user, appId and groupKey.
// It trims the appId and groupKey of any whitespace and removes any double quotes from the strings.
// It then deletes the relevant notifications in the database.
func (t *NotificationRepositoryImpl) DeleteGroupNotifications(clientId string, appId string, groupKey string) error {
	appId = strings.TrimSpace(appId)
	groupKey = strings.TrimSpace(groupKey)
	appId = strings.Trim(appId, `"'`)
	groupKey = strings.Trim(groupKey, `"'`)
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "DeleteGroupNotifications",
		Message:   "Deleting group notifications for userId: " + clientId + ", appId: " + appId + ", groupKey: " + groupKey,
		UserId:    clientId,
		AppId:     appId,
	})
	deleteResult, err := t.Db.Collection("notifications").DeleteMany(context.Background(), bson.M{"userId": clientId, "appId": appId, "groupKey": groupKey})
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "DeleteGroupNotifications",
			Message:   "Failed to delete group notifications for userId: " + clientId + ", appId: " + appId + ", groupKey: " + groupKey,
			Error:     err,
			UserId:    clientId,
			AppId:     appId,
		})
		return err
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "DeleteGroupNotifications",
		Message:   "Deleted group notifications for userId: " + clientId + ", appId: " + appId + ", groupKey: " + groupKey + " | Deleted: " + fmt.Sprintf("%d", deleteResult.DeletedCount),
		UserId:    clientId,
		AppId:     appId,
	})
	return nil
}

// DeleteNotification deletes a notification for a given user.
// It takes a clientId and a notificationId as arguments, trims and removes any double quotes from the strings,
// converts the notificationId to an ObjectID, and then deletes the relevant notification in the database.
// It returns an error if the notification is not found or if there is an issue with the database query.
func (t *NotificationRepositoryImpl) DeleteNotification(clientId string, notificationId string) error {
	notificationId = strings.TrimSpace(notificationId)
	notificationId = strings.Trim(notificationId, `"'`)
	logger.Log.Debug(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "DeleteNotification",
		Message:   "Deleting notification for userId: " + clientId,
		UserId:    clientId,
	})
	objID, err := primitive.ObjectIDFromHex(notificationId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "DeleteNotification",
			Message:   "Failed to convert notification ID for userId: " + clientId,
			Error:     err,
			UserId:    clientId,
		})
		return err
	}
	deleteResult, err := t.Db.Collection("notifications").DeleteOne(context.Background(), bson.M{"userId": clientId, "_id": objID})
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Repository",
			Component: "NotificationRepository",
			Operation: "DeleteNotification",
			Message:   "Failed to delete notification for userId: " + clientId,
			Error:     err,
			UserId:    clientId,
		})
		return err
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Repository",
		Component: "NotificationRepository",
		Operation: "DeleteNotification",
		Message:   "Deleted notification for userId: " + clientId + " | Deleted: " + fmt.Sprintf("%d", deleteResult.DeletedCount),
		UserId:    clientId,
	})
	return nil
}
