package data

// Application constants
const PRODUCTION_ENV = "production"
const DEFAULT_ORIGINS = "http://127.0.0.1:4200,http://localhost:4200"

// WebSocket action types
const (
	NEW_NOTIFICATION    = "newNotification"
	LIST_NOTIFICATIONS  = "listNotifications"
	LIST_CONFIGURATIONS = "listConfigurations"
)

// Notification event types
const (
	// Mark as Read Actions
	MARK_AS_READ              = "markAsRead"
	MARK_APP_AS_READ          = "markAppAsRead"
	MARK_GROUP_AS_READ        = "markGroupAsRead"
	MARK_NOTIFICATION_AS_READ = "markNotificationAsRead"

	// Delete Actions
	DELETE_NOTIFICATIONS       = "deleteNotifications"
	DELETE_APP_NOTIFICATIONS   = "deleteAppNotifications"
	DELETE_GROUP_NOTIFICATIONS = "deleteGroupNotifications"
	DELETE_NOTIFICATION        = "deleteNotification"

	// Other Actions
	RELOAD_NOTIFICATIONS       = "reloadNotifications"
	TOGGLE_NOTIFICATION_STATUS = "toggleNotificationStatus"
)

const (
	LOG_METHOD_FILE  = "file"
	LOG_METHOD_AZURE = "azure"
)

// Log Levels
const (
	DEBUG = "debug"
	INFO  = "info"
	WARN  = "warn"
	ERROR = "error"
)

const CORRELATION_ID = "correlationId"
