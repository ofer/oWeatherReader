package main

import (
	"encoding/json"
	"os"
)

// loadConfig loads configuration from config.json file or creates default config
func loadConfig() error {
	file, err := os.Open("config.json")
	if err != nil {
		// Create default config if it doesn't exist
		config = Config{
			OllamaServerURL:               "http://localhost:11434",
			OllamaModel:                   "llama3.2",
			IndoorDeviceModel:             "LaCrosse-TX141W",
			OutdoorDeviceModel:            "LaCrosse-TX141W",
			RecommendationIntervalMinutes: 15,
		}
		return nil
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&config)
}

// saveConfig saves the current configuration to config.json file
func saveConfig() error {
	file, err := os.Create("config.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(&config)
}
