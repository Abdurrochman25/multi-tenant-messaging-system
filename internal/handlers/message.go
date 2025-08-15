package handlers

import (
	"net/http"
	"strconv"

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
		s.Fiber.Get("/v1/messages", handler.GetMessages),
	}
}

// PublishMessage publishes a message to a tenant's queue
// @Summary Publish a message
// @Description Publish a message to a specific tenant's queue
// @Tags messages
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param message body models.MessageRequest true "Message data"
// @Success 202 {object} map[string]interface{}
// @Failure 400 {object} fiber.Error
// @Failure 500 {object} fiber.Error
// @Router /tenants/{tenant_id}/messages [post]
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

// GetMessages retrieves messages with cursor pagination
// @Summary Get messages with pagination
// @Description Retrieve messages using cursor-based pagination
// @Tags messages
// @Produce json
// @Param cursor query string false "Cursor for pagination"
// @Param limit query int false "Number of messages to retrieve (max 100)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} fiber.Error
// @Router /messages [get]
func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
	cursor := c.Query("cursor", "0")
	limitStr := c.Query("limit", "20")
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	cursorID, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil {
		cursorID = 0
	}

	messages, nextCursor, err := h.messageServices.GetMessagesPaginated(c.Context(), cursorID, limit)
	if err != nil {
		return c.JSON(fiber.NewError(http.StatusInternalServerError, "Failed to fetch messages"))
	}

	response := fiber.Map{
		"data": messages,
	}

	if nextCursor > 0 {
		response["next_cursor"] = strconv.FormatInt(nextCursor, 10)
	}

	return c.JSON(response)
}
