package services

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gofiber/fiber/v2"
)

type ShutdownManager struct {
	TenantManager *TenantManager
	Server        *fiber.App
	wg            sync.WaitGroup
}

func (sm *ShutdownManager) GracefulShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down server...")

	// Stop all tenant consumers
	sm.TenantManager.StopAllConsumers()

	// Wait for ongoing processing to complete
	sm.wg.Wait()

	// Shutdown HTTP server
	if err := sm.Server.Shutdown(); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
