package main

//go:generate sqlboiler --wipe psql
//go:generate swag init

// @title Multi-Tenant Messaging System API
// @version 1.0
// @description A messaging system with multi-tenant support, dynamic consumer management, and configurable concurrency
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email support@example.com
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @host localhost:3000
// @BasePath /v1
import (
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal"
	_ "github.com/lib/pq"
)

func main() {
	internal.Init()
}
