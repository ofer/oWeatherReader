package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type WeatherReport struct {
	DbId                 uint      `gorm:"primaryKey;autoIncrement"`
	Time                 time.Time `gorm:"index"`
	DeviceModel          string    `gorm:"index"`
	TemperatureInF       float32
	HumidityInPercentage uint8
}

type DeviceModel struct {
	DbId        uint `gorm:"primaryKey;autoIncrement"`
	DeviceModel string
	Name        string
}

type DeviceModelCount struct {
	DeviceModel string
	ReportCount uint64
	Name        string
}

type OllamaRecommendation struct {
	DbId                              uint      `gorm:"primaryKey;autoIncrement"`
	Time                              time.Time `gorm:"index"`
	ShouldOperateAirConditioner       bool
	TemperatureToSetAirConditionerInF int
	ShouldWindowBeOpen                bool
	WeatherDescription                string
	IndoorTemperatureF                float32
	OutdoorTemperatureF               float32
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type OllamaRecommendationResponse struct {
	ShouldOperateAirConditioner       bool   `json:"shouldOperateAirConditioner"`
	TemperatureToSetAirConditionerInF int    `json:"temperatureToSetAirConditionerInF"`
	ShouldWindowBeOpen                bool   `json:"shouldWindowBeOpen"`
	WeatherDescription                string `json:"weatherDescription"`
}

type Config struct {
	OllamaServerURL               string `json:"ollamaServerURL"`
	OllamaModel                   string `json:"ollamaModel"`
	IndoorDeviceModel             string `json:"indoorDeviceModel"`
	OutdoorDeviceModel            string `json:"outdoorDeviceModel"`
	RecommendationIntervalMinutes int    `json:"recommendationIntervalMinutes"`
}

var config Config

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

func getWeatherReportsByModel(c *gin.Context, db *gorm.DB) {
	model := c.Param("model")
	var weatherReports []WeatherReport
	threeDaysAgo := time.Now().AddDate(0, 0, -5)
	result := db.Where("device_model = ? AND time > ?", model, threeDaysAgo).Find(&weatherReports)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve weather reports"})
		return
	}
	c.JSON(http.StatusOK, weatherReports)
}

func setupRouter(db *gorm.DB) *gin.Engine {
	// Disable Console Color
	// gin.DisableConsoleColor()
	r := gin.Default()

	// Serve frontend static files, the ui directory
	r.NoRoute(func(c *gin.Context) {
		dir, file := path.Split(c.Request.RequestURI)
		ext := filepath.Ext(file)
		if file == "" || ext == "" {
			c.File("./ui/dist/ui/index.html")
		} else {
			c.File("./ui/dist/ui/" + path.Join(dir, file))
		}
	})

	// Ping test
	r.GET("/reports/latest", func(c *gin.Context) {
		getLatestWeatherReport(c, db)
	})

	r.GET("/reports/:model", func(c *gin.Context) {
		getWeatherReportsByModel(c, db)
	})

	r.GET("/models", func(c *gin.Context) {
		getModels(c, db)
	})

	r.GET("/recommendations/latest", func(c *gin.Context) {
		getLatestRecommendation(c, db)
	})

	return r
}

func getModels(c *gin.Context, db *gorm.DB) {
	var deviceModels []DeviceModelCount
	// the device model count is a mix of the device model table and a count of the weather reports, so we need to do a join
	result := db.Table("device_models").Select("device_models.device_model, device_models.name, count(weather_reports.device_model) as report_count").Joins("left join weather_reports on device_models.device_model = weather_reports.device_model").Group("device_models.device_model").Find(&deviceModels)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve device models"})
		return
	}
	c.JSON(http.StatusOK, deviceModels)
}

func getLatestWeatherReport(c *gin.Context, db *gorm.DB) {
	var weatherReport WeatherReport
	result := db.Order("db_id desc").First(&weatherReport)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve latest weather report"})
		return
	}
	c.JSON(http.StatusOK, weatherReport)
}

