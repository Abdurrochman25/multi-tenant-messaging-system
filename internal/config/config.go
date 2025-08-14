package config

import (
	"path/filepath"
	"runtime"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/pkg/util"
)

type Config struct {
	AppSecret string
	Database  Database
}

func NewConfig() Config {
	util.LoadEnv(".env")

	return Config{
		AppSecret: util.GetEnv("APP_SECRET", ""),
		Database: Database{
			DatabaseName: util.GetEnv("PSQL_DBNAME", "postgres"),
			Host:         util.GetEnv("PSQL_HOST", "postgres"),
			Port:         util.GetEnvAsInt("PSQL_PORT", 5432),
			Username:     util.GetEnv("PSQL_USER", "admin"),
			Password:     util.GetEnv("PSQL_PASS", "admin"),
			Option: map[string]string{
				"sslmode": util.GetEnv("PSQL_SSLMODE", "disable"),
			},
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
