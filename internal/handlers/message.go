package handlers

import (
	"net/http"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/config"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/models"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type MessageHandler struct {
	messageServices *services.MessageService
}

func NewMessageHandler(s *config.Server, ms *services.MessageService) []fiber.Router {
	handler := MessageHandler{
		messageServices: ms,
	}

	return []fiber.Router{
		s.Fiber.Post("/v1/tenants/:tenant_id/messages", handler.PublishMessage),
	}
}

// [POST] /v1/tenants/:tenant_id/messages
func (h *MessageHandler) PublishMessage(c *fiber.Ctx) error {
	tenantID := c.Params("tenant_id")

	messageReq := new(models.MessageRequest)
	if err := c.BodyParser(messageReq); err != nil {
		return c.JSON(fiber.NewError(http.StatusBadRequest, err.Error()))
	}

	messageID := uuid.NewString()

	// Publish to RabbitMQ
	err := h.messageServices.Publish(c.Context(), tenantID, messageID, messageReq)
	if err != nil {
		return c.JSON(fiber.NewError(http.StatusInternalServerError, "Failed to publish message"))
	}

	return c.Status(http.StatusAccepted).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "queued",
		"tenant_id":  tenantID,
	})
}
