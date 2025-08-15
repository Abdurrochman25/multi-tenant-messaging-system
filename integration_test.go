package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/config"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/models"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/mq"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/router"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/services"
	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

type TestSuite struct {
	pool       *dockertest.Pool
	pgResource *dockertest.Resource
	mqResource *dockertest.Resource
	db         *sql.DB
	app        *fiber.App
	dbURL      string
	mqURL      string
}

func TestIntegration(t *testing.T) {
	suite := &TestSuite{}

	// Setup
	if err := suite.Setup(); err != nil {
		t.Fatalf("Failed to setup test suite: %v", err)
	}
	defer suite.Teardown()

	// Run tests
	t.Run("TenantLifecycle", suite.TestTenantLifecycle)
	t.Run("MessagePublishing", suite.TestMessagePublishing)
	t.Run("ConcurrencyUpdate", suite.TestConcurrencyUpdate)
	t.Run("CursorPagination", suite.TestCursorPagination)
}

func (s *TestSuite) Setup() error {
	var err error

	// Create docker pool
	s.pool, err = dockertest.NewPool("")
	if err != nil {
		return fmt.Errorf("could not create docker pool: %w", err)
	}

	// Start PostgreSQL container
	s.pgResource, err = s.pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "13",
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_USER=user",
			"POSTGRES_DB=testdb",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return fmt.Errorf("could not start postgres resource: %w", err)
	}

	// Start RabbitMQ container
	s.mqResource, err = s.pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "rabbitmq",
		Tag:        "3-management",
		Env: []string{
			"RABBITMQ_DEFAULT_USER=admin",
			"RABBITMQ_DEFAULT_PASS=admin",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return fmt.Errorf("could not start rabbitmq resource: %w", err)
	}

	// Connect to PostgreSQL
	s.dbURL = fmt.Sprintf("postgres://user:secret@localhost:%s/testdb?sslmode=disable",
		s.pgResource.GetPort("5432/tcp"))

	s.pool.MaxWait = 120 * time.Second
	if err := s.pool.Retry(func() error {
		s.db, err = sql.Open("postgres", s.dbURL)
		if err != nil {
			return err
		}
		return s.db.Ping()
	}); err != nil {
		return fmt.Errorf("could not connect to postgres: %w", err)
	}

	// Connect to RabbitMQ
	s.mqURL = fmt.Sprintf("amqp://admin:admin@localhost:%s/", s.mqResource.GetPort("5672/tcp"))
	if err := s.pool.Retry(func() error {
		mqClient, err := mq.Dial(s.mqURL)
		if err != nil {
			return err
		}
		defer mqClient.Conn.Close()
		return nil
	}); err != nil {
		return fmt.Errorf("could not connect to rabbitmq: %w", err)
	}

	// Run migrations
	if err := s.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Setup Fiber app
	if err := s.setupApp(); err != nil {
		return fmt.Errorf("failed to setup app: %w", err)
	}

	return nil
}

