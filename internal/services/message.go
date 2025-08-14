package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/models"
	"github.com/aarondl/sqlboiler/v4/boil"
)

type MessageService struct {
	db       *sql.DB
	rabbitmq *RabbitMQService
}

func NewMessageService(db *sql.DB, rabbitmq *RabbitMQService) *MessageService {
	return &MessageService{db: db, rabbitmq: rabbitmq}
}

func (s *MessageService) Publish(ctx context.Context, tenantID, messageID string, messageReq *models.MessageRequest) error {
	messagePayload := map[string]any{
		"id":           messageID,
		"tenant_id":    tenantID,
		"type":         messageReq.Type,
		"data":         messageReq.Data,
		"created_at":   time.Now(),
		"scheduled_at": messageReq.ScheduledAt,
	}

	// Convert to JSON
	payload, err := json.Marshal(messagePayload)
	if err != nil {
		return fmt.Errorf("Failed to serialize message")
	}

	// Prepare headers
	headers := make(map[string]any)
	headers["message_id"] = messageID
	headers["tenant_id"] = tenantID
	headers["priority"] = messageReq.Priority
	headers["created_at"] = time.Now().Unix()

	// Handle scheduled messages
	if messageReq.ScheduledAt != nil && messageReq.ScheduledAt.After(time.Now()) {
		headers["scheduled_at"] = messageReq.ScheduledAt.Unix()
	}

	message := &models.Message{
		ID:       messageID,
		TenantID: tenantID,
		Payload:  payload,
	}

	err = message.Insert(ctx, s.db, boil.Infer())
	if err != nil {
		return err
	}

	return s.rabbitmq.PublishMessageWithHeaders(ctx, tenantID, payload, headers)
}
