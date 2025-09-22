package main

import (
	"log"
	"s3browser/internal/app"
)

func main() {
	// Create application instance
	s3App := app.NewApp()

	// Initialize all dependencies
	if err := s3App.Initialize(); err != nil {
		log.Fatal("Failed to initialize application:", err)
	}

	// Run the application
	if err := s3App.Run(":8080"); err != nil {
		log.Fatal("Application error:", err)
	}
}
