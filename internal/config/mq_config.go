package config

import (
	"fmt"
	"strings"
)

type RabbitMQ struct {
	Username string
	Password string
	Host     string
	Port     int
}

func (r *RabbitMQ) ConnectionString() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("amqp://%s:%s@%s:%d/", r.Username, r.Password, r.Host, r.Port))

	return b.String()
}
