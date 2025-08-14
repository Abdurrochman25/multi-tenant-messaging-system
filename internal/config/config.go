package config

import (
	"path/filepath"
	"runtime"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/pkg/util"
)

type RabbitMQ struct {
	Username string
	Password string
	Host     string
	Port     int
}

type Config struct {
	AppSecret string
	Database  Database
	RabbitMQ  RabbitMQ
}

func NewConfig() Config {
	util.LoadEnv(".env")

	return Config{
		AppSecret: util.GetEnv("APP_SECRET", ""),
		Database: Database{
			DatabaseName: util.GetEnv("PSQL_DBNAME", "postgres"),
			Host:         util.GetEnv("PSQL_HOST", "localhost"),
			Port:         util.GetEnvAsInt("PSQL_PORT", 5432),
			Username:     util.GetEnv("PSQL_USER", "admin"),
			Password:     util.GetEnv("PSQL_PASS", "admin"),
			Option: map[string]string{
				"sslmode": util.GetEnv("PSQL_SSLMODE", "disable"),
			},
		},
		RabbitMQ: RabbitMQ{
			Host:     util.GetEnv("RABBITMQ_HOST", "localhost"),
			Port:     util.GetEnvAsInt("RABBITMQ_PORT", 5672),
			Username: util.GetEnv("RABBITMQ_USER", "admin"),
			Password: util.GetEnv("RABBITMQ_PASS", "admin"),
		},
	}
}

var basepath string

func init() {
	_, currentFile, _, _ := runtime.Caller(0)
	basepath = filepath.Dir(currentFile)
}

// Path returns the absolute path the given relative file or directory path,
// relative to the google.golang.org/grpc/examples/data directory in the
// user's GOPATH.  If rel is already absolute, it is returned unmodified.
func Path(rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}

	return filepath.Join(basepath, rel)
}
