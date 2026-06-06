package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"
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

	r.GET("/reports/history", func(c *gin.Context) {
		getDailyAggregates(c, db)
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

	type ClassifiedDeviceModel struct {
		DeviceModel string `json:"DeviceModel"`
		ReportCount uint8  `json:"ReportCount"`
		Name        string `json:"Name"`
		IsIndoor    bool   `json:"isIndoor"`
		IsOutdoor   bool   `json:"isOutdoor"`
	}

	classified := make([]ClassifiedDeviceModel, len(deviceModels))
	for i, dm := range deviceModels {
		classified[i] = ClassifiedDeviceModel{
			DeviceModel: dm.DeviceModel,
			ReportCount: uint8(dm.ReportCount),
			Name:        dm.Name,
			IsIndoor:    dm.DeviceModel == config.IndoorDeviceModel,
			IsOutdoor:   dm.DeviceModel == config.OutdoorDeviceModel,
		}
	}

	c.JSON(http.StatusOK, classified)
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

type DailyAggregate struct {
	Date              string  `json:"date"`
	Model             string  `json:"model"`
	AvgTemp           float32 `json:"avgTemp"`
	HighTemp          float32 `json:"highTemp"`
	LowTemp           float32 `json:"lowTemp"`
	AvgHighTemp       float32 `json:"avgHighTemp"`
	AvgLowTemp        float32 `json:"avgLowTemp"`
	AvgHumidity       float32 `json:"avgHumidity"`
	AvgHighHumidity   uint8   `json:"avgHighHumidity"`
	AvgLowHumidity    uint8   `json:"avgLowHumidity"`
	HighHumidity      uint8   `json:"highHumidity"`
	LowHumidity       uint8   `json:"lowHumidity"`
	ModelName         string  `json:"modelName"`
}

func getDailyAggregates(c *gin.Context, db *gorm.DB) {
	daysStr := c.DefaultQuery("days", "90")
	var days int
	if _, err := fmt.Sscanf(daysStr, "%d", &days); err != nil || days <= 0 || days > 365 {
		days = 90
	}
	since := time.Now().AddDate(0, 0, -days)

	// Optional: filter by specific device models (comma-separated)
	modelsParam := c.Query("models")
	var models []string
	if modelsParam != "" {
		for _, m := range strings.Split(modelsParam, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				models = append(models, m)
			}
		}
	}

	// Monthly aggregation when showing a full year
	useMonthly := days > 90

	var results []DailyAggregate
	if useMonthly {
		var monthlyResults []struct {
			Month            string          `gorm:"column:month"`
			Model            sql.NullString  `gorm:"column:model"`
			ModelName        sql.NullString  `gorm:"column:model_name"`
			AvgTemp          sql.NullFloat64 `gorm:"column:avg_temp"`
			HighTemp         sql.NullFloat64 `gorm:"column:high_temp"`
			LowTemp          sql.NullFloat64 `gorm:"column:low_temp"`
			AvgHighTemp      sql.NullFloat64 `gorm:"column:avg_high_temp"`
			AvgLowTemp       sql.NullFloat64 `gorm:"column:avg_low_temp"`
			AvgHumidity      sql.NullFloat64 `gorm:"column:avg_humidity"`
			AvgHighHumidity  sql.NullFloat64 `gorm:"column:avg_high_humidity"`
			AvgLowHumidity   sql.NullFloat64 `gorm:"column:avg_low_humidity"`
			HighHumidity     sql.NullFloat64 `gorm:"column:high_humidity"`
			LowHumidity      sql.NullFloat64 `gorm:"column:low_humidity"`
		}

		whereClause := "weather_reports.time > ?"
		args := []interface{}{since}
		if len(models) > 0 {
			placeholders := make([]string, len(models))
			for i, m := range models {
				placeholders[i] = "?"
				args = append(args, m)
			}
			whereClause += " AND weather_reports.device_model IN (" + strings.Join(placeholders, ",") + ")"
		}

		monthlyQuery := fmt.Sprintf(`
			SELECT
				t.month,
				t.model,
				t.model_name,
				ROUND(AVG(t.avg_temp), 1) as avg_temp,
				MAX(t.max_temp) as high_temp,
				MIN(t.min_temp) as low_temp,
				ROUND(AVG(t.max_temp), 1) as avg_high_temp,
				ROUND(AVG(t.min_temp), 1) as avg_low_temp,
				ROUND(AVG(t.avg_humidity), 0) as avg_humidity,
				ROUND(AVG(t.max_humidity), 0) as avg_high_humidity,
				ROUND(AVG(t.min_humidity), 0) as avg_low_humidity,
				CAST(MAX(t.max_humidity) AS REAL) as high_humidity,
				CAST(MIN(t.min_humidity) AS REAL) as low_humidity
			FROM (
				SELECT
					strftime('%%Y-%%m', weather_reports.time) as month,
					weather_reports.device_model as model,
					device_models.name as model_name,
					ROUND(AVG(weather_reports.temperature_in_f), 1) as avg_temp,
					ROUND(MAX(weather_reports.temperature_in_f), 1) as max_temp,
					ROUND(MIN(weather_reports.temperature_in_f), 1) as min_temp,
					ROUND(AVG(weather_reports.humidity_in_percentage), 0) as avg_humidity,
					MAX(weather_reports.humidity_in_percentage) as max_humidity,
					MIN(weather_reports.humidity_in_percentage) as min_humidity
				FROM weather_reports
				JOIN device_models ON device_models.device_model = weather_reports.device_model
				WHERE %s
				GROUP BY strftime('%%Y-%%d', weather_reports.time), weather_reports.device_model
			) t
			GROUP BY t.month, t.model
			ORDER BY t.month ASC, t.model ASC
		`, whereClause)

		result := db.Raw(monthlyQuery, args...).Scan(&monthlyResults)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve monthly aggregates"})
			return
		}

		results = make([]DailyAggregate, 0, len(monthlyResults))
		for _, r := range monthlyResults {
			agg := DailyAggregate{
				Date:      r.Month,
				Model:     func() string { if r.Model.Valid { return r.Model.String } else { return "unknown" } }(),
				ModelName: func() string { if r.ModelName.Valid { return r.ModelName.String } else { return r.Model.String } }(),
			}
			if r.AvgTemp.Valid {
				agg.AvgTemp = float32(r.AvgTemp.Float64)
			}
			if r.HighTemp.Valid {
				agg.HighTemp = float32(r.HighTemp.Float64)
			}
			if r.LowTemp.Valid {
				agg.LowTemp = float32(r.LowTemp.Float64)
			}
			if r.AvgHighTemp.Valid {
				agg.AvgHighTemp = float32(r.AvgHighTemp.Float64)
			}
			if r.AvgLowTemp.Valid {
				agg.AvgLowTemp = float32(r.AvgLowTemp.Float64)
			}
			if r.AvgHumidity.Valid {
				agg.AvgHumidity = float32(r.AvgHumidity.Float64)
			}
			if r.AvgHighHumidity.Valid {
				agg.AvgHighHumidity = uint8(r.AvgHighHumidity.Float64)
			}
			if r.AvgLowHumidity.Valid {
				agg.AvgLowHumidity = uint8(r.AvgLowHumidity.Float64)
			}
			if r.HighHumidity.Valid {
				agg.HighHumidity = uint8(r.HighHumidity.Float64)
			}
			if r.LowHumidity.Valid {
				agg.LowHumidity = uint8(r.LowHumidity.Float64)
			}
			results = append(results, agg)
		}
	} else {
		type rawDaily struct {
			Date        sql.NullString `gorm:"column:date"`
			Model       sql.NullString `gorm:"column:model"`
			AvgTemp     sql.NullFloat64 `gorm:"column:avg_temp"`
			HighTemp    sql.NullFloat64 `gorm:"column:high_temp"`
			LowTemp     sql.NullFloat64 `gorm:"column:low_temp"`
			AvgHumidity sql.NullFloat64 `gorm:"column:avg_humidity"`
			HighHumidity sql.NullInt64  `gorm:"column:high_humidity"`
			LowHumidity  sql.NullInt64  `gorm:"column:low_humidity"`
			ModelName   sql.NullString `gorm:"column:model_name"`
		}

		var rawResults []rawDaily

		whereClause := "weather_reports.time > ?"
		args := []interface{}{since}
		if len(models) > 0 {
			placeholders := make([]string, len(models))
			for i, m := range models {
				placeholders[i] = "?"
				args = append(args, m)
			}
			whereClause += " AND weather_reports.device_model IN (" + strings.Join(placeholders, ",") + ")"
		}

		dailyQuery := fmt.Sprintf(`
			SELECT
				DATETIME(weather_reports.time, 'start of day') as date,
				weather_reports.device_model as model,
				ROUND(AVG(weather_reports.temperature_in_f), 1) as avg_temp,
				ROUND(MAX(weather_reports.temperature_in_f), 1) as high_temp,
				ROUND(MIN(weather_reports.temperature_in_f), 1) as low_temp,
				ROUND(AVG(weather_reports.humidity_in_percentage), 0) as avg_humidity,
				CAST(MAX(weather_reports.humidity_in_percentage) AS INTEGER) as high_humidity,
				CAST(MIN(weather_reports.humidity_in_percentage) AS INTEGER) as low_humidity,
				device_models.name as model_name
			FROM weather_reports
			JOIN device_models ON device_models.device_model = weather_reports.device_model
			WHERE %s
			GROUP BY date, weather_reports.device_model
			ORDER BY date ASC, weather_reports.device_model ASC
		`, whereClause)

		result := db.Raw(dailyQuery, args...).Scan(&rawResults)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve daily aggregates"})
			return
		}

		// Compute 5-year daily averages (avg high/low per model per month-day) for historical comparison.
		var fiveYearAvgResults []struct {
			Model       string          `gorm:"column:model"`
			MonthDay    string          `gorm:"column:month_day"`
			AvgHighTemp sql.NullFloat64 `gorm:"column:avg_high_temp"`
			AvgLowTemp  sql.NullFloat64 `gorm:"column:avg_low_temp"`
		}
		fiveYearQuery := `
			WITH daily AS (
				SELECT
					weather_reports.device_model as model,
					strftime('%m-%d', time) as month_day,
					ROUND(MAX(temperature_in_f), 1) as daily_high,
					ROUND(MIN(temperature_in_f), 1) as daily_low
				FROM weather_reports
				WHERE date(time) >= (SELECT strftime('%Y-%m-%d', MAX(time), '-5 years', 'start of day') FROM weather_reports)
				AND date(time) < (SELECT strftime('%Y-%m-%d', MAX(time), 'start of day') FROM weather_reports)
				GROUP BY date(time), weather_reports.device_model, strftime('%m-%d', time)
			)
			SELECT
				model,
				month_day,
				ROUND(AVG(daily_high), 1) as avg_high_temp,
				ROUND(AVG(daily_low), 1) as avg_low_temp
			FROM daily
			GROUP BY model, month_day
		`
		db.Raw(fiveYearQuery).Scan(&fiveYearAvgResults)

		// Map: "MM-DD" -> {high, low} per model
		type modelAvg struct {
			high float32
			low  float32
		}
		fiveYearMap := make(map[string]map[string]modelAvg)
		for _, r := range fiveYearAvgResults {
			if !r.AvgHighTemp.Valid || !r.AvgLowTemp.Valid {
				continue
			}
			if fiveYearMap[r.MonthDay] == nil {
				fiveYearMap[r.MonthDay] = make(map[string]modelAvg)
			}
			fiveYearMap[r.MonthDay][r.Model] = modelAvg{
				high: float32(r.AvgHighTemp.Float64),
				low:  float32(r.AvgLowTemp.Float64),
			}
		}

		type modelAgg struct {
			Date           string
			Model          string
			ModelName      string
			HighTemp       float32
			LowTemp        float32
			HighHumidity   uint8
			LowHumidity    uint8
			AvgHighTemp    float32
			AvgLowTemp     float32
			sumAvgTemp     float32
			countAvgTemp   int
			sumAvgHum      float32
			countAvgHum    int
		}

		aggMap := make(map[string]*modelAgg)
		for _, r := range rawResults {
			if !r.Date.Valid || !r.Model.Valid {
				continue
			}
			key := r.Date.String + "|" + r.Model.String
			agg, exists := aggMap[key]
			if !exists {
				agg = &modelAgg{
					Date:     r.Date.String,
					Model:    r.Model.String,
					ModelName: func() string { if r.ModelName.Valid { return r.ModelName.String } else { return r.Model.String } }(),
				}
				if r.HighTemp.Valid {
					agg.HighTemp = float32(r.HighTemp.Float64)
				}
				if r.LowTemp.Valid {
					agg.LowTemp = float32(r.LowTemp.Float64)
				}
				if r.HighHumidity.Valid {
					agg.HighHumidity = uint8(r.HighHumidity.Int64)
				}
				if r.LowHumidity.Valid {
					agg.LowHumidity = uint8(r.LowHumidity.Int64)
				}
				aggMap[key] = agg
			} else {
				if r.HighTemp.Valid && float32(r.HighTemp.Float64) > agg.HighTemp {
					agg.HighTemp = float32(r.HighTemp.Float64)
				}
				if r.LowTemp.Valid && float32(r.LowTemp.Float64) < agg.LowTemp {
					agg.LowTemp = float32(r.LowTemp.Float64)
				}
				if r.HighHumidity.Valid && uint8(r.HighHumidity.Int64) > agg.HighHumidity {
					agg.HighHumidity = uint8(r.HighHumidity.Int64)
				}
				if r.LowHumidity.Valid && uint8(r.LowHumidity.Int64) < agg.LowHumidity {
					agg.LowHumidity = uint8(r.LowHumidity.Int64)
				}
			}
			if r.AvgTemp.Valid {
				agg.sumAvgTemp += float32(r.AvgTemp.Float64)
				agg.countAvgTemp++
			}
			if r.AvgHumidity.Valid {
				agg.sumAvgHum += float32(r.AvgHumidity.Float64)
				agg.countAvgHum++
			}
		}

		// Look up 5-year historical averages by month-day and model
		for _, agg := range aggMap {
			// Extract MM-DD from either "YYYY-MM-DD" or "YYYY-MM-DD HH:MM:SS"
			var mm, dd string
			if len(agg.Date) >= 10 {
				mm = agg.Date[5:7]
				dd = agg.Date[8:10]
			}
			monthDay := mm + "-" + dd
			if byModel, ok := fiveYearMap[monthDay]; ok {
				if avg, ok := byModel[agg.Model]; ok {
					agg.AvgHighTemp = avg.high
					agg.AvgLowTemp = avg.low
				}
			}
		}

		results = make([]DailyAggregate, 0, len(aggMap))
		for _, agg := range aggMap {
			avgTemp := float32(0)
			if agg.countAvgTemp > 0 {
				avgTemp = agg.sumAvgTemp / float32(agg.countAvgTemp)
			}
			avgHumidity := float32(0)
			if agg.countAvgHum > 0 {
				avgHumidity = agg.sumAvgHum / float32(agg.countAvgHum)
			}
			results = append(results, DailyAggregate{
				Date:           agg.Date,
				Model:          agg.Model,
				ModelName:      agg.ModelName,
				AvgTemp:        avgTemp,
				HighTemp:       agg.HighTemp,
				LowTemp:        agg.LowTemp,
				AvgHumidity:    avgHumidity,
				HighHumidity:   agg.HighHumidity,
				LowHumidity:    agg.LowHumidity,
				AvgHighTemp:    agg.AvgHighTemp,
				AvgLowTemp:     agg.AvgLowTemp,
			})
		}
	}

	c.JSON(http.StatusOK, results)
}
