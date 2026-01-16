package clientStore

import (
	"encoding/json"
	"errors"
	"r2-notify/config"
	"r2-notify/data"
	"r2-notify/logger"
	"r2-notify/models"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	clients      = make(map[string][]*websocket.Conn) // userID -> []connection
	clientsMutex sync.RWMutex
)

// StoreClient adds a new connection to the list of connections for the given user
// and stores the updated models.ClientInfo struct in Redis.
// It is safe to call this function concurrently from multiple goroutines.
func StoreClient(info models.ClientInfo, conn *websocket.Conn) error {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Service",
		Component: "ClientStore",
		Operation: "StoreClient",
		Message:   "Storing client in memory for clientID: " + info.ID,
		UserId:    info.ID,
	})
	clientsMutex.Lock()
	clients[info.ID] = append(clients[info.ID], conn)
	clientsMutex.Unlock()
	// Marshal and store the updated ClientInfo struct in Redis
	data, _ := json.Marshal(info)
	err := config.RDB.Set(config.Ctx, "client:"+info.ID, data, 0).Err()
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Service",
			Component: "ClientStore",
			Operation: "StoreClient",
			Message:   "Failed to store client in Redis for clientID: " + info.ID,
			Error:     err,
			UserId:    info.ID,
		})
		return err
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Service",
		Component: "ClientStore",
		Operation: "StoreClient",
		Message:   "Successfully stored client for clientID: " + info.ID,
		UserId:    info.ID,
	})
	return nil
}

// DeleteClient removes the client with the given ID from the in-memory map and from Redis, where the client's info is stored.
// It is safe to call this function concurrently from multiple goroutines.
func DeleteClient(id string) error {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Service",
		Component: "ClientStore",
		Operation: "DeleteClient",
		Message:   "Deleting client for clientID: " + id,
		UserId:    id,
	})
	clientsMutex.Lock()
	delete(clients, id)
	clientsMutex.Unlock()
	err := config.RDB.Del(config.Ctx, "client:"+id).Err()
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Service",
			Component: "ClientStore",
			Operation: "DeleteClient",
			Message:   "Failed to delete client from Redis for clientID: " + id,
			Error:     err,
			UserId:    id,
		})
		return err
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "Service",
		Component: "ClientStore",
		Operation: "DeleteClient",
		Message:   "Successfully deleted client for clientID: " + id,
		UserId:    id,
	})
	return nil
}

// RemoveConnection removes a single connection from the list of connections for the given user.
// If the last connection is removed, it also removes the user from the in-memory map and from Redis.
// It is safe to call this function concurrently from multiple goroutines.
func RemoveConnection(userId string, conn *websocket.Conn) {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Service",
		Component: "ClientStore",
		Operation: "RemoveConnection",
		Message:   "Removing connection for userId: " + userId,
		UserId:    userId,
	})
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	conns, exists := clients[userId]
	if !exists {
		logger.Log.Warn(logger.LogPayload{
			Service:   "Service",
			Component: "ClientStore",
			Operation: "RemoveConnection",
			Message:   "User not found in clients map for userId: " + userId,
			UserId:    userId,
		})
		return
	}

	// Filter out the closing connection
	remaining := conns[:0]
	for _, c := range conns {
		if c != conn {
			remaining = append(remaining, c)
		}
	}

	if len(remaining) == 0 {
		// No connections left, clean up completely
		delete(clients, userId)
		_ = config.RDB.Del(config.Ctx, "client:"+userId).Err()
		logger.Log.Info(logger.LogPayload{
			Service:   "Service",
			Component: "ClientStore",
			Operation: "RemoveConnection",
			Message:   "Removed last connection and cleaned up client for userId: " + userId,
			UserId:    userId,
		})
	} else {
		clients[userId] = remaining
		logger.Log.Debug(logger.LogPayload{
			Service:   "Service",
			Component: "ClientStore",
			Operation: "RemoveConnection",
			Message:   "Removed connection for userId: " + userId,
			UserId:    userId,
		})
	}
}

// GetClientInfo fetches the client information from Redis by the given user ID.
// It returns the models.ClientInfo struct and an error if the client does not exist.
// It is safe to call this function concurrently from multiple goroutines.
func GetClientInfo(id string) (models.ClientInfo, error) {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Service",
		Component: "ClientStore",
		Operation: "GetClientInfo",
		Message:   "Fetching client info for clientID: " + id,
		UserId:    id,
	})
	val, err := config.RDB.Get(config.Ctx, "client:"+id).Result()
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Service",
			Component: "ClientStore",
			Operation: "GetClientInfo",
			Message:   "Failed to fetch client info from Redis for clientID: " + id,
			Error:     err,
			UserId:    id,
		})
		return models.ClientInfo{}, err
	}
	var clientInfo models.ClientInfo
	if err := json.Unmarshal([]byte(val), &clientInfo); err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Service",
			Component: "ClientStore",
			Operation: "GetClientInfo",
			Message:   "Failed to unmarshal client info for clientID: " + id,
			Error:     err,
			UserId:    id,
		})
		return models.ClientInfo{}, err
	}
	logger.Log.Debug(logger.LogPayload{
		Service:   "Service",
		Component: "ClientStore",
		Operation: "GetClientInfo",
		Message:   "Successfully fetched client info for clientID: " + id,
		UserId:    id,
	})
	return clientInfo, nil
}

