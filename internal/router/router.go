package router

import (
	"slices"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/config"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/handlers"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/services"
)

func AttachAllRoutes(s *config.Server, tm *services.TenantManager, ms *services.MessageService) {
	slices.Concat(
		s.Router.Routes,
		handlers.NewTenantHandler(s, tm),
		handlers.NewMessageHandler(s, ms),
	)
}