func (s *TestSuite) runMigrations() error {
	migrations := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`,
		`CREATE TABLE IF NOT EXISTS tenants (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name VARCHAR(255) NOT NULL,
			status VARCHAR(50),
			max_workers INTEGER,
			current_workers INTEGER,
			queue_name VARCHAR(255),
			consumer_tag VARCHAR(255),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			deleted_at TIMESTAMPTZ
		);`,
		`CREATE TABLE IF NOT EXISTS messages (
			id UUID DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL,
			payload JSONB,
			status VARCHAR(50) DEFAULT 'pending',
			retry_count INTEGER DEFAULT 0,
			max_retries INTEGER DEFAULT 3,
			scheduled_at TIMESTAMPTZ,
			processed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			PRIMARY KEY (id, tenant_id)
		) PARTITION BY LIST (tenant_id);`,
		`CREATE TABLE IF NOT EXISTS tenant_configs (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			config_key VARCHAR(255) NOT NULL,
			config_value JSONB,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS message_processing_logs (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			message_id UUID,
			tenant_id UUID,
			worker_id VARCHAR(255),
			status VARCHAR(50),
			error_message TEXT,
			processing_duration_ms INTEGER,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS dead_letter_messages (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			original_message_id UUID,
			tenant_id UUID,
			payload JSONB,
			failure_reason TEXT,
			retry_count INTEGER,
			last_error TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);`,
	}

	for _, migration := range migrations {
		if _, err := s.db.Exec(migration); err != nil {
			return fmt.Errorf("failed to run migration: %w", err)
		}
	}

	return nil
}

func (s *TestSuite) setupApp() error {
	// Parse ports
	pgPort, _ := strconv.Atoi(s.pgResource.GetPort("5432/tcp"))
	mqPort, _ := strconv.Atoi(s.mqResource.GetPort("5672/tcp"))

	// Create test config
	conf := config.Config{
		AppSecret: "test-secret",
		Database: config.Database{
			DatabaseName: "testdb",
			Host:         "localhost",
			Port:         pgPort,
			Username:     "user",
			Password:     "secret",
			Option: map[string]string{
				"sslmode": "disable",
			},
		},
		RabbitMQ: config.RabbitMQ{
			Host:     "localhost",
			Port:     mqPort,
			Username: "admin",
			Password: "admin",
		},
	}

	server := config.NewServer(conf)
	server.DB = s.db

	// Initialize RabbitMQ
	mqClient, err := mq.Dial(s.mqURL)
	if err != nil {
		return err
	}
	server.RabbitMQ = mqClient.Conn

	mqChannel, err := mqClient.Channel()
	if err != nil {
		return err
	}

	// Initialize services
	rabbitmqService := services.NewRabbitMQService(mqClient.Conn, mqChannel)
	tenantManager := services.NewTenantManager(s.db, rabbitmqService)
	messageServices := services.NewMessageService(s.db, rabbitmqService)

	// Setup Fiber app
	s.app = fiber.New(fiber.Config{
		Immutable: true,
	})

	server.Fiber = s.app
	server.Router = &config.Router{
		Routes: []fiber.Router{},
	}
	router.AttachAllRoutes(server, tenantManager, messageServices)

	return nil
}

func (s *TestSuite) Teardown() {
	if s.db != nil {
		s.db.Close()
	}
	if s.pgResource != nil {
		s.pool.Purge(s.pgResource)
	}
	if s.mqResource != nil {
		s.pool.Purge(s.mqResource)
	}
}

func (s *TestSuite) TestTenantLifecycle(t *testing.T) {
	// Create tenant
	tenantData := map[string]interface{}{
		"name":        "test-tenant",
		"max_workers": 5,
	}
	body, _ := json.Marshal(tenantData)

	req, _ := http.NewRequest("POST", "/v1/tenants", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req, 10*1000)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		body := make([]byte, 1024)
		resp.Body.Read(body)
		t.Fatalf("Expected status 201, got %d, body: %s", resp.StatusCode, string(body))
	}

	var tenant models.Tenant
	if err := json.NewDecoder(resp.Body).Decode(&tenant); err != nil {
		t.Fatalf("Failed to decode tenant response: %v", err)
	}

	// Delete tenant
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/v1/tenants/%s", tenant.ID), nil)
	resp, err = s.app.Test(req, 10*1000)
	if err != nil {
		t.Fatalf("Failed to delete tenant: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}
}

func (s *TestSuite) TestMessagePublishing(t *testing.T) {
	// Create tenant first
	tenantData := map[string]interface{}{
		"name":        "message-test-tenant",
		"max_workers": 3,
	}
	body, _ := json.Marshal(tenantData)

	req, _ := http.NewRequest("POST", "/v1/tenants", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req, 10*1000)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	var tenant models.Tenant
	json.NewDecoder(resp.Body).Decode(&tenant)

	// Publish message
	messageData := map[string]interface{}{
		"type": "email",
		"data": map[string]interface{}{
			"to":      "test@example.com",
			"subject": "Test Message",
		},
		"priority": 1,
	}
	body, _ = json.Marshal(messageData)

	req, _ = http.NewRequest("POST", fmt.Sprintf("/v1/tenants/%s/messages", tenant.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = s.app.Test(req, 10*1000)
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("Expected status 202, got %d", resp.StatusCode)
	}
}

func (s *TestSuite) TestConcurrencyUpdate(t *testing.T) {
	// Create tenant first
	tenantData := map[string]interface{}{
		"name":        "concurrency-test-tenant",
		"max_workers": 3,
	}
	body, _ := json.Marshal(tenantData)

	req, _ := http.NewRequest("POST", "/v1/tenants", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req, 10*1000)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	var tenant models.Tenant
	json.NewDecoder(resp.Body).Decode(&tenant)

	// Update concurrency
	configData := map[string]interface{}{
		"workers": 7,
	}
	body, _ = json.Marshal(configData)

	req, _ = http.NewRequest("PUT", fmt.Sprintf("/v1/tenants/%s/config/concurrency", tenant.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = s.app.Test(req, 10*1000)
	if err != nil {
		t.Fatalf("Failed to update concurrency: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}
}

func (s *TestSuite) TestCursorPagination(t *testing.T) {
	// Test pagination endpoint
	req, _ := http.NewRequest("GET", "/v1/messages?limit=10", nil)
	resp, err := s.app.Test(req, 10*1000)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode pagination response: %v", err)
	}

	if _, exists := result["data"]; !exists {
		t.Fatalf("Expected 'data' field in response")
	}
}
