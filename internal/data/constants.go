package data

// Application constants
const SERVICE_NAME = "r2-notify-server"
const PRODUCTION_ENV = "production"
const DEFAULT_ORIGINS = "*"

// WebSocket event types
const (
	NEW_NOTIFICATION    = "newNotification"
	LIST_NOTIFICATIONS  = "listNotifications"
	LIST_CONFIGURATIONS = "listConfigurations"
)

// Notification event types
const (
	// Mark as Read events
	MARK_AS_READ              = "markAsRead"
	MARK_APP_AS_READ          = "markAppAsRead"
	MARK_GROUP_AS_READ        = "markGroupAsRead"
	MARK_NOTIFICATION_AS_READ = "markNotificationAsRead"

	// Delete events
	DELETE_NOTIFICATIONS       = "deleteNotifications"
	DELETE_APP_NOTIFICATIONS   = "deleteAppNotifications"
	DELETE_GROUP_NOTIFICATIONS = "deleteGroupNotifications"
	DELETE_NOTIFICATION        = "deleteNotification"

	// Other events
	RELOAD_NOTIFICATIONS    = "reloadNotifications"
	SET_NOTIFICATION_STATUS = "setNotificationStatus"
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
