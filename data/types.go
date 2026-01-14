package data

import "time"

type EventHubNotificationPayload struct {
	AppId    string `validate:"required" json:"appId"`
	UserId   string `validate:"required" json:"userId"`
	GroupKey string `validate:"required" json:"groupKey"`
	Message  string `validate:"required" json:"message"`
	Status   string `validate:"required" json:"status"`
}

type Notification struct {
	Id         string    `json:"id"`
	AppId      string    `json:"appId"`
	UserID     string    `json:"userId"`
	GroupKey   string    `json:"groupKey"`
	Message    string    `json:"message"`
	ReadStatus bool      `json:"readStatus"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type NotificationStatusUpdate struct {
	Id     string `json:"id"`
	AppId  string `json:"appId"`
	UserId string `json:"userId"`
	Status string `json:"status"`
}

type Action struct {
	Action string `json:"action"`
}

type ActionNotification struct {
	Action
	Notification
}

type NotificationList struct {
	Action
	Data []Notification `json:"data"`
}

type Configuration struct {
	Action
	Id                 string `json:"id"`
	UserID             string `json:"userId"`
	EnableNotification bool   `json:"enableNotification"`
}

type CreateNotificationRequest struct {
	GroupKey string `validate:"required" json:"groupKey"`
	Message  string `validate:"required" json:"message"`
	Status   string `validate:"required" json:"status"`
}
