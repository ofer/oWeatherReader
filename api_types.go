package main

import (
	"time"
)

// OpenAIChatMessage represents a single message in an OpenAI chat completion request
type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIChatRequest represents a request to the OpenAI chat completions API
type OpenAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []OpenAIChatMessage `json:"messages"`
}

// OpenAIChatChoice represents a single choice in the OpenAI chat completion response
type OpenAIChatChoice struct {
	Index   int               `json:"index"`
	Message OpenAIChatMessage `json:"message"`
}

// OpenAIChatResponse represents a response from the OpenAI chat completions API
type OpenAIChatResponse struct {
	ID      string             `json:"id"`
	Choices []OpenAIChatChoice `json:"choices"`
}

// AIRecommendationResponse represents the structured response from the AI for climate recommendations
type AIRecommendationResponse struct {
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

// WeatherReportEvent represents a server-sent event when a new weather report is accepted
type WeatherReportEvent struct {
	DbId        uint   `json:"dbId"`
	Time        int64  `json:"time"`
	DeviceModel string `json:"deviceModel"`
}
