# R2 Notify Server (Realtime Notification Service)

The R2 Notify Server is a real-time notification service that allows clients to send and receive notifications via WebSockets. It provides a REST API and an Event Hub integration for creating notifications, and supports various notification actions.

---

## Prerequisites
Before running the R2 Notify Server, make sure you have the following prerequisites installed:

Go 1.16 or later
MongoDB 4.0 or later
Azure Event Hubs (optional)

## Getting Started
To get started with the R2 Notify Server, follow these steps:

1. Clone the repository:
```bash
git clone https://github.com/your-username/r2-notify-server.git
```

2. Build the binary
```bash
cd r2-notify-server
go build
```

3. Setup the environment variables

4. Start the server
```bash
./r2-notify-server
```

## Create Notification (REST)

Notifications can be created using a REST API endpoint.

### Endpoint
POST /notification

### Headers
```
X-User-ID: <USER_ID>
X-App-ID: <APP_ID>
Content-Type: application/json
```

### Request Body
```
{
  "groupKey": "Pre Allocation",
  "message": "Allocate suppliers FIFO to orders Finished...",
  "status": "success"
}
```

### Example cURL
```
curl --location 'http://localhost:8081/notification' \
--header 'X-User-ID: RICMAN36' \
--header 'X-App-ID: supply-chain-app' \
--header 'Content-Type: application/json' \
--data '{
  "groupKey": "Pre Allocation",
  "message": "Allocate suppliers FIFO to orders Finished...",
  "status": "success"
}'
```

## Create Notification (Event Hub)

Notifications can also be created by publishing events to the Event Hub.

### Event Hub Name
app-notifications

### Event Payload
```
{
  "appId": "supply-chain-app",
  "userId": "RICMAN36",
  "groupKey": "Pre Allocation",
  "message": "Allocate suppliers FIFO to orders Finished...",
  "status": "success"
}
```

| Field    | Type   | Required |
| -------- | ------ | -------- |
| appId    | string | Yes      |
| userId   | string | Yes      |
| groupKey | string | Yes      |
| message  | string | Yes      |
| status   | string | Yes      |

### Notification

The Notification model represents a single notification. It contains the following fields:

- `_id` : The unique identifier of the notification.
- `appId`: The ID of the app that sent the notification.
- `userId`: The ID of the user who received the notification.
- `groupKey`: The key of the notification group.
- `message`: The content of the notification.
- `status`: The status of the notification (e.g., "success", "error", "warning", "info").
- `readStatus`: Indicates whether the notification has been read.
- `createdAt`: The timestamp when the notification was created.
- `updatedAt`: The timestamp when the notification was last updated.

### Configuration


## Notification Actions
The R2 Notify Server supports various notification actions. Here are some of the available actions:

- markAsRead() - Marks all notifications as read
- markAppAsRead(appId) - Marks all notifications from a specific app as read
- markGroupAsRead(appId, groupKey) - Marks all notifications in a group as read
- markNotificationAsRead(id) - Marks a specific notification as read
- deleteNotifications() - Deletes all notifications
- deleteAppNotifications(appId) - Deletes all notifications from a specific app
- deleteGroupNotifications(appId, groupKey) - Deletes all notifications in a group
- deleteNotification(id) - Deletes a specific notification
- reloadNotifications() - Reloads all notifications from the server
- setNotificationStatus(enable) - Enables or disables notifications

Additionally, the following events are fired by the R2 Notify Server:

- newNotification - Fired when a new notification is received
- listNotifications - Receives a list of notifications
- listConfigurations - Receives notification configurations

## Notes

- Notifications created via REST or Event Hub are persisted and delivered to connected clients in real time via WebSockets.

- createdAt and updatedAt timestamps are managed internally by the service.