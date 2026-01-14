package models

import "time"

type ClientInfo struct {
	ID                 string    `json:"id"`
	ConnectedAt        time.Time `json:"connectedAt"`
	EnableNotification bool      `json:"enableNotification"`
}