func getLatestRecommendation(c *gin.Context, db *gorm.DB) {
	var recommendation OllamaRecommendation
	result := db.Order("db_id desc").First(&recommendation)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve latest recommendation"})
		return
	}
	c.JSON(http.StatusOK, recommendation)
}

func rtlMonitor(db *gorm.DB) {
	fmt.Println("Running rtl_433")
	command := exec.Command("/usr/bin/rtl_433", "-f", "433000000", "-F", "json", "-M", "time:iso:utc:tz")
	stdout, err := command.StdoutPipe()

	reader := bufio.NewReader(stdout)

	// if there is an error with our execution
	// handle it here
	if err != nil {
		log.Fatal("Stdout Pipe:", err)
	}

	err = command.Start()

	if err != nil {
		log.Fatal("Start command:", err)
	}

	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("Read Error:", err)
			return
		}
		log.Println(str)
		var weatherReport WeatherReport
		var rtl433WeatherReport Rtl433WeatherReport
		err = json.Unmarshal([]byte(str), &rtl433WeatherReport)

		if err != nil {
			log.Println("Unmarshal Error:", err)
			continue
		}

		weatherReport.Time = rtl433WeatherReport.Time
		weatherReport.DeviceModel = rtl433WeatherReport.Model

		// convert to F if necessary
		if rtl433WeatherReport.Temperature_F != nil {
			weatherReport.TemperatureInF = *rtl433WeatherReport.Temperature_F
		} else {
			if rtl433WeatherReport.Temperature_C != nil {
				weatherReport.TemperatureInF = *rtl433WeatherReport.Temperature_C*1.8 + 32
			} else {
				continue
			}
		}
		weatherReport.HumidityInPercentage = uint8(rtl433WeatherReport.Humidity)

		// check whether the device exists in the database, if it doesn't, add it
		checkForDeviceModel(db, weatherReport)

		var shouldIgnoreReport = false
		// find if the last reported humdity is 1 and the new one is 99, if so, ignore it
		var lastWeatherReport WeatherReport
		result := db.Where("device_model = ?", weatherReport.DeviceModel).Order("db_id desc").First(&lastWeatherReport)
		if result.Error != nil {
			log.Println("Failed to retrieve last weather report:", result.Error)
		} else {
			if lastWeatherReport.HumidityInPercentage < 5 && weatherReport.HumidityInPercentage == 99 {
				log.Println("deciding on proper humidity due to erroneous humidity report")
				if lastWeatherReport.TemperatureInF > 70 {
					log.Println("temp is > 80, setting humidity to 1")
					weatherReport.HumidityInPercentage = 1
				} else {
					shouldIgnoreReport = true
				}
			}
		}

		// find if this report already exists in the database
		var existingWeatherReport WeatherReport
		result = db.Where("time = ? AND device_model = ?", weatherReport.Time, weatherReport.DeviceModel).First(&existingWeatherReport)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		} else {
			// if it does, ignore it, else create one
			if existingWeatherReport.TemperatureInF != weatherReport.TemperatureInF ||
				existingWeatherReport.HumidityInPercentage != weatherReport.HumidityInPercentage ||
				weatherReport.Time.Unix()-existingWeatherReport.Time.Unix() > 5 {
			} else {
				log.Println("Ignoring duplicate report")
				shouldIgnoreReport = true
			}
		}

		if !shouldIgnoreReport {
			db.Create(&weatherReport)
		}

	}
}

// checkForDeviceModelInfo checks whether the device model exists in the database, if it doesn't, adds it
func checkForDeviceModel(db *gorm.DB, weatherReport WeatherReport) {
	var deviceModelInfo DeviceModel
	deviceModelInfo.DeviceModel = weatherReport.DeviceModel
	deviceModelInfo.Name = weatherReport.DeviceModel

	result := db.Where("device_model = ?", weatherReport.DeviceModel).First(&deviceModelInfo)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		db.Create(&deviceModelInfo)
	}
}

func setupDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("weather.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database", err)
	}

	db.AutoMigrate(&WeatherReport{}, &DeviceModel{}, &OllamaRecommendation{})

	return db
}

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
