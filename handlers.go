package main

import (
	"net/http"
	"path"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupRouter configures the HTTP routes
func setupRouter(db *gorm.DB, hub *EventHub) *gin.Engine {
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

	// API endpoints
	r.GET("/reports/latest", func(c *gin.Context) {
		getLatestWeatherReport(c, db)
	})

	r.GET("/reports/:model", func(c *gin.Context) {
		getWeatherReportsByModel(c, db)
	})

	r.GET("/reports/events", func(c *gin.Context) {
		handleReportEvents(c, hub)
	})

	r.GET("/models", func(c *gin.Context) {
		getModels(c, db)
	})

	r.GET("/recommendations/latest", func(c *gin.Context) {
		getLatestRecommendation(c, db)
	})

	r.GET("/config", func(c *gin.Context) {
		getConfig(c)
	})

	r.POST("/config", func(c *gin.Context) {
		updateConfig(c)
	})

	return r
}

// handleReportEvents handles GET /reports/events using Server-Sent Events
func handleReportEvents(c *gin.Context, hub *EventHub) {
	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	// Register this client with the event hub
	clientID, messageCh := hub.register()

	// Ensure cleanup on request completion
	defer hub.unregister(clientID)

	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messageCh:
			if !ok {
				return
			}
			c.Writer.Write([]byte(msg))
			c.Writer.Flush()
		}
	}
}

// getLatestWeatherReport handles GET /reports/latest
func getLatestWeatherReport(c *gin.Context, db *gorm.DB) {
	var weatherReport WeatherReport
	result := db.Order("db_id desc").First(&weatherReport)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve latest weather report"})
		return
	}
	c.JSON(http.StatusOK, weatherReport)
}

// getWeatherReportsByModel handles GET /reports/:model
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

// getModels handles GET /models
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

// getLatestRecommendation handles GET /recommendations/latest
func getLatestRecommendation(c *gin.Context, db *gorm.DB) {
	var recommendation OllamaRecommendation
	result := db.Order("db_id desc").First(&recommendation)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve latest recommendation"})
		return
	}
	c.JSON(http.StatusOK, recommendation)
}

// getConfig handles GET /config
func getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, config)
}

// updateConfig handles POST /config
func updateConfig(c *gin.Context) {
	var newConfig Config
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// Validate required fields
	if newConfig.OpenAIBaseURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OpenAIBaseURL is required"})
		return
	}
	if newConfig.OpenAIModel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OpenAIModel is required"})
		return
	}
	if newConfig.IndoorDeviceModel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "IndoorDeviceModel is required"})
		return
	}
	if newConfig.OutdoorDeviceModel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OutdoorDeviceModel is required"})
		return
	}
	if newConfig.RecommendationIntervalMinutes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "RecommendationIntervalMinutes must be greater than 0"})
		return
	}

	// Update the global config
	oldInterval := config.RecommendationIntervalMinutes
	config = newConfig

	// Save to file
	if err := saveConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save configuration"})
		return
	}

	// If recommendation interval changed, restart the recommendation worker
	if oldInterval != newConfig.RecommendationIntervalMinutes {
		restartRecommendationWorker()
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated successfully", "config": config})
}
