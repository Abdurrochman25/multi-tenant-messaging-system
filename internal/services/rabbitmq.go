package services

import (
	"fmt"
	"log"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/config"
	"github.com/gofrs/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQService struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRabbitMQService(s *config.Server) *RabbitMQService {
	ch, err := s.RabbitMQ.Channel()
	if err != nil {
		log.Fatalf("Failed to initialize rabbitmq channel; error: %v", err)
	}

	return &RabbitMQService{conn: s.RabbitMQ, ch: ch}
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

func (r *RabbitMQService) PublishMessage(tenantID uuid.UUID, payload []byte) error {
	queueName := fmt.Sprintf("tenant_%s_queue", tenantID.String())

	return r.ch.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        payload,
		},
	)
}
