package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// queryOpenAIForRecommendation queries the OpenAI chat completions API for climate recommendations
func queryOpenAIForRecommendation(db *gorm.DB) (*OllamaRecommendation, error) {
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

	// Create system prompt
	systemPrompt := `You are a smart home automation assistant. Based on the current weather conditions, provide recommendations for air conditioning and window management.
Windows should never be open when the air conditioner is operating and the air conditioner should not operate if the windows are open.

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
- Temperature differential between indoor and outdoor`

	// Create user prompt with current conditions
	userPrompt := fmt.Sprintf(`Current weather conditions:
- Indoor temperature: %.1f°F (%.1f%% humidity)
- Outdoor temperature: %.1f°F (%.1f%% humidity)
- Time: %s`,
		indoorReport.TemperatureInF, float64(indoorReport.HumidityInPercentage),
		outdoorReport.TemperatureInF, float64(outdoorReport.HumidityInPercentage),
		time.Now().Format("2006-01-02 15:04:05"))

	log.Printf("OpenAI user prompt: %s", userPrompt)

	// Create request to OpenAI
	openAIReq := OpenAIChatRequest{
		Model: config.OpenAIModel,
		Messages: []OpenAIChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	reqBody, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Build the request
	url := strings.TrimRight(config.OpenAIBaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if config.OpenAIAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.OpenAIAPIKey)
	}

	// Send request
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to OpenAI: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI server returned status: %d", resp.StatusCode)
	}

	var openAIResp OpenAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("failed to decode OpenAI response: %v", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI returned no choices")
	}

	// Parse the JSON response from the assistant message
	var recommendation AIRecommendationResponse
	cleanResponse := strings.TrimSpace(openAIResp.Choices[0].Message.Content)

	// Try to extract JSON from the response (in case there's extra text)
	jsonStart := strings.Index(cleanResponse, "{")
	jsonEnd := strings.LastIndex(cleanResponse, "}") + 1
	if jsonStart >= 0 && jsonEnd > jsonStart {
		cleanResponse = cleanResponse[jsonStart:jsonEnd]
	}

	if err := json.Unmarshal([]byte(cleanResponse), &recommendation); err != nil {
		return nil, fmt.Errorf("failed to parse recommendation JSON: %v, response was: %s", err, cleanResponse)
	}

	// Create OllamaRecommendation record (keeping DB model name for backward compatibility)
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

var (
	recommendationWorkerCancel context.CancelFunc
	recommendationWorkerMutex  sync.Mutex
)

// recommendationWorker runs periodically to get AI recommendations
func recommendationWorker(db *gorm.DB) {
	ctx, cancel := context.WithCancel(context.Background())

	recommendationWorkerMutex.Lock()
	recommendationWorkerCancel = cancel
	recommendationWorkerMutex.Unlock()

	ticker := time.NewTicker(time.Duration(config.RecommendationIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	// Run immediately on startup
	log.Println("Running initial OpenAI recommendation query...")
	if recommendation, err := queryOpenAIForRecommendation(db); err != nil {
		log.Printf("Failed to get initial recommendation: %v", err)
	} else {
		log.Printf("Initial recommendation: AC=%v, Temp=%d°F, Window=%v",
			recommendation.ShouldOperateAirConditioner,
			recommendation.TemperatureToSetAirConditionerInF,
			recommendation.ShouldWindowBeOpen)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Recommendation worker stopped")
			return
		case <-ticker.C:
			log.Println("Querying OpenAI for recommendations...")
			if recommendation, err := queryOpenAIForRecommendation(db); err != nil {
				log.Printf("Failed to get recommendation: %v", err)
			} else {
				log.Printf("New recommendation: AC=%v, Temp=%d°F, Window=%v",
					recommendation.ShouldOperateAirConditioner,
					recommendation.TemperatureToSetAirConditionerInF,
					recommendation.ShouldWindowBeOpen)
			}
		}
	}
}

// restartRecommendationWorker stops the current worker and starts a new one with updated config
func restartRecommendationWorker() {
	recommendationWorkerMutex.Lock()
	defer recommendationWorkerMutex.Unlock()

	if recommendationWorkerCancel != nil {
		log.Println("Stopping current recommendation worker...")
		recommendationWorkerCancel()
		// Give it a moment to stop
		time.Sleep(100 * time.Millisecond)
	}

	log.Println("Starting new recommendation worker with updated config...")
	// Note: We need access to the database here, so we'll need to store it globally
	go recommendationWorker(globalDB)
}
