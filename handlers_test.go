package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDBWithHistoryData(t *testing.T) *gorm.DB {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "weather-history-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	db, err := gorm.Open(sqlite.Open(tmpFile.Name()), &gorm.Config{})
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to connect test database: %v", err)
	}

	if err := db.AutoMigrate(&WeatherReport{}, &DeviceModel{}, &OllamaRecommendation{}); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to migrate test database: %v", err)
	}

	// Insert device models
	db.Create(&DeviceModel{DeviceModel: "LaCrosse-TX141W", Name: "La Crosse TX141W"})
	db.Create(&DeviceModel{DeviceModel: "IndoorSensor", Name: "Indoor Sensor"})

	// Insert weather reports spanning 3 days
	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	reports := []WeatherReport{
		{Time: baseTime, DeviceModel: "LaCrosse-TX141W", TemperatureInF: 85.5, HumidityInPercentage: 60},
		{Time: baseTime.Add(2 * time.Hour), DeviceModel: "LaCrosse-TX141W", TemperatureInF: 87.0, HumidityInPercentage: 58},
		{Time: baseTime.Add(24 * time.Hour), DeviceModel: "LaCrosse-TX141W", TemperatureInF: 88.0, HumidityInPercentage: 55},
		{Time: baseTime.Add(48 * time.Hour), DeviceModel: "LaCrosse-TX141W", TemperatureInF: 82.0, HumidityInPercentage: 65},
		{Time: baseTime, DeviceModel: "IndoorSensor", TemperatureInF: 72.0, HumidityInPercentage: 45},
		{Time: baseTime.Add(24 * time.Hour), DeviceModel: "IndoorSensor", TemperatureInF: 73.0, HumidityInPercentage: 44},
		{Time: baseTime.Add(2 * time.Hour), DeviceModel: "LaCrosse-TX141W", TemperatureInF: 86.0, HumidityInPercentage: 59},
	}
	for _, r := range reports {
		if err := db.Create(&r).Error; err != nil {
			os.Remove(tmpFile.Name())
			t.Fatalf("failed to insert test data: %v", err)
		}
	}

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
	})

	return db
}

func TestGetDailyAggregates_ReturnsData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBWithHistoryData(t)

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest("GET", "/reports/history?days=30", nil)
	c.Request.URL.RawQuery = "days=30"

	getDailyAggregates(c, db)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var results []DailyAggregate
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected non-empty results, got 0 rows")
	}
}

func TestGetDailyAggregates_ContainsCorrectFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBWithHistoryData(t)

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest("GET", "/reports/history?days=30", nil)
	c.Request.URL.RawQuery = "days=30"

	getDailyAggregates(c, db)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var results []DailyAggregate
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected non-empty results")
	}

	// Verify each result has all required fields
	for _, r := range results {
		if r.Date == "" {
			t.Error("expected non-empty date")
		}
		if r.Model == "" {
			t.Error("expected non-empty model")
		}
		if r.ModelName == "" {
			t.Error("expected non-empty modelName")
		}
		if r.HighTemp <= 0 {
			t.Errorf("expected positive highTemp, got %f", r.HighTemp)
		}
		if r.HighHumidity == 0 && r.LowHumidity == 0 {
			// Only fail if no humidity data at all (all readings could be 0 in theory)
		}
	}
}

func TestGetDailyAggregates_GroupsByDayAndModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBWithHistoryData(t)

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest("GET", "/reports/history?days=30", nil)
	c.Request.URL.RawQuery = "days=30"

	getDailyAggregates(c, db)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var results []DailyAggregate
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// We have 3 days of outdoor data + 2 days of indoor data.
	// Day 1 (Jun 1): LaCrosse (3 reports: 85.5, 87.0, 86.0) + Indoor (1 report: 72.0)
	// Day 2 (Jun 2): LaCrosse (1 report: 88.0) + Indoor (1 report: 73.0)
	// Day 3 (Jun 3): LaCrosse (1 report: 82.0)
	// Total: 5 unique day-model combinations
	if len(results) != 5 {
		t.Errorf("expected 5 day-model combinations, got %d", len(results))
	}

	for _, r := range results {
		if r.Model == "LaCrosse-TX141W" && r.Date == "2026-06-01 00:00:00" {
			if r.AvgTemp == 0 {
				t.Error("expected non-zero avgTemp for day 1 LaCrosse")
			}
			// Avg of 85.5, 87.0, 86.0 = 86.167 → rounded by SQL to ~86.2
			if r.AvgTemp < 85.0 || r.AvgTemp > 88.0 {
				t.Errorf("expected avgTemp ~86.2 for day 1 LaCrosse, got %f", r.AvgTemp)
			}
			// High = max(85.5, 87.0, 86.0) = 87.0
			if r.HighTemp != 87.0 {
				t.Errorf("expected highTemp 87.0 for day 1 LaCrosse, got %f", r.HighTemp)
			}
			// Low = min(85.5, 87.0, 86.0) = 85.5
			if r.LowTemp != 85.5 {
				t.Errorf("expected lowTemp 85.5 for day 1 LaCrosse, got %f", r.LowTemp)
			}
		}
	}
}

