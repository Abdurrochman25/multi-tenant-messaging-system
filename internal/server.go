package internal

import (
	"context"
	"time"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func Init() {
	conf := config.NewConfig()

	s := config.NewServer(conf)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	if err := s.NewDB(ctx); err != nil {
		cancel()
		log.Fatalf("Failed to initialize database; error: %v", err)
	}
	cancel()

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

	// Custom 404 Handler
	s.Fiber.Use(func(c *fiber.Ctx) error {
		return c.SendString("Not Found")
	})
	log.Fatal(s.Fiber.Listen(":3000"))
}
