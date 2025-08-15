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

func NewTenantHandler(s *config.Server, tm *services.TenantManager) []fiber.Router {
	handler := TenantHandler{tenantManager: tm}

	return []fiber.Router{
		s.Fiber.Post("/v1/tenants", handler.CreateTenant),
		s.Fiber.Delete("/v1/tenants/:id", handler.DeleteTenant),
		s.Fiber.Put("/v1/tenants/:id/config/concurrency", handler.UpdateTenantConfig),
	}
}

// CreateTenant creates a new tenant
// @Summary Create a new tenant
// @Description Create a new tenant with specified configuration
// @Tags tenants
// @Accept json
// @Produce json
// @Param tenant body models.Tenant true "Tenant information"
// @Success 201 {object} models.Tenant
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tenants [post]
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

// DeleteTenant deletes a tenant
// @Summary Delete a tenant
// @Description Delete a tenant and stop its consumer
// @Tags tenants
// @Param id path string true "Tenant ID"
// @Success 200 {object} string
// @Failure 500 {object} map[string]string
// @Router /tenants/{id} [delete]
func (h *TenantHandler) DeleteTenant(c *fiber.Ctx) error {
	tenantID := c.Params("id")

	err := h.tenantManager.DeleteTenant(c.Context(), tenantID)
	if err != nil {
		return c.JSON(fiber.NewError(http.StatusInternalServerError, err.Error()))
	}

	return c.Status(http.StatusOK).JSON("tenant deleted")
}

// UpdateTenantConfig updates tenant configuration
// @Summary Update tenant concurrency configuration
// @Description Update the worker count for a tenant
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Param config body models.TenantConfigRequest true "Configuration"
// @Success 200 {object} string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tenants/{id}/config/concurrency [put]
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