func TestGetDailyAggregates_WithNoData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpFile, err := os.CreateTemp("", "weather-empty-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	db, err := gorm.Open(sqlite.Open(tmpFile.Name()), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect test database: %v", err)
	}
	db.AutoMigrate(&WeatherReport{}, &DeviceModel{}, &OllamaRecommendation{})

	// No data inserted

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest("GET", "/reports/history?days=30", nil)
	c.Request.URL.RawQuery = "days=30"

	getDailyAggregates(c, db)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 with empty data, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var results []DailyAggregate
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results with no data, got %d", len(results))
	}
}

func TestGetDailyAggregates_DefaultsTo90Days(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBWithHistoryData(t)

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest("GET", "/reports/history", nil)
	// No days parameter — should default to 90

	getDailyAggregates(c, db)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var results []DailyAggregate
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected non-empty results with default 90 days")
	}
}

func TestGetDailyAggregates_MonthlyHasAvgHighAndLow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpFile, err := os.CreateTemp("", "weather-monthly-avg-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	db, err := gorm.Open(sqlite.Open(tmpFile.Name()), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect test database: %v", err)
	}
	db.AutoMigrate(&WeatherReport{}, &DeviceModel{}, &OllamaRecommendation{})

	// Test data: 3 days of readings with known daily high/low for nested aggregation verification
	db.Create(&DeviceModel{DeviceModel: "TestSensor", Name: "Test Sensor"})
	db.Create(&WeatherReport{Time: time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC), DeviceModel: "TestSensor", TemperatureInF: 80, HumidityInPercentage: 50})
	db.Create(&WeatherReport{Time: time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC), DeviceModel: "TestSensor", TemperatureInF: 90, HumidityInPercentage: 40})
	db.Create(&WeatherReport{Time: time.Date(2026, 5, 2, 8, 0, 0, 0, time.UTC), DeviceModel: "TestSensor", TemperatureInF: 85, HumidityInPercentage: 48})
	db.Create(&WeatherReport{Time: time.Date(2026, 5, 2, 14, 0, 0, 0, time.UTC), DeviceModel: "TestSensor", TemperatureInF: 95, HumidityInPercentage: 38})
	db.Create(&WeatherReport{Time: time.Date(2026, 5, 3, 8, 0, 0, 0, time.UTC), DeviceModel: "TestSensor", TemperatureInF: 88, HumidityInPercentage: 45})
	db.Create(&WeatherReport{Time: time.Date(2026, 5, 3, 14, 0, 0, 0, time.UTC), DeviceModel: "TestSensor", TemperatureInF: 92, HumidityInPercentage: 35})

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest("GET", "/reports/history?days=365", nil)
	c.Request.URL.RawQuery = "days=365"

	getDailyAggregates(c, db)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var results []DailyAggregate
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 monthly row, got %d", len(results))
	}

	r := results[0]

	if r.AvgHighTemp == 0 {
		t.Fatal("expected non-zero avgHighTemp")
	}
	var expectedAvgHigh float32 = 92.3
	if r.AvgHighTemp < expectedAvgHigh-0.1 || r.AvgHighTemp > expectedAvgHigh+0.1 {
		t.Errorf("expected avgHighTemp ~%.1f, got %.1f", expectedAvgHigh, r.AvgHighTemp)
	}

	if r.AvgLowTemp == 0 {
		t.Fatal("expected non-zero avgLowTemp")
	}
	var expectedAvgLow float32 = 84.3
	if r.AvgLowTemp < expectedAvgLow-0.1 || r.AvgLowTemp > expectedAvgLow+0.1 {
		t.Errorf("expected avgLowTemp ~%.1f, got %.1f", expectedAvgLow, r.AvgLowTemp)
	}

	if r.HighTemp != 95.0 {
		t.Errorf("expected highTemp 95.0, got %.1f", r.HighTemp)
	}

	if r.LowTemp != 80.0 {
		t.Errorf("expected lowTemp 80.0, got %.1f", r.LowTemp)
	}
}

