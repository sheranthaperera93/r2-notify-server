package data

import (
	"time"
)

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

type Event struct {
	Event string `json:"event"`
}

type EventNotification struct {
	Event
	Data Notification `json:"data"`
}

type NotificationList struct {
	Event
	Data []Notification `json:"data"`
}

type NotificationConfig struct {
	Id                 string `json:"id"`
	UserID             string `json:"userId"`
	EnableNotification bool   `json:"enableNotification"`
}

type Configuration struct {
	Event
	Data NotificationConfig `json:"data"`
}

type CreateNotificationRequest struct {
	GroupKey string `validate:"required" json:"groupKey"`
	Message  string `validate:"required" json:"message"`
	Status   string `validate:"required" json:"status"`
}

type GoogleAuthRequest struct {
	Token string `json:"token"`
}

type UserInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
}
