package consumer

// Package consumer contains the code for the Event Hub notification event consumers.

import (
	"context"
	"encoding/json"
	"fmt"
	"r2-notify/config"
	"r2-notify/data"
	"r2-notify/logger"
	"r2-notify/models"
	clientStore "r2-notify/services"
	notificationService "r2-notify/services/notification"
	"r2-notify/utils"
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
	logger.Log.Debug(logger.LogPayload{
		Message:   "Connected to Event Hub",
		Service:   "Azure EventHub",
		Component: "EventHub Consumer",
		Operation: "StartEventHubConsumer",
	})

	// Default consumer group
	runtimeInfo, err := hub.GetRuntimeInformation(ctx)
	if err != nil {
		return err
	}

	for _, partitionID := range runtimeInfo.PartitionIDs {
		go func(pid string) {
			hub.Receive(ctx, pid, func(ctx context.Context, event *eventhub.Event) error {

				correlationId := utils.GenerateUUID()

				logger.Log.Debug(logger.LogPayload{
					Message:       fmt.Sprintf("Received event from Event Hub %s", string(event.Data)),
					Service:       "Azure EventHub",
					Component:     "EventHub Consumer",
					Operation:     "OnEventReceived",
					CorrelationId: correlationId,
				})

				var eventData data.EventHubNotificationPayload
				if err := json.Unmarshal(event.Data, &eventData); err != nil {
					logger.Log.Error(logger.LogPayload{
						Message:       "Invalid message format",
						Service:       "Azure EventHub",
						Component:     "EventHub Consumer",
						Operation:     "OnEventReceived",
						Error:         err,
						CorrelationId: correlationId,
					})
					return nil
				}
				// Prepare notification model
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
					logger.Log.Error(logger.LogPayload{
						Message:       "Notification entry insert error",
						Service:       "Azure EventHub",
						Component:     "EventHub Consumer",
						Operation:     "OnEventReceived",
						Error:         err,
						CorrelationId: correlationId,
					})
					return nil
				}

				// Send Notification to connected client web socket
				payload := data.ActionNotification{
					Action: data.Action{Action: data.NEW_NOTIFICATION},
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

				logger.Log.Info(logger.LogPayload{
					Message:       fmt.Sprintf("Sending notification to user %v", m),
					Service:       "Azure EventHub",
					Component:     "EventHub Consumer",
					Operation:     "OnEventReceived",
					CorrelationId: correlationId,
				})

				return nil
			}, eventhub.ReceiveWithLatestOffset())
		}(partitionID)
	}

	<-ctx.Done()
	logger.Log.Info(logger.LogPayload{
		Message:   "Shutting down event hub consumer",
		Service:   "Azure EventHub",
		Component: "EventHub Consumer",
		Operation: "Shutdown EventHub Consumer",
	})
	hub.Close(context.Background())

	return nil
}
