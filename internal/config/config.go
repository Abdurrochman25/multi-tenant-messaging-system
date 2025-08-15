package config

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/pkg/util"
	"github.com/spf13/viper"
)

type Config struct {
	AppSecret string   `yaml:"app_secret" mapstructure:"app_secret"`
	Database  Database `yaml:"database" mapstructure:"database"`
	RabbitMQ  RabbitMQ `yaml:"rabbitmq" mapstructure:"rabbitmq"`
	Workers   int      `yaml:"workers" mapstructure:"workers"`
}

func NewConfig() Config {
	// First try to load from YAML config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set default values
	viper.SetDefault("workers", 3)
	viper.SetDefault("app_secret", "default-secret")
	viper.SetDefault("database.url", "postgres://admin:admin@localhost:5432/postgres?sslmode=disable")
	viper.SetDefault("rabbitmq.url", "amqp://admin:admin@localhost:5672/")

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		// Fallback to environment variables
		util.LoadEnv(".env")
	}

	// Override with environment variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var config Config

	// Parse config from YAML structure
	config = Config{
		AppSecret: viper.GetString("app_secret"),
		Workers:   viper.GetInt("workers"),
	}

	// Parse database - try URL first, then individual fields
	if dbURL := viper.GetString("database.url"); dbURL != "" {
		config.Database = ParseDatabaseURL(dbURL)
	} else if viper.IsSet("database.host") {
		config.Database = Database{
			DatabaseName: viper.GetString("database.database_name"),
			Host:         viper.GetString("database.host"),
			Port:         viper.GetInt("database.port"),
			Username:     viper.GetString("database.username"),
			Password:     viper.GetString("database.password"),
			Option: map[string]string{
				"sslmode": viper.GetString("database.sslmode"),
			},
		}
	} else {
		// Fallback to individual environment variables
		config.Database = Database{
			DatabaseName: util.GetEnv("PSQL_DBNAME", "postgres"),
			Host:         util.GetEnv("PSQL_HOST", "localhost"),
			Port:         util.GetEnvAsInt("PSQL_PORT", 5432),
			Username:     util.GetEnv("PSQL_USER", "admin"),
			Password:     util.GetEnv("PSQL_PASS", "admin"),
			Option: map[string]string{
				"sslmode": util.GetEnv("PSQL_SSLMODE", "disable"),
			},
		}
	}

	// Parse RabbitMQ - try URL first, then individual fields
	if mqURL := viper.GetString("rabbitmq.url"); mqURL != "" {
		config.RabbitMQ = ParseRabbitMQURL(mqURL)
	} else if viper.IsSet("rabbitmq.host") {
		config.RabbitMQ = RabbitMQ{
			Host:     viper.GetString("rabbitmq.host"),
			Port:     viper.GetInt("rabbitmq.port"),
			Username: viper.GetString("rabbitmq.username"),
			Password: viper.GetString("rabbitmq.password"),
		}
	} else {
		// Fallback to environment variables
		config.RabbitMQ = RabbitMQ{
			Host:     util.GetEnv("RABBITMQ_HOST", "localhost"),
			Port:     util.GetEnvAsInt("RABBITMQ_PORT", 5672),
			Username: util.GetEnv("RABBITMQ_USER", "admin"),
			Password: util.GetEnv("RABBITMQ_PASS", "admin"),
		}
	}

	return config
}

var basepath string

func init() {
	_, currentFile, _, _ := runtime.Caller(0)
	basepath = filepath.Dir(currentFile)
}

func Path(rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}

	return filepath.Join(basepath, rel)
}

func ParseDatabaseURL(dbURL string) Database {
	u, err := url.Parse(dbURL)
	if err != nil {
		return Database{}
	}

	password, _ := u.User.Password()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 5432
	}

	options := make(map[string]string)
	for key, values := range u.Query() {
		if len(values) > 0 {
			options[key] = values[0]
		}
	}

	return Database{
		DatabaseName: strings.TrimPrefix(u.Path, "/"),
		Host:         u.Hostname(),
		Port:         port,
		Username:     u.User.Username(),
		Password:     password,
		Option:       options,
	}
}

func ParseRabbitMQURL(mqURL string) RabbitMQ {
	u, err := url.Parse(mqURL)
	if err != nil {
		return RabbitMQ{}
	}

	password, _ := u.User.Password()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 5672
	}

	return RabbitMQ{
		Host:     u.Hostname(),
		Port:     port,
		Username: u.User.Username(),
		Password: password,
	}
}
