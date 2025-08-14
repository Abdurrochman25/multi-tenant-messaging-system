package config

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Router struct {
	Routes []fiber.Router
}

type Server struct {
	Config   Config
	DB       *sql.DB
	RabbitMQ *amqp.Connection
	Fiber    *fiber.App
	Router   *Router
}

func NewServer(config Config) *Server {
	return &Server{
		Config: config,
		DB:     nil,
		Fiber:  nil,
	}
}

func (s *Server) NewDB(ctx context.Context) error {
	db, err := sql.Open("postgres", s.Config.Database.ConnectionString())
	if err != nil {
		return err
	}

	if err := db.PingContext(ctx); err != nil {
		return err
	}

	s.DB = db
	boil.SetDB(db)
	boil.DebugMode = true

	log.Info("database successfully connected")
	return nil
}

func (s *Server) NewRabbitMQ() {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		s.Config.RabbitMQ.Username, s.Config.RabbitMQ.Password,
		s.Config.RabbitMQ.Host, s.Config.RabbitMQ.Port,
	)
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("Failed to initialize rabbitmq; error: %v", err)
	}
	s.RabbitMQ = conn

	log.Info("rabbitmq successfully connected")
}
