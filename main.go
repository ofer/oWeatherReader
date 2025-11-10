package main

import (
	"fmt"
	"log"
)

// main is the entry point of the application
func main() {
	fmt.Println("Starting oWeatherReader")

	// Load configuration
	if err := loadConfig(); err != nil {
		log.Printf("Failed to load config: %v, using defaults", err)
	}

	db := setupDatabase()
	go rtlMonitor(db)
	go ollamaRecommendationWorker(db)

	r := setupRouter(db)
	// Listen and Server in 0.0.0.0:8080
	r.Run(":6656")
}
