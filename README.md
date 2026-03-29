# r2-notify-server

A real-time notification server built with Go and Gin. Your backend pushes notifications to it via a REST API, and connected clients receive them instantly over WebSocket — no polling, no delays.

It handles user authentication, API key management, notification persistence, and WebSocket connection management out of the box.

---

## Features

- ⚡ **Real-time delivery** — notifications are pushed to connected clients instantly via WebSocket
- 🔐 **Secure WebSocket auth** — short-lived single-use tokens keep API keys out of URLs and server logs
- 🗝️ **API key management** — create, list, update, and revoke keys via REST, backed by [Unkey](https://unkey.dev)
- 💾 **Persistent notifications** — stored in MongoDB, delivered on connect so clients never miss anything
- 🔔 **Notification config** — users can enable/disable notifications and the preference is persisted
- 🏓 **Connection health** — server pings clients every 30s and cleans up stale connections automatically
- 📦 **Ready-to-use client packages** — [`r2-notify-client`](https://www.npmjs.com/package/r2-notify-client) and [`r2-notify-react`](https://www.npmjs.com/package/r2-notify-react) handle everything on the frontend

---

## Prerequisites

- Go 1.21+
- MongoDB 4.0+
- Redis 6.0+
- [Unkey](https://unkey.dev) account for API key management

---

## Installation & Setup

### 1. Clone the repository

```bash
git clone https://github.com/sheranthaperera93/r2-notify-server.git
cd r2-notify-server
```

### 2. Configure environment variables

```bash
cp .env.example .env
```

Open `.env` and fill in the values — see the [Environment Variables](#environment-variables) section below.

### 3. Run the server

```bash
go run ./cmd/server/main.go
```

The server starts on the port defined in your `.env` (default `8081`). You should see:

```
r2-notify started on port 8081
```

### Docker

```bash
docker build -t r2-notify-server .
docker run -p 8081:8081 --env-file .env r2-notify-server
```

---

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `PORT` | no | Server port. Default `8081` |
| `ENV` | yes | `development` or `production` |
| `MONGO_URI` | yes | MongoDB connection string |
| `REDIS_URL` | yes | Redis connection string |
| `UNKEY_ROOT_KEY` | yes | Unkey root key for API key validation |
| `ALLOWED_ORIGINS` | yes | Comma-separated allowed WebSocket origins e.g. `http://localhost:5173,https://yourapp.com` |
| `JWT_SECRET` | yes | Secret used to sign authentication tokens |
| `RESEND_API_KEY` | yes | [Resend](https://resend.com) API key for email delivery |
| `RESEND_FROM` | yes | From address for outgoing emails e.g. `noreply@yourapp.com` |

---

## How It Works

```
Your backend  ──POST /notification──►  r2-notify-server  ──WebSocket──►  Browser client
               (API key in header)       (persists + fans out)             (r2-notify-client)
```

1. A user logs in and creates an API key via the dashboard or `/api/v1/keys`
2. The browser client fetches a short-lived WS token by POSTing the API key to `/ws-token`
3. The client opens a WebSocket using that token — the API key never appears in a URL
4. Your backend publishes notifications to `/notification` using the same API key
5. The server persists the notification and immediately pushes it to the connected client

---

## Authentication

### User Authentication

Users register, verify their email, and log in via the `/api/v1/auth` endpoints. Login returns a JWT which is required to access protected endpoints like key management.

### API Keys

API keys are managed via [Unkey](https://unkey.dev). Once a user has a key they can use it to connect over WebSocket and to publish notifications from their backend. Keys can be revoked at any time.

---

## WebSocket Connection

Clients connect in two steps. This keeps the API key out of browser DevTools, server logs, and CDN access logs.

### Step 1 — Acquire a short-lived token

```
POST /ws-token
Authorization: Bearer <api-key>
```

**Response**

```json
{ "token": "abc123xyz..." }
```

The token is valid for **30 seconds** and is **single-use** — it is consumed and deleted the moment the WebSocket connection is established. A captured URL cannot be replayed.

### Step 2 — Open the WebSocket

```
GET /ws?token=<token>
```

On successful connection the server immediately pushes:
- The client's full notification list (`listNotifications`)
- The client's notification config (`listConfigurations`)

> If you're using [`r2-notify-client`](https://www.npmjs.com/package/r2-notify-client) or [`r2-notify-react`](https://www.npmjs.com/package/r2-notify-react), both steps happen automatically — you just provide `serverUrl` and `apiKey`.

---

## Publish a Notification

Send notifications from your own backend. The API key identifies which user receives the notification.

```
POST /notification
X-API-Key: <api-key>
X-App-ID: <your-app-id>
Content-Type: application/json
```

**Request body**

```json
{
  "groupKey": "Pre Allocation",
  "message": "Allocate suppliers FIFO to orders finished.",
  "status": "success"
}
```

| Field | Type | Required | Values |
|---|---|---|---|
| `groupKey` | string | yes | Any string — used to group related notifications |
| `message` | string | yes | The notification text |
| `status` | string | yes | `success` \| `error` \| `warning` \| `info` |

**Example**

```bash
curl -X POST http://localhost:8081/notification \
  -H "X-API-Key: your-api-key" \
  -H "X-App-ID: supply-chain-app" \
  -H "Content-Type: application/json" \
  -d '{"groupKey":"Pre Allocation","message":"Job finished.","status":"success"}'
```

The notification is persisted and immediately pushed to all connected WebSocket clients for that user.

---

## WebSocket Events

### Server → Client

| Event | Payload | When |
|---|---|---|
| `listNotifications` | `Notification[]` | On connect, and after any create/read/delete action |
| `newNotification` | `Notification` | When a new notification is published in real time |
| `listConfigurations` | `NotificationConfig` | On connect, and after config is updated |

### Client → Server

Send a JSON message with an `event` field and optional `data`.

| Event | Data | Description |
|---|---|---|
| `markAsRead` | — | Mark all notifications as read |
| `markAppAsRead` | `{ appId }` | Mark all notifications for an app as read |
| `markGroupAsRead` | `{ appId, groupKey }` | Mark all notifications in a group as read |
| `markNotificationAsRead` | `{ id }` | Mark a single notification as read |
| `deleteNotifications` | — | Delete all notifications |
| `deleteAppNotifications` | `{ appId }` | Delete all notifications for an app |
| `deleteGroupNotifications` | `{ appId, groupKey }` | Delete all notifications in a group |
| `deleteNotification` | `{ id }` | Delete a single notification |
| `reloadNotifications` | — | Re-fetch and push the full notification list |
| `setNotificationStatus` | `{ enableNotification: boolean }` | Enable or disable notifications for this user |

**Example message**

```json
{ "event": "markNotificationAsRead", "data": { "id": "64f3a..." } }
```

---

## Notification Model

```json
{
  "id": "64f3a...",
  "appId": "supply-chain-app",
  "userId": "user_abc123",
  "groupKey": "Pre Allocation",
  "message": "Job finished.",
  "status": "success",
  "readStatus": false,
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-01T00:00:00Z"
}
```

---

## REST API Reference

### Health Check

```
GET /health
→ { "status": "ok", "service": "r2-notify" }
```

### Auth (public)

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/auth/register` | Register a new user |
| `POST` | `/api/v1/auth/verify-email` | Verify email address |
| `POST` | `/api/v1/auth/login` | Login — returns access + refresh tokens |
| `POST` | `/api/v1/auth/refresh` | Refresh access token |
| `POST` | `/api/v1/auth/logout` | Logout |
| `POST` | `/api/v1/auth/forgot-password` | Request password reset email |
| `POST` | `/api/v1/auth/reset-password` | Reset password with token |

### API Keys (JWT required)

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/keys` | Create a new API key |
| `GET` | `/api/v1/keys` | List all your API keys |
| `GET` | `/api/v1/keys/:keyId` | Get details for a specific key |
| `PATCH` | `/api/v1/keys/:keyId` | Update a key |
| `DELETE` | `/api/v1/keys/:keyId` | Revoke a key |

### User (JWT required)

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/user/me` | Get the authenticated user's profile |

---

## Client Packages

If you're connecting from a browser, use the official client packages — they handle the two-step token flow, reconnection, and event handling automatically.

- **[r2-notify-client](https://www.npmjs.com/package/r2-notify-client)** — framework-agnostic TypeScript client
- **[r2-notify-react](https://www.npmjs.com/package/r2-notify-react)** — React provider + hooks

---

## License

MIT © Sherantha Perera