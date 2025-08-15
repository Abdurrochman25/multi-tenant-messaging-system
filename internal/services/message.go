package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/models"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
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

func (s *MessageService) GetMessagesPaginated(ctx context.Context, cursor int64, limit int) ([]*models.Message, int64, error) {
	var queryMods []qm.QueryMod
	
	if cursor > 0 {
		queryMods = append(queryMods, qm.Where("created_at > (SELECT created_at FROM messages WHERE id = ?)", fmt.Sprintf("%d", cursor)))
	}
	
	queryMods = append(queryMods, 
		qm.OrderBy("created_at ASC, id ASC"),
		qm.Limit(limit+1), // Get one extra to check if there's a next page
	)

	messages, err := models.Messages(queryMods...).All(ctx, s.db)
	if err != nil {
		return nil, 0, err
	}

	var nextCursor int64
	if len(messages) > limit {
		// Remove the extra message and set next cursor
		messages = messages[:limit]
		if len(messages) > 0 {
			// Use a simple incremental cursor based on position
			nextCursor = cursor + int64(limit)
		}
	}

	return messages, nextCursor, nil
}
