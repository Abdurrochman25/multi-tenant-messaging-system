package main

import (
	"context"
	"log"
	"time"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/config"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/control"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/models"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/mq"
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/services"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	_ "github.com/lib/pq"
)

func main() {
	conf := config.NewConfig()
	s := config.NewServer(conf)

	// Init DB
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.NewDB(ctxTimeout); err != nil {
		log.Fatalf("Failed to initialize database; error: %v", err)
	}

	// Init RabbitMQ
	mqUrl := conf.RabbitMQ.ConnectionString()
	mqClient, err := mq.Dial(mqUrl)
	if err != nil {
		log.Fatalf("Failed to initialize rabbitmq; error: %v", err)
	}
	mqChannel, err := mqClient.Channel()
	if err != nil {
		log.Fatalf("Failed to initialize rabbitmq; error: %v", err)
	}

	// Declare control infra
	ctrlCh, _ := mqClient.Channel()
	defer ctrlCh.Close()
	if err := ctrlCh.ExchangeDeclare(control.Exchange, "direct", true, false, false, false, nil); err != nil {
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	rabbitmqService := services.NewRabbitMQService(mqClient.Conn, mqChannel)
	tm := services.NewTenantManager(s.DB, rabbitmqService)

	// Restore tenants from DB (id, workers)
	ctx := context.Background()
	tenants, err := models.Tenants(qm.Select("id", "current_workers"), qm.Where("deleted_at IS NULL")).All(ctx, s.DB)
	if err != nil {
		log.Fatalf("Failed to restore tenants from DB: %v", err)
	}
	log.Printf("tenants : %v", tenants)
	for _, tenant := range tenants {
		worker := tenant.CurrentWorkers.Int
		if worker <= 0 {
			worker = 3
		}
		if err := tm.StartTenant(ctx, tenant.ID, worker); err != nil {
			log.Printf("start tenant, tenant: %s, err: %s", tenant.ID, err.Error())
		}
	}

	// Subscribe to control exchange
	// Each worker has its own control queue (exclusive) bound to all 3 routing keys
	qch, _ := mqClient.Channel()
	defer qch.Close()
	q, err := qch.QueueDeclare("", true, true, true, false, nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, rk := range []string{control.RKCreate, control.RKUpdate, control.RKDelete} {
		if err := qch.QueueBind(q.Name, rk, control.Exchange, false, nil); err != nil {
			log.Fatal(err)
		}
	}

	delivs, err := qch.Consume(q.Name, "", true, true, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	// 3) Main control loop
	for d := range delivs {
		msg, ok := mq.ParseJSON[control.Message](d.Body)
		if !ok {
			continue
		}
		switch d.RoutingKey {
		case control.RKCreate:
			if _, err := tm.CreateTenant(ctx, msg.TenantID, int(msg.Workers)); err != nil {
				log.Printf("create/start, tenant: %s, err: %s", msg.TenantID, err.Error())
			}
		case control.RKUpdate:
			if msg.Workers > 0 {
				tm.UpdateConcurrency(ctx, msg.TenantID, &models.TenantConfigRequest{Workers: int(msg.Workers)})
			}
		case control.RKDelete:
			_ = tm.DeleteTenant(ctx, msg.TenantID)
		}
	}

	// Setup graceful shutdown
	shutdownManager := &services.ShutdownManager{
		TenantManager: tm,
		// Server:        s.Fiber,
	}
	// Start graceful shutdown handler
	shutdownManager.GracefulShutdown()
}
