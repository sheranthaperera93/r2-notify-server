# Notify Stream (Realtime Notification Service)

Notification Hub is a microservice responsible for creating, managing, and delivering real-time notifications to clients via WebSockets.

---

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

## Notes

- Notifications created via REST or Event Hub are persisted and delivered to connected clients in real time via WebSockets.

- createdAt and updatedAt timestamps are managed internally by the service.