package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/models"
	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type TenantConsumer struct {
	TenantID    string
	Channel     *amqp.Channel
	StopChannel chan bool
	WorkerCount *int64
	WorkerPool  chan struct{}
}

type TenantManager struct {
	db        *sql.DB
	rabbitmq  *RabbitMQService
	consumers map[string]*TenantConsumer
	mutex     sync.RWMutex
}

func NewTenantManager(db *sql.DB, rabbitmq *RabbitMQService) *TenantManager {
	return &TenantManager{
		db:        db,
		rabbitmq:  rabbitmq,
		consumers: make(map[string]*TenantConsumer),
	}
}

func (tm *TenantManager) CreateTenant(ctx context.Context, name string, maxWorkers int) (*models.Tenant, error) {
	// 1. Create tenant in database
	tenant := &models.Tenant{
		ID:             uuid.NewString(),
		Name:           name,
		Status:         null.StringFrom("active"),
		MaxWorkers:     null.IntFrom(maxWorkers),
		CurrentWorkers: null.IntFrom(maxWorkers),
		QueueName:      fmt.Sprintf("tenant_%s_queue", uuid.NewString()),
	}

	// 2. Insert to database with partition creation
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	if err := tenant.Insert(ctx, tm.db, boil.Infer()); err != nil {
		return nil, err
	}

	// 3. Create message partition
	err = tm.createMessagePartition(tenant.ID)
	if err != nil {
		// Rollback tenant creation
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// 4. Create RabbitMQ queue
	err = tm.rabbitmq.CreateTenantQueue(tenant.ID)
	if err != nil {
		return nil, err
	}

	// 5. Start consumer
	err = tm.StartConsumer(tenant.ID, maxWorkers)
	if err != nil {
		return nil, err
	}

	return tenant, nil
}

func (tm *TenantManager) DeleteTenant(ctx context.Context, tenantID string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Stop consumer first
	if consumer, exists := tm.consumers[tenantID]; exists {
		// Send shutdown signal
		close(consumer.StopChannel)

		// Close RabbitMQ channel
		if consumer.Channel != nil {
			consumer.Channel.Close()
		}

		// Remove from consumers map
		delete(tm.consumers, tenantID)

		log.Printf("Consumer for tenant %s stopped successfully", tenantID)
	}

	// Delete RabbitMQ queue
	err := tm.rabbitmq.DeleteTenantQueue(tenantID)
	if err != nil {
		log.Printf("Failed to delete queue for tenant %s: %v", tenantID, err)
		// Continue with cleanup even if queue deletion fails
	}

	// Soft delete tenant from database (preserve data for audit)
	tenant := &models.Tenant{
		ID:        tenantID,
		Status:    null.StringFrom("stopped"),
		DeletedAt: null.TimeFrom(time.Now()),
	}

	rowsAffected, err := tenant.Update(ctx, tm.db, boil.Whitelist("status", "deleted_at"))
	if err != nil {
		return fmt.Errorf("failed to delete tenant from database: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found or already deleted")
	}

	log.Printf("Tenant %s deleted successfully", tenantID)

	return nil
}

func (tm *TenantManager) UpdateConcurrency(ctx context.Context, tenantID string, config *models.TenantConfigRequest) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Update database first
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}
	rowsAffected, err := models.TenantConfigs(qm.Where("tenant_id=?", tenantID)).
		UpdateAll(ctx, tm.db, models.M{"config_value": configJSON})
	if err != nil {
		return fmt.Errorf("failed to update tenant in database: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found or deleted")
	}

	// 2. Update consumer worker pool
	consumer, exists := tm.consumers[tenantID]
	if !exists {
		return fmt.Errorf("consumer for tenant %s not found", tenantID)
	}

	currentWorkerCount := int(atomic.LoadInt64(consumer.WorkerCount))

	if config.Workers > currentWorkerCount {
		// Add more workers to pool
		for range config.Workers - currentWorkerCount {
			select {
			case consumer.WorkerPool <- struct{}{}:
				// Worker added successfully
			default:
				// Pool is full, this shouldn't happen but handle gracefully
				break
			}
		}
	} else if config.Workers < currentWorkerCount {
		// Remove workers from pool
		for range currentWorkerCount - config.Workers {
			select {
			case <-consumer.WorkerPool:
				// Worker removed successfully
			default:
				// No workers available to remove, that's ok
				break
			}
		}
	}

	// 3. Update worker count
	atomic.StoreInt64(consumer.WorkerCount, int64(config.Workers))

	log.Printf("Updated tenant %s worker count from %d to %d",
		tenantID, currentWorkerCount, config.Workers)

	return nil
}

