package internal

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/config"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/mq"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/router"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/services"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

func Init() {
	conf := config.NewConfig()

	s := config.NewServer(conf)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.NewDB(ctx); err != nil {
		log.Fatalf("Failed to initialize database; error: %v", err)
	}

	// Init RabbitMQ
	mqUrl := conf.RabbitMQ.ConnectionString()
	mqClient, err := mq.Dial(mqUrl)
	if err != nil {
		log.Fatalf("Failed to initialize rabbitmq; error: %v", err)
	}
	s.RabbitMQ = mqClient.Conn
	mqChannel, err := mqClient.Channel()
	if err != nil {
		log.Fatalf("Failed to initialize rabbitmq; error: %v", err)
	}

	rabbitmqService := services.NewRabbitMQService(mqClient.Conn, mqChannel)
	tenantManager := services.NewTenantManager(s.DB, rabbitmqService)
	messageServices := services.NewMessageService(s.DB, rabbitmqService)

	s.Fiber = fiber.New(fiber.Config{
		Immutable: true,
	})

	s.Fiber.Use(logger.New())

	s.Router = &config.Router{
		Routes: []fiber.Router{
			s.Fiber.Get("/", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			}),
		},
	}

	router.AttachAllRoutes(s, tenantManager, messageServices)

	// Swagger documentation
	s.Fiber.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Prometheus metrics endpoint
	s.Fiber.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// Custom 404 Handler
	s.Fiber.Use(func(c *fiber.Ctx) error {
		return c.SendString("Not Found")
	})

	go func() {
		if err := s.Fiber.Listen(":3000"); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Info("Server closed")
			} else {
				log.Fatalf("Failed to start server, err: %s", err.Error())
			}
		}
	}()

	// Setup graceful shutdown
	shutdownManager := &services.ShutdownManager{
		TenantManager: tenantManager,
		Server:        s.Fiber,
	}
	// Start graceful shutdown handler
	shutdownManager.GracefulShutdown()
}
