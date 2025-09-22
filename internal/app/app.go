package app

import (
	"log"
	"os"
	"os/signal"
	"s3browser/internal/di"
	"s3browser/internal/server"
	"syscall"
)

// App represents the application
type App struct {
	container *di.Container
	server    *server.Server
}

// NewApp creates a new application instance
func NewApp() *App {
	container := di.NewContainer()
	return &App{
		container: container,
	}
}

// Initialize initializes all dependencies
func (a *App) Initialize() error {
	if err := a.container.Initialize(); err != nil {
		return err
	}

	// Start scheduler
	if err := a.container.StartScheduler(); err != nil {
		return err
	}

	// Create and setup server
	a.server = server.NewServer(a.container)
	a.server.SetupRoutes()

	return nil
}

// Run starts the application
func (a *App) Run(addr string) error {
	// Start server in goroutine
	go func() {
		if err := a.server.Start(addr); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	return a.Shutdown()
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown() error {
	// Stop scheduler
	if err := a.container.StopScheduler(); err != nil {
		log.Printf("Error stopping scheduler: %v", err)
	}

	// Close all queues
	a.container.CloseQueues()

	// Shutdown server
	if a.server != nil {
		if err := a.server.Shutdown(); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
	}

	return nil
}

// Container returns the DI container
func (a *App) Container() *di.Container {
	return a.container
}

// Server returns the server instance
func (a *App) Server() *server.Server {
	return a.server
}
