package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQService struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRabbitMQService(conn *amqp.Connection, ch *amqp.Channel) *RabbitMQService {
	return &RabbitMQService{conn: conn, ch: ch}
}

func (r *RabbitMQService) CreateTenantQueue(tenantID string) error {
	queueName := fmt.Sprintf("tenant_%s_queue", tenantID)

	_, err := r.ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)

	return err
}

func (r *RabbitMQService) DeleteTenantQueue(tenantID string) error {
	queueName := fmt.Sprintf("tenant_%s_queue", tenantID)
	_, err := r.ch.QueueDelete(queueName, false, false, false)
	return err
}

func (r *RabbitMQService) PublishMessageWithHeaders(ctx context.Context, tenantID string, payload []byte, headers map[string]any) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	queueName := fmt.Sprintf("tenant_%s_queue", tenantID)
	err := r.ch.PublishWithContext(ctxTimeout,
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,
		amqp.Publishing{
			ContentType: fiber.MIMEApplicationJSON,
			Body:        payload,
			Headers:     headers,
		})
	if err != nil {
		log.Printf("Failed to publish a message, err: %s", err.Error())
		return err
	}
	log.Printf(" [x] Sent %s", payload)
	return nil
}