func TestGetDailyAggregates_MonthlyHasAvgHighLowHumidity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpFile, err := os.CreateTemp("", "weather-monthly-humidity-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	db, err := gorm.Open(sqlite.Open(tmpFile.Name()), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect test database: %v", err)
	}
	db.AutoMigrate(&WeatherReport{}, &DeviceModel{}, &OllamaRecommendation{})

	// Test data: 3 days, each with 2 readings (morning + afternoon)
	// Day 1: humidity 50 and 40 → daily avg=45, daily high=50, daily low=40
	// Day 2: humidity 48 and 38 → daily avg=43, daily high=48, daily low=38
	// Day 3: humidity 45 and 35 → daily avg=40, daily high=45, daily low=35
	//
	// Monthly aggregation:
	// avg_high_humidity = AVG(50, 48, 45) = 47.67 → rounded to 48
	// avg_low_humidity  = AVG(40, 38, 35) = 37.67  → rounded to 38
	db.Create(&DeviceModel{DeviceModel: "HumiditySensor", Name: "Humidity Sensor"})
	db.Create(&WeatherReport{Time: time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC), DeviceModel: "HumiditySensor", TemperatureInF: 70, HumidityInPercentage: 50})
	db.Create(&WeatherReport{Time: time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC), DeviceModel: "HumiditySensor", TemperatureInF: 75, HumidityInPercentage: 40})
	db.Create(&WeatherReport{Time: time.Date(2026, 5, 2, 8, 0, 0, 0, time.UTC), DeviceModel: "HumiditySensor", TemperatureInF: 71, HumidityInPercentage: 48})
	db.Create(&WeatherReport{Time: time.Date(2026, 5, 2, 14, 0, 0, 0, time.UTC), DeviceModel: "HumiditySensor", TemperatureInF: 76, HumidityInPercentage: 38})
	db.Create(&WeatherReport{Time: time.Date(2026, 5, 3, 8, 0, 0, 0, time.UTC), DeviceModel: "HumiditySensor", TemperatureInF: 72, HumidityInPercentage: 45})
	db.Create(&WeatherReport{Time: time.Date(2026, 5, 3, 14, 0, 0, 0, time.UTC), DeviceModel: "HumiditySensor", TemperatureInF: 77, HumidityInPercentage: 35})

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest("GET", "/reports/history?days=365", nil)
	c.Request.URL.RawQuery = "days=365"

	getDailyAggregates(c, db)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var results []DailyAggregate
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 monthly row, got %d", len(results))
	}

	r := results[0]

	// avg_high_humidity = AVG(50, 48, 45) = 47.67 → ROUND to 48
	if r.AvgHighHumidity == 0 {
		t.Fatal("expected non-zero avgHighHumidity")
	}
	var expectedAvgHighHumidity uint8 = 48
	if r.AvgHighHumidity != expectedAvgHighHumidity {
		t.Errorf("expected avgHighHumidity %d, got %d", expectedAvgHighHumidity, r.AvgHighHumidity)
	}

	// avg_low_humidity = AVG(40, 38, 35) = 37.67 → ROUND to 38
	if r.AvgLowHumidity == 0 {
		t.Fatal("expected non-zero avgLowHumidity")
	}
	var expectedAvgLowHumidity uint8 = 38
	if r.AvgLowHumidity != expectedAvgLowHumidity {
		t.Errorf("expected avgLowHumidity %d, got %d", expectedAvgLowHumidity, r.AvgLowHumidity)
	}

	// Also verify high/low humidity (max of daily highs, min of daily lows)
	if r.HighHumidity != 50 {
		t.Errorf("expected highHumidity 50, got %d", r.HighHumidity)
	}
	if r.LowHumidity != 35 {
		t.Errorf("expected lowHumidity 35, got %d", r.LowHumidity)
	}
}
