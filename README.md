# Multi-Tenant Messaging System

A high-performance, scalable messaging system with multi-tenant support, dynamic consumer management, and configurable concurrency. Built with Go, PostgreSQL, and RabbitMQ.

## 🚀 Features

- **Multi-Tenant Architecture**: Isolated message processing per tenant
- **Dynamic Consumer Management**: Real-time scaling of worker pools
- **Configurable Concurrency**: Adjustable worker counts per tenant
- **Message Processing**: Reliable message handling with retry logic
- **Dead Letter Queue**: Failed message management
- **RESTful API**: Complete CRUD operations with Swagger documentation
- **Graceful Shutdown**: Clean resource cleanup
- **Docker Support**: Containerized deployment with Docker Compose

## 🏗 Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   API Server    │────│    RabbitMQ     │────│   Worker Nodes  │
│                 │    │                 │    │                 │
│ • REST API      │    │ • Message Queue │    │ • Message Proc. │
│ • Swagger Docs  │    │ • Routing       │    │ • Auto Scaling  │
│                 │    │ • Dead Letter   │    │ • Retry Logic   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   PostgreSQL    │
                    │                 │
                    │ • Tenant Data   │
                    │ • Messages      │
                    │ • Partitioned   │
                    └─────────────────┘
```

## 📋 Prerequisites

- Go 1.24+
- PostgreSQL 13+
- RabbitMQ 3.8+
- Docker & Docker Compose (optional)

## 🛠 Installation

### Option 1: Docker Compose (Recommended)

1. **Clone the repository**
   ```bash
   git clone https://github.com/Abdurrochman25/multi-tenant-messaging-system.git
   cd multi-tenant-messaging-system
   ```

2. **Start the services**
   ```bash
   docker-compose up -d
   ```

3. **Verify services are running**
   ```bash
   docker-compose ps
   ```

### Option 2: Local Development

1. **Clone and setup**
   ```bash
   git clone https://github.com/Abdurrochman25/multi-tenant-messaging-system.git
   cd multi-tenant-messaging-system
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Setup environment**
   ```bash
   cp .env.example .env
   # Edit .env with your database and RabbitMQ credentials
   ```

4. **Run database migrations**
   ```bash
   # Run your migration scripts
   ```

5. **Start API server**
   ```bash
   go run cmd/api/main.go
   ```

6. **Start worker nodes**
   ```bash
   go run cmd/worker/main.go
   ```

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PSQL_HOST` | PostgreSQL host | `localhost` |
| `PSQL_PORT` | PostgreSQL port | `5432` |
| `PSQL_USER` | PostgreSQL username | `admin` |
| `PSQL_PASS` | PostgreSQL password | `admin123@#` |
| `PSQL_DBNAME` | Database name | `app` |
| `PSQL_SSLMODE` | SSL mode | `disable` |
| `RABBITMQ_HOST` | RabbitMQ host | `localhost` |
| `RABBITMQ_PORT` | RabbitMQ port | `5672` |
| `RABBITMQ_USER` | RabbitMQ username | `guest` |
| `RABBITMQ_PASS` | RabbitMQ password | `guest` |

### Docker Environment

When using Docker Compose, services are automatically configured with the correct environment variables.

## 📚 API Documentation

### Base URL
- Local: `http://localhost:3000`
- Docker: `http://localhost:3000`

### Swagger Documentation
- **URL**: `http://localhost:3000/swagger/`
- Interactive API documentation with request/response examples

### Authentication
```bash
curl -X POST http://localhost:3000/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "your-tenant-id",
    "username": "admin", 
    "password": "password"
  }'
```

### Tenant Management

**Create Tenant**
```bash
curl -X POST http://localhost:3000/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "name": "acme-corp",
    "max_workers": 5
  }'
```

**Update Tenant Concurrency**
```bash
curl -X PUT http://localhost:3000/v1/tenants/{tenant_id}/config/concurrency \
  -H "Content-Type: application/json" \
  -d '{
    "workers": 10
  }'
```

**Delete Tenant**
```bash
curl -X DELETE http://localhost:3000/v1/tenants/{tenant_id}
```

### Message Publishing

**Send Message**
```bash
curl -X POST http://localhost:3000/v1/tenants/{tenant_id}/messages \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email",
    "data": {
      "to": "user@example.com",
      "subject": "Welcome!",
      "body": "Hello World"
    },
    "priority": 1
  }'
```

**Get Messages (Paginated)**
```bash
curl "http://localhost:3000/v1/messages?limit=10&cursor=abc123"
```

## 🧪 Testing

### Unit Tests
```bash
go test ./...
```

### Integration Tests
```bash
# Pause any running RabbitMQ containers first
docker pause <rabbitmq-container-name>

# Run integration tests
go test -v ./integration_test.go

# Resume RabbitMQ container
docker unpause <rabbitmq-container-name>
```

### Load Testing
```bash
# Using Apache Bench
ab -n 1000 -c 10 -T application/json \
   -p message.json \
   http://localhost:3000/v1/tenants/{tenant_id}/messages
```

## 🔍 Monitoring

### Service Endpoints
- **API**: `http://localhost:3000/`
- **Metrics**: `http://localhost:3000/metrics` (Prometheus format)

### RabbitMQ Management
- **URL**: `http://localhost:15672`
- **Credentials**: `guest` / `guest`

### Optional Monitoring Stack
```bash
# Start with monitoring services
docker-compose --profile monitoring up -d

# Access dashboards
# Prometheus: http://localhost:9090
# Grafana: http://localhost:3001 (admin/admin)
```

## 🚀 Deployment

### Docker Production
```bash
# Build images
docker build -f Dockerfile.api -t messaging-api:latest .
docker build -f Dockerfile.worker -t messaging-worker:latest .

# Deploy with production compose
docker-compose -f docker-compose.prod.yml up -d
```

### Scaling Workers
```bash
# Scale worker nodes
docker-compose up -d --scale worker-1=5 --scale worker-2=3
```

### Environment-specific Configs
```bash
# Development
docker-compose -f docker-compose.yml up -d

# Production
docker-compose -f docker-compose.prod.yml up -d

# With monitoring
docker-compose --profile monitoring up -d
```

## 🔧 Development

### Project Structure
```
├── cmd/
│   ├── api/          # API server entry point
│   └── worker/       # Worker entry point
├── internal/
│   ├── config/       # Configuration management
│   ├── handlers/     # HTTP handlers
│   ├── services/     # Business logic
│   ├── models/       # Database models
│   └── middleware/   # HTTP middleware
├── pkg/              # Shared packages
├── docs/             # Swagger documentation
└── integration_test.go
```

### Code Generation
```bash
# Generate database models
make generate-model

# Generate Swagger docs
make swagger
```

### Database Setup
```bash
# Apply schema (manual setup)
psql -h localhost -U admin -d app -f internal/database/migrations/schema.sql

# Or use Docker volume mount (automatic with docker-compose)
# Schema is automatically applied on container startup
docker-compose up -d postgres
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📈 Performance Tips

1. **Optimize Worker Count**: Monitor CPU usage and adjust workers per tenant
2. **Database Partitioning**: Messages are partitioned by tenant_id for better performance
3. **Connection Pooling**: Configure appropriate database connection pools
4. **Message Batching**: Process messages in batches when possible
5. **Monitoring**: Use Prometheus metrics to identify bottlenecks
