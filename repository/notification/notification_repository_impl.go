package notificationRepository

import (
	"address-book-notification-service/models"
	"context"
	"errors"
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
	cursor, err := t.Db.Collection("notifications").Find(context.Background(), bson.M{"userId": userId, "readStatus": false})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var notification models.Notification
		if err := cursor.Decode(&notification); err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return notifications, nil
}

// FindById retrieves a notification document from the database using the specified notificationId and userId.
// It returns the notification if found, or an error if the notification is not found or if there is an issue with the database query.
func (t NotificationRepositoryImpl) FindById(notificationId primitive.ObjectID, userId string) (notification models.Notification, err error) {

	result := t.Db.Collection("notifications").FindOne(context.Background(), bson.M{"_id": notificationId, "userId": userId})
	if err := result.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return models.Notification{}, errors.New("notification not found")
		}
		return models.Notification{}, err
	}
	if err := result.Decode(&notification); err != nil {
		return models.Notification{}, err
	}
	return notification, nil
}

// Create creates a new notification document in the database and returns the ID of the newly created document, or an error if the creation fails.
func (t *NotificationRepositoryImpl) Create(notification models.Notification) (primitive.ObjectID, error) {
	result, err := t.Db.Collection("notifications").InsertOne(context.Background(), notification)
	if err != nil {
		return primitive.NilObjectID, err
	}
	id, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("failed to convert inserted ID to ObjectID")
	}
	return id, nil
}

// MarkAsRead marks all unread notifications for a given user as read.
// It trims and removes any double quotes from the clientId,
// and then updates all relevant notifications in the database with the current time and sets the readStatus to true.
// It returns an error if there is an issue with the database query.
func (t *NotificationRepositoryImpl) MarkAsRead(clientId string) error {
	updatedResults, err := t.Db.Collection("notifications").UpdateMany(context.Background(), bson.M{"userId": clientId}, bson.M{"$set": bson.M{"readStatus": true, "updatedAt": primitive.NewDateTimeFromTime(time.Now())}})
	if err != nil {
		println("Error", err.Error())
		return err
	}
	println("Matched: ", updatedResults.MatchedCount, " | Modified: ", updatedResults.ModifiedCount)
	return nil
}

// MarkAppAsRead marks all unread notifications for a given user and appId as read.
func (t *NotificationRepositoryImpl) MarkAppAsRead(clientId string, appId string) error {
	appId = strings.TrimSpace(appId)
	appId = strings.Trim(appId, `"'`)
	updatedResults, err := t.Db.Collection("notifications").UpdateMany(context.Background(), bson.M{"userId": clientId, "appId": appId}, bson.M{"$set": bson.M{"readStatus": true, "updatedAt": primitive.NewDateTimeFromTime(time.Now())}})
	if err != nil {
		println("Error", err.Error())
		return err
	}
	println("Matched: ", updatedResults.MatchedCount, " | Modified: ", updatedResults.ModifiedCount)
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
	updatedResults, err := t.Db.Collection("notifications").UpdateMany(context.Background(), bson.M{"userId": clientId, "appId": appId, "groupKey": groupKey}, bson.M{"$set": bson.M{"readStatus": true, "updatedAt": primitive.NewDateTimeFromTime(time.Now())}})
	if err != nil {
		println("Error", err.Error())
		return err
	}
	println("Matched: ", updatedResults.MatchedCount, " | Modified: ", updatedResults.ModifiedCount)
	return nil
}

// MarkNotificationAsRead marks a notification as read for a given user.
// It takes a clientId and a notificationId as arguments, trims and removes any double quotes from the strings,
// converts the notificationId to an ObjectID, and then updates the relevant notification in the database with the current time and sets the readStatus to true.
// It returns an error if the notification is not found or if there is an issue with the database query.
func (t *NotificationRepositoryImpl) MarkNotificationAsRead(clientId string, notificationId string) error {
	notificationId = strings.TrimSpace(notificationId)
	notificationId = strings.Trim(notificationId, `"'`)
	objID, err := primitive.ObjectIDFromHex(notificationId)
	if err != nil {
		return err
	}
	updatedResults, err := t.Db.Collection("notifications").UpdateByID(context.Background(), objID, bson.M{"$set": bson.M{"readStatus": true, "updatedAt": primitive.NewDateTimeFromTime(time.Now())}})
	if err != nil {
		println("Error", err.Error())
		return err
	}
	println("Matched: ", updatedResults.MatchedCount, " | Modified: ", updatedResults.ModifiedCount)
	return nil
}

// DeleteAllNotifications deletes all notifications for a given user.
// It trims and removes any double quotes from the clientId,
// and then deletes all relevant notifications in the database.
// It returns an error if there is an issue with the database query.
func (t *NotificationRepositoryImpl) DeleteNotifications(clientId string) error {
	_, err := t.Db.Collection("notifications").DeleteMany(context.Background(), bson.M{"userId": clientId})
	if err != nil {
		return err
	}
	return nil
}

// DeleteAppNotifications deletes all notifications for a given user and appId.
func (t *NotificationRepositoryImpl) DeleteAppNotifications(clientId string, appId string) error {
	appId = strings.TrimSpace(appId)
	appId = strings.Trim(appId, `"'`)
	_, err := t.Db.Collection("notifications").DeleteMany(context.Background(), bson.M{"userId": clientId, "appId": appId})
	if err != nil {
		return err
	}
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
	_, err := t.Db.Collection("notifications").DeleteMany(context.Background(), bson.M{"userId": clientId, "appId": appId, "groupKey": groupKey})
	if err != nil {
		return err
	}
	return nil
}

// DeleteNotification deletes a notification for a given user.
// It takes a clientId and a notificationId as arguments, trims and removes any double quotes from the strings,
// converts the notificationId to an ObjectID, and then deletes the relevant notification in the database.
// It returns an error if the notification is not found or if there is an issue with the database query.
func (t *NotificationRepositoryImpl) DeleteNotification(clientId string, notificationId string) error {
	notificationId = strings.TrimSpace(notificationId)
	notificationId = strings.Trim(notificationId, `"'`)
	objID, err := primitive.ObjectIDFromHex(notificationId)
	if err != nil {
		return err
	}
	_, err = t.Db.Collection("notifications").DeleteOne(context.Background(), bson.M{"userId": clientId, "_id": objID})
	if err != nil {
		return err
	}
	return nil
}