func (tm *TenantManager) createMessagePartition(tenantID string) error {
	partitionName := fmt.Sprintf("messages_tenant_%s",
		strings.ReplaceAll(tenantID, "-", ""))

	query := fmt.Sprintf(`
        CREATE TABLE %s PARTITION OF messages 
        FOR VALUES IN ('%s')
    `, partitionName, tenantID)

	_, err := tm.db.Exec(query)
	return err
}

func (tm *TenantManager) StartConsumer(tenantID string, workerCount int) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Create consumer channel
	ch, err := tm.rabbitmq.conn.Channel()
	if err != nil {
		return err
	}

	queueName := fmt.Sprintf("tenant_%s_queue", tenantID)
	consumerTag := fmt.Sprintf("consumer_%s", tenantID)

	// Get message channel
	msgs, err := ch.Consume(
		queueName,   // queue
		consumerTag, // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	// Create worker pool
	workerPool := make(chan struct{}, workerCount)
	for range workerCount {
		workerPool <- struct{}{}
	}

	consumer := &TenantConsumer{
		TenantID:    tenantID,
		Channel:     ch,
		StopChannel: make(chan bool),
		WorkerCount: new(int64),
		WorkerPool:  workerPool,
	}

	atomic.StoreInt64(consumer.WorkerCount, int64(workerCount))
	tm.consumers[tenantID] = consumer

	// Start consumer goroutine
	go tm.runConsumer(consumer, msgs)

	return nil
}

func (tm *TenantManager) runConsumer(consumer *TenantConsumer, msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-consumer.StopChannel:
			return
		case msg := <-msgs:
			// Get worker from pool (blocking if all busy)
			<-consumer.WorkerPool

			// Process message in separate goroutine
			go func(delivery amqp.Delivery) {
				defer func() {
					// Return worker to pool
					consumer.WorkerPool <- struct{}{}
				}()

				tm.processMessage(consumer.TenantID, delivery)
			}(msg)
		}
	}
}

func (tm *TenantManager) processMessage(tenantID string, delivery amqp.Delivery) {
	startTime := time.Now()
	workerID := fmt.Sprintf("worker_%d", time.Now().UnixNano())

	// Parse message ID from headers or generate one
	var messageID string
	if msgIDBytes, ok := delivery.Headers["message_id"]; ok {
		if msgIDStr, ok := msgIDBytes.(string); ok {
			messageID = msgIDStr
		}
	}

	// Log processing start
	tm.logProcessingEvent(messageID, tenantID, workerID, "started", "", 0)

	// Simulate message processing (replace with actual business logic)
	err := tm.handleBusinessLogic(tenantID, messageID, delivery.Body)

	processingDuration := int(time.Since(startTime).Milliseconds())

	if err != nil {
		// Log failure
		tm.logProcessingEvent(messageID, tenantID, workerID, "failed", err.Error(), processingDuration)

		// Check retry logic
		retryCount := tm.getRetryCount(delivery)
		if retryCount < 3 { // Max retries
			// Requeue with delay
			tm.requeueMessage(tenantID, delivery, retryCount+1)
			delivery.Ack(false)
		} else {
			// Send to dead letter
			tm.sendToDeadLetter(tenantID, messageID, delivery.Body, err.Error(), retryCount)
			delivery.Ack(false)
		}
		return
	}

	// Success - update message status and log
	tm.updateMessageStatus(messageID, tenantID, "completed")
	tm.logProcessingEvent(messageID, tenantID, workerID, "completed", "", processingDuration)
	delivery.Ack(false)
}

