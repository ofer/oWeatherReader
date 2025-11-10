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

	// API endpoints
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