// UpdateClientInfo updates the client information stored in Redis for the given ClientInfo.
// It serializes the ClientInfo struct to JSON and stores it under the key "client:<ID>".
// Returns an error if the operation fails.
func UpdateClientInfo(info models.ClientInfo) error {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Service",
		Component: "ClientStore",
		Operation: "UpdateClientInfo",
		Message:   "Updating client info for clientID: " + info.ID,
		UserId:    info.ID,
	})
	data, _ := json.Marshal(info)
	err := config.RDB.Set(config.Ctx, "client:"+info.ID, data, 0).Err()
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Service",
			Component: "ClientStore",
			Operation: "UpdateClientInfo",
			Message:   "Failed to update client info in Redis for clientID: " + info.ID,
			Error:     err,
			UserId:    info.ID,
		})
		return err
	}
	logger.Log.Debug(logger.LogPayload{
		Service:   "Service",
		Component: "ClientStore",
		Operation: "UpdateClientInfo",
		Message:   "Successfully updated client info for clientID: " + info.ID,
		UserId:    info.ID,
	})
	return nil
}

// SendNotificationToUser sends a notification to a user identified by the UserID field in the given
// data.ActionNotification struct. It does not bypass the notification check, meaning the user's
// notification status will be checked before sending the notification. If the user has disabled
// notifications, the function will return an error.
func SendNotificationToUser(payload data.ActionNotification) error {
	return sendToUser(payload.UserID, payload, false)
}

// SendConfigurationToUser sends the user configuration to the user identified by the UserID field
// in the given data.Configuration struct. If bypassNotificationCheck is true, the function will not
// check the user's notification status before sending the configuration. Otherwise, it will check
// the user's notification status and return an error if notifications are disabled.
func SendConfigurationToUser(payload data.Configuration, bypassNotificationCheck bool) error {
	return sendToUser(payload.UserID, payload, bypassNotificationCheck)
}

// SendNotificationListToUser sends a list of notifications to a user identified by the given userID.
// It uses the NotificationList struct to encapsulate the notifications data.
// The function will check the user's notification status before sending.
// Returns an error if the user is not connected or if notifications are disabled.
func SendNotificationListToUser(userID string, notifications data.NotificationList) error {
	return sendToUser(userID, notifications, false)
}

// getConnAndInfo retrieves the websocket connections and the client information for the given user ID.
// If the user is not connected, it returns an error. Otherwise, it returns the connections and the client
// information.
func getConnAndInfo(userID string) ([]*websocket.Conn, *models.ClientInfo, error) {
	conns, ok := clients[userID]
	if !ok {
		return nil, nil, errors.New("user not connected")
	}
	clientInfo, err := GetClientInfo(userID)
	if err != nil {
		return nil, nil, err
	}
	return conns, &clientInfo, nil
}

// sendToUser sends a payload to all active websocket connections for a specified user.
// It locks the clients map for reading and retrieves the user's connections and client information.
// If notifications are disabled for the user and bypassNotificationCheck is false, it returns an error.
// It serializes the payload to JSON and attempts to write it to each connection.
// Connections that fail to receive the message are removed from the active list.
// Returns an error if the user is not connected or if JSON marshalling fails.
func sendToUser(userID string, payload interface{}, bypassNotificationCheck bool) error {
	logger.Log.Debug(logger.LogPayload{
		Service:   "Service",
		Component: "ClientStore",
		Operation: "SendToUser",
		Message:   "Sending payload to userId: " + userID,
		UserId:    userID,
	})
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()
	conns, clientInfo, err := getConnAndInfo(userID)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Service",
			Component: "ClientStore",
			Operation: "SendToUser",
			Message:   "Failed to get client connections for userId: " + userID,
			Error:     err,
			UserId:    userID,
		})
		return err
	}
	if !bypassNotificationCheck && !clientInfo.EnableNotification {
		notifyDisabledErr := errors.New("notifications are disabled for this user")
		logger.Log.Warn(logger.LogPayload{
			Service:   "Service",
			Component: "ClientStore",
			Operation: "SendToUser",
			Message:   "Notifications disabled for userId: " + userID,
			UserId:    userID,
		})
		return notifyDisabledErr
	}
	data, err := json.Marshal(payload)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "Service",
			Component: "ClientStore",
			Operation: "SendToUser",
			Message:   "Failed to marshal payload for userId: " + userID,
			Error:     err,
			UserId:    userID,
		})
		return err
	}
	var activeConns []*websocket.Conn
	for _, conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			logger.Log.Warn(logger.LogPayload{
				Service:   "Service",
				Component: "ClientStore",
				Operation: "SendToUser",
				Message:   "Failed to write message to connection for userId: " + userID,
				Error:     err,
				UserId:    userID,
			})
			continue
		}
		activeConns = append(activeConns, conn)
	}
	// Update with only active connections
	clients[userID] = activeConns
	logger.Log.Debug(logger.LogPayload{
		Service:   "Service",
		Component: "ClientStore",
		Operation: "SendToUser",
		Message:   "Successfully sent payload to userId: " + userID,
		UserId:    userID,
	})
	return nil
}