func (tm *TenantManager) handleBusinessLogic(tenantID, messageID string, payload []byte) error {
	// This is where you implement your actual message processing logic
	// For example:

	// 1. Parse the payload
	var messageData map[string]any
	if err := json.Unmarshal(payload, &messageData); err != nil {
		return fmt.Errorf("failed to parse message payload: %w", err)
	}

	// 2. Store message in database
	query := `
        INSERT INTO messages (id, tenant_id, payload, status, scheduled_at, processed_at)
        VALUES ($1, $2, $3, 'processing', $4, NOW())
        ON CONFLICT (id, tenant_id) 
        DO UPDATE SET status = 'processing', processed_at = NOW()
    `
	_, err := tm.db.Exec(query, messageID, tenantID, payload, time.Now())
	if err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	// 3. Simulate some processing work
	time.Sleep(100 * time.Millisecond)

	// 4. Example business logic based on message type
	if msgType, ok := messageData["type"].(string); ok {
		switch msgType {
		case "email":
			return tm.processEmailMessage(tenantID, messageData)
		case "webhook":
			return tm.processWebhookMessage(tenantID, messageData)
		case "notification":
			return tm.processNotificationMessage(tenantID, messageData)
		default:
			return tm.processGenericMessage(tenantID, messageData)
		}
	}

	return nil
}

func (tm *TenantManager) processEmailMessage(tenantID string, data map[string]interface{}) error {
	// Implement email processing logic
	// Example: send email, update status, etc.
	return nil
}

func (tm *TenantManager) processWebhookMessage(tenantID string, data map[string]interface{}) error {
	// Implement webhook processing logic
	// Example: make HTTP request, handle response, etc.
	return nil
}

func (tm *TenantManager) processNotificationMessage(tenantID string, data map[string]interface{}) error {
	// Implement notification processing logic
	// Example: send push notification, SMS, etc.
	return nil
}

func (tm *TenantManager) processGenericMessage(tenantID string, data map[string]interface{}) error {
	// Default processing logic
	return nil
}

func (tm *TenantManager) logProcessingEvent(messageID, tenantID string, workerID, status, errorMsg string, duration int) {
	query := `
        INSERT INTO message_processing_logs 
        (message_id, tenant_id, worker_id, status, error_message, processing_duration_ms)
        VALUES ($1, $2, $3, $4, $5, $6)
    `
	tm.db.Exec(query, messageID, tenantID, workerID, status,
		sql.NullString{String: errorMsg, Valid: errorMsg != ""},
		sql.NullInt64{Int64: int64(duration), Valid: duration > 0})
}

func (tm *TenantManager) updateMessageStatus(messageID, tenantID string, status string) {
	query := `
        UPDATE messages 
        SET status = $1, processed_at = NOW() 
        WHERE id = $2 AND tenant_id = $3
    `
	tm.db.Exec(query, status, messageID, tenantID)
}

func (tm *TenantManager) getRetryCount(delivery amqp.Delivery) int {
	if retryBytes, ok := delivery.Headers["retry_count"]; ok {
		if retryCount, ok := retryBytes.(int); ok {
			return retryCount
		}
	}
	return 0
}

func (tm *TenantManager) requeueMessage(tenantID string, delivery amqp.Delivery, retryCount int) {
	queueName := fmt.Sprintf("tenant_%s_queue", tenantID)

	// Add retry count to headers
	headers := delivery.Headers
	if headers == nil {
		headers = make(amqp.Table)
	}
	headers["retry_count"] = retryCount
	headers["retry_timestamp"] = time.Now().Unix()

	// Publish with delay (exponential backoff)
	delay := time.Duration(retryCount*retryCount) * time.Second
	go func() {
		time.Sleep(delay)
		tm.rabbitmq.ch.Publish(
			"",        // exchange
			queueName, // routing key
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				ContentType: delivery.ContentType,
				Body:        delivery.Body,
				Headers:     headers,
			},
		)
	}()
}

func (tm *TenantManager) sendToDeadLetter(tenantID, messageID string, payload []byte, errorMsg string, retryCount int) {
	query := `
        INSERT INTO dead_letter_messages 
        (original_message_id, tenant_id, payload, failure_reason, retry_count, last_error)
        VALUES ($1, $2, $3, $4, $5, $6)
    `
	tm.db.Exec(query, messageID, tenantID, payload, "Max retries exceeded", retryCount, errorMsg)
}
