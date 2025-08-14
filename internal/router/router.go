package router

import (
	"slices"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/config"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/handlers"
)

func AttachAllRoutes(s *config.Server) {
	slices.Concat(
		s.Router.Routes,
		handlers.NewTenantHandler(s),
	)
}
