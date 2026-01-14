package consumer

// Package consumer contains the code for the Event Hub notification event consumers.

import (
	"address-book-notification-service/config"
	"address-book-notification-service/data"
	"address-book-notification-service/models"
	clientStore "address-book-notification-service/services"
	notificationService "address-book-notification-service/services/notification"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
)

// StartEventHubConsumer starts the Event Hub consumer for notification events.
// It starts a goroutine for each partition in the Event Hub and reads the events from the partition.
// For each event received, it creates a notification record in the database and sends the notification to the connected client web socket.
func StartEventHubConsumer(ctx context.Context, notificationService notificationService.NotificationService) error {

	cfg := config.LoadConfig()
	connectionString := fmt.Sprintf("%s;EntityPath=%s", cfg.EventHubNameSpaceConString, cfg.EventHubNotificationEventName)

	hub, err := eventhub.NewHubFromConnectionString(connectionString)
	if err != nil {
		return fmt.Errorf("failed to connect to Event Hub: %w", err)
	}
	fmt.Println("Connected to Event Hub")

	// Default consumer group
	runtimeInfo, err := hub.GetRuntimeInformation(ctx)
	if err != nil {
		return err
	}

	for _, partitionID := range runtimeInfo.PartitionIDs {
		go func(pid string) {
			hub.Receive(ctx, pid, func(ctx context.Context, event *eventhub.Event) error {

				fmt.Println("Received event:", string(event.Data))

				var eventData data.EventHubNotificationPayload
				if err := json.Unmarshal(event.Data, &eventData); err != nil {
					log.Println("Invalid message format:", err)
					return nil
				}

				log.Println("Received Event:", eventData)

				m := models.Notification{
					UserId:     eventData.UserId,
					AppId:      eventData.AppId,
					GroupKey:   eventData.GroupKey,
					Message:    eventData.Message,
					Status:     eventData.Status,
					ReadStatus: false,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}

				// Create notification record in database
				recordId, err := notificationService.Create(m)
				if err != nil {
					log.Println("Notification entry insert error:", err)
					return nil
				}

				// Send Notification to connected client web socket
				payload := data.ActionNotification{
					Action: data.Action{Action: "newNotification"},
					Notification: data.Notification{
						Id:        recordId.Hex(),
						UserID:    eventData.UserId,
						AppId:     eventData.AppId,
						GroupKey:  eventData.GroupKey,
						Message:   eventData.Message,
						Status:    eventData.Status,
						CreatedAt: m.CreatedAt,
						UpdatedAt: m.UpdatedAt,
					},
				}
				m.Id = recordId
				clientStore.SendNotificationToUser(payload)
				return nil
			}, eventhub.ReceiveWithLatestOffset())
		}(partitionID)
	}

	select {} // Keep running

}
