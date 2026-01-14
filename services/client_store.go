package clientStore

import (
	"address-book-notification-service/config"
	"address-book-notification-service/data"
	"address-book-notification-service/models"
	"encoding/json"
	"errors"
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
	clientsMutex.Lock()
	clients[info.ID] = append(clients[info.ID], conn)
	clientsMutex.Unlock()
	// Marshal and store the updated ClientInfo struct in Redis
	data, _ := json.Marshal(info)
	return config.RDB.Set(config.Ctx, "client:"+info.ID, data, 0).Err()
}

// DeleteClient removes the client with the given ID from the in-memory map and from Redis, where the client's info is stored.
// It is safe to call this function concurrently from multiple goroutines.
func DeleteClient(id string) error {
	clientsMutex.Lock()
	delete(clients, id)
	clientsMutex.Unlock()
	return config.RDB.Del(config.Ctx, "client:"+id).Err()
}

// RemoveConnection removes a single connection from the list of connections for the given user.
// If the last connection is removed, it also removes the user from the in-memory map and from Redis.
// It is safe to call this function concurrently from multiple goroutines.
func RemoveConnection(userId string, conn *websocket.Conn) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	conns, exists := clients[userId]
	if !exists {
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
	} else {
		clients[userId] = remaining
	}
}

// GetClientInfo fetches the client information from Redis by the given user ID.
// It returns the models.ClientInfo struct and an error if the client does not exist.
// It is safe to call this function concurrently from multiple goroutines.
func GetClientInfo(id string) (models.ClientInfo, error) {
	val, err := config.RDB.Get(config.Ctx, "client:"+id).Result()
	if err != nil {
		return models.ClientInfo{}, err
	}
	var clientInfo models.ClientInfo
	if err := json.Unmarshal([]byte(val), &clientInfo); err != nil {
		return models.ClientInfo{}, err
	}
	return clientInfo, nil
}

// UpdateClientInfo updates the client information stored in Redis for the given ClientInfo.
// It serializes the ClientInfo struct to JSON and stores it under the key "client:<ID>".
// Returns an error if the operation fails.
func UpdateClientInfo(info models.ClientInfo) error {
	data, _ := json.Marshal(info)
	return config.RDB.Set(config.Ctx, "client:"+info.ID, data, 0).Err()
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
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()
	conns, clientInfo, err := getConnAndInfo(userID)
	if err != nil {
		return err
	}
	if !bypassNotificationCheck && !clientInfo.EnableNotification {
		return errors.New("notifications are disabled for this user")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var activeConns []*websocket.Conn
	for _, conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			continue
		}
		activeConns = append(activeConns, conn)
	}
	// Update with only active connections
	clients[userID] = activeConns
	return nil
}
