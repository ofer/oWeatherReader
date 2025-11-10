package main

// Config represents the application configuration
type Config struct {
	OllamaServerURL               string `json:"ollamaServerURL"`
	OllamaModel                   string `json:"ollamaModel"`
	IndoorDeviceModel             string `json:"indoorDeviceModel"`
	OutdoorDeviceModel            string `json:"outdoorDeviceModel"`
	RecommendationIntervalMinutes int    `json:"recommendationIntervalMinutes"`
}

// Global configuration instance
var config Config
