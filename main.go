package main

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// Global database instance for dynamic config updates
var globalDB *gorm.DB

// Global event hub for broadcasting weather report events
var eventHub *EventHub

// main is the entry point of the application
func main() {
	fmt.Println("Starting oWeatherReader")

	// Load configuration
	if err := loadConfig(); err != nil {
		log.Printf("Failed to load config: %v, using defaults", err)
	}

	db := setupDatabase()
	globalDB = db // Store globally for dynamic config updates

	eventHub = NewEventHub()
	defer eventHub.Shutdown()

	go rtlMonitor(db, eventHub)
	go recommendationWorker(db)

	r := setupRouter(db, eventHub)
	// Listen and Server in 0.0.0.0:8080
	r.Run(":6656")
}
