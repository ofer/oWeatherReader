package main

// Config represents the application configuration
type Config struct {
	OpenAIBaseURL                 string `json:"openAIBaseURL"`
	OpenAIModel                   string `json:"openAIModel"`
	OpenAIAPIKey                  string `json:"openAIAPIKey"`
	IndoorDeviceModel             string `json:"indoorDeviceModel"`
	OutdoorDeviceModel            string `json:"outdoorDeviceModel"`
	RecommendationIntervalMinutes int    `json:"recommendationIntervalMinutes"`
}

// Global configuration instance
var config Config
