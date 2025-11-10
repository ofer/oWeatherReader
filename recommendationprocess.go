package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

// queryOllamaForRecommendation queries the Ollama AI service for climate recommendations
func queryOllamaForRecommendation(db *gorm.DB) (*OllamaRecommendation, error) {
	// Get latest indoor and outdoor temperature reports
	var indoorReport, outdoorReport WeatherReport

	// Get latest indoor report
	result := db.Where("device_model = ?", config.IndoorDeviceModel).Order("db_id desc").First(&indoorReport)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get indoor temperature: %v", result.Error)
	}

	// Get latest outdoor report
	result = db.Where("device_model = ?", config.OutdoorDeviceModel).Order("db_id desc").First(&outdoorReport)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get outdoor temperature: %v", result.Error)
	}

	// Create prompt for Ollama
	prompt := fmt.Sprintf(`You are a smart home automation assistant. Based on the current weather conditions, provide recommendations for air conditioning and window management.

Current conditions:
- Indoor temperature: %.1f°F (%.1f%% humidity)
- Outdoor temperature: %.1f°F (%.1f%% humidity)
- Time: %s

Please respond with ONLY a valid JSON object in this exact format:
{
  "shouldOperateAirConditioner": boolean,
  "temperatureToSetAirConditionerInF": integer,
  "shouldWindowBeOpen": boolean,
  "weatherDescription": "string description of current conditions and reasoning in 2 sentences"
}

Consider factors like:
- Energy efficiency (avoid AC when windows can provide cooling)
- Comfort levels (typical comfort range is 68-78°F)
- Humidity levels
- Temperature differential between indoor and outdoor`,
		indoorReport.TemperatureInF, float64(indoorReport.HumidityInPercentage),
		outdoorReport.TemperatureInF, float64(outdoorReport.HumidityInPercentage),
		time.Now().Format("2006-01-02 15:04:05"))

	log.Printf("Ollama prompt: %s", prompt)
	// Create request to Ollama
	ollamaReq := OllamaRequest{
		Model:  config.OllamaModel,
		Prompt: prompt,
		Stream: false,
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Send request to Ollama
	resp, err := http.Post(config.OllamaServerURL+"/api/generate", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Ollama: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama server returned status: %d", resp.StatusCode)
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama response: %v", err)
	}

	// Parse the JSON response from Ollama
	var recommendation OllamaRecommendationResponse
	cleanResponse := strings.TrimSpace(ollamaResp.Response)

	// Try to extract JSON from the response (in case there's extra text)
	jsonStart := strings.Index(cleanResponse, "{")
	jsonEnd := strings.LastIndex(cleanResponse, "}") + 1
	if jsonStart >= 0 && jsonEnd > jsonStart {
		cleanResponse = cleanResponse[jsonStart:jsonEnd]
	}

	if err := json.Unmarshal([]byte(cleanResponse), &recommendation); err != nil {
		return nil, fmt.Errorf("failed to parse recommendation JSON: %v, response was: %s", err, cleanResponse)
	}

	// Create OllamaRecommendation record
	result_rec := &OllamaRecommendation{
		Time:                              time.Now(),
		ShouldOperateAirConditioner:       recommendation.ShouldOperateAirConditioner,
		TemperatureToSetAirConditionerInF: recommendation.TemperatureToSetAirConditionerInF,
		ShouldWindowBeOpen:                recommendation.ShouldWindowBeOpen,
		WeatherDescription:                recommendation.WeatherDescription,
		IndoorTemperatureF:                indoorReport.TemperatureInF,
		OutdoorTemperatureF:               outdoorReport.TemperatureInF,
	}

	// Save to database
	if err := db.Create(result_rec).Error; err != nil {
		log.Printf("Failed to save recommendation to database: %v", err)
	}

	return result_rec, nil
}

// ollamaRecommendationWorker runs periodically to get AI recommendations
func ollamaRecommendationWorker(db *gorm.DB) {
	ticker := time.NewTicker(time.Duration(config.RecommendationIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	// Run immediately on startup
	log.Println("Running initial Ollama recommendation query...")
	if recommendation, err := queryOllamaForRecommendation(db); err != nil {
		log.Printf("Failed to get initial Ollama recommendation: %v", err)
	} else {
		log.Printf("Initial recommendation: AC=%v, Temp=%d°F, Window=%v",
			recommendation.ShouldOperateAirConditioner,
			recommendation.TemperatureToSetAirConditionerInF,
			recommendation.ShouldWindowBeOpen)
	}

	for range ticker.C {
		log.Println("Querying Ollama for recommendations...")
		if recommendation, err := queryOllamaForRecommendation(db); err != nil {
			log.Printf("Failed to get Ollama recommendation: %v", err)
		} else {
			log.Printf("New recommendation: AC=%v, Temp=%d°F, Window=%v",
				recommendation.ShouldOperateAirConditioner,
				recommendation.TemperatureToSetAirConditionerInF,
				recommendation.ShouldWindowBeOpen)
		}
	}
}
