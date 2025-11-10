package main

import (
	"time"
)

// OllamaRequest represents a request to the Ollama AI service
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaResponse represents a response from the Ollama AI service
type OllamaResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// OllamaRecommendationResponse represents the structured response from Ollama for climate recommendations
type OllamaRecommendationResponse struct {
	ShouldOperateAirConditioner       bool   `json:"shouldOperateAirConditioner"`
	TemperatureToSetAirConditionerInF int    `json:"temperatureToSetAirConditionerInF"`
	ShouldWindowBeOpen                bool   `json:"shouldWindowBeOpen"`
	WeatherDescription                string `json:"weatherDescription"`
}

// Rtl433WeatherReport represents the raw weather data from rtl_433 software
type Rtl433WeatherReport struct {
	Time          time.Time
	Model         string
	Id            uint32
	Channel       uint8
	Battery_ok    uint8
	Temperature_F *float32
	Temperature_C *float32
	Humidity      float32
	Button        *uint8
	Mic           string
}
