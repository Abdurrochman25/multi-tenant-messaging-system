package mq

import (
	"github.com/gofiber/fiber/v2/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	Conn *amqp.Connection
}

func Dial(url string) (*Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Errorf("Failed to initialize rabbitmq; error: %v", err)
		return nil, err
	}

	return &Client{Conn: conn}, nil
}

func (c *Client) Channel() (*amqp.Channel, error) {
	ch, err := c.Conn.Channel()
	if err != nil {
		log.Errorf("Failed to initialize rabbitmq channel; error: %v", err)
		return nil, err
	}
	return ch, nil
}
