package handlers

import (
	"net/http"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/config"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/models"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/services"
	"github.com/aarondl/null/v8"
	"github.com/gofiber/fiber/v2"
)

type TenantHandler struct {
	tenantManager *services.TenantManager
}

func NewTenantHandler(s *config.Server) []fiber.Router {
	rabbitmqService := services.NewRabbitMQService(s)
	tenantManager := services.NewTenantManager(s.DB, rabbitmqService)
	handler := TenantHandler{tenantManager: tenantManager}

	return []fiber.Router{
		s.Fiber.Post("/v1/tenants", handler.CreateTenant),
		s.Fiber.Delete("/v1/tenants/:id", handler.DeleteTenant),
		s.Fiber.Put("/v1/tenants/:id/config/concurrency", handler.UpdateTenantConfig),
	}
}

// [POST] /v1/tenants
func (h *TenantHandler) CreateTenant(c *fiber.Ctx) error {
	tenantRequest := new(models.Tenant)
	if err := c.BodyParser(tenantRequest); err != nil {
		return c.Status(http.StatusBadRequest).JSON(err)
	}

	if tenantRequest.MaxWorkers == null.IntFrom(0) {
		tenantRequest.MaxWorkers = null.IntFrom(3)
	}

	tenant, err := h.tenantManager.CreateTenant(c.Context(), tenantRequest.Name, tenantRequest.MaxWorkers.Int)
	if err != nil {
		return c.JSON(fiber.NewError(http.StatusInternalServerError, err.Error()))
	}

	return c.Status(http.StatusCreated).JSON(tenant)
}

// [DELETE] /v1/tenants/{id}
func (h *TenantHandler) DeleteTenant(c *fiber.Ctx) error {
	tenantID := c.Params("id")

	err := h.tenantManager.DeleteTenant(c.Context(), tenantID)
	if err != nil {
		return c.JSON(fiber.NewError(http.StatusInternalServerError, err.Error()))
	}

	return c.Status(http.StatusOK).JSON("tenant deleted")
}

// [PUT] /v1/tenants/{id}/config/concurreny
func (h *TenantHandler) UpdateTenantConfig(c *fiber.Ctx) error {
	tenantID := c.Params("id")

	config := new(models.TenantConfigRequest)
	if err := c.BodyParser(&config); err != nil {
		return c.JSON(fiber.NewError(http.StatusBadRequest, err.Error()))
	}

	err := h.tenantManager.UpdateConcurrency(c.Context(), tenantID, config)
	if err != nil {
		return c.JSON(fiber.NewError(http.StatusInternalServerError, err.Error()))
	}

	return c.Status(http.StatusOK).JSON("tenant config updated")
}
