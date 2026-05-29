package main

import (
	"os"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "weather-test-*.db")
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

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
	})

	return db
}

func TestSaveWeatherReport_InsertsNewReport(t *testing.T) {
	db := setupTestDB(t)

	report := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "TestDevice-1",
		TemperatureInF:       72.5,
		HumidityInPercentage: 45,
	}

	saved, inserted, err := saveWeatherReport(db, report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !inserted {
		t.Fatal("expected report to be inserted")
	}
	if saved.DeviceModel != "TestDevice-1" {
		t.Fatalf("expected DeviceModel 'TestDevice-1', got '%s'", saved.DeviceModel)
	}
}

func TestSaveWeatherReport_CreatesDeviceModel(t *testing.T) {
	db := setupTestDB(t)

	report := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "NewModel-XYZ",
		TemperatureInF:       68.0,
		HumidityInPercentage: 50,
	}

	_, inserted, err := saveWeatherReport(db, report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !inserted {
		t.Fatal("expected report to be inserted")
	}

	var dm DeviceModel
	if err := db.Where("device_model = ?", "NewModel-XYZ").First(&dm).Error; err != nil {
		t.Fatal("expected device model to be created")
	}
	if dm.DeviceModel != "NewModel-XYZ" {
		t.Fatalf("expected DeviceModel 'NewModel-XYZ', got '%s'", dm.DeviceModel)
	}
}

func TestSaveWeatherReport_IgnoresDuplicate(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now().Truncate(time.Second)

	existing := WeatherReport{
		Time:                 now,
		DeviceModel:          "TestDevice-2",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}
	db.Create(&existing)

	duplicate := WeatherReport{
		Time:                 now,
		DeviceModel:          "TestDevice-2",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}

	_, inserted, err := saveWeatherReport(db, duplicate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inserted {
		t.Fatal("expected duplicate report to be ignored")
	}

	var count int64
	db.Model(&WeatherReport{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 report in database, got %d", count)
	}
}

func TestSaveWeatherReport_InsertsWhenTimeGapGreaterThan5s(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now().Truncate(time.Second)

	existing := WeatherReport{
		Time:                 now,
		DeviceModel:          "TestDevice-3",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}
	db.Create(&existing)

	newReport := WeatherReport{
		Time:                 now.Add(10 * time.Second),
		DeviceModel:          "TestDevice-3",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}

	_, inserted, err := saveWeatherReport(db, newReport)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !inserted {
		t.Fatal("expected report with time gap > 5s to be inserted")
	}

	var count int64
	db.Model(&WeatherReport{}).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 reports in database, got %d", count)
	}
}

func TestSaveWeatherReport_InsertsWhenValuesDiffer(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now().Truncate(time.Second)

	existing := WeatherReport{
		Time:                 now,
		DeviceModel:          "TestDevice-4",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}
	db.Create(&existing)

	newReport := WeatherReport{
		Time:                 now,
		DeviceModel:          "TestDevice-4",
		TemperatureInF:       71.0, // different temperature
		HumidityInPercentage: 40,
	}

	_, inserted, err := saveWeatherReport(db, newReport)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !inserted {
		t.Fatal("expected report with different values to be inserted")
	}

	var count int64
	db.Model(&WeatherReport{}).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 reports in database, got %d", count)
	}
}

func TestApplyHumidityCorrection_CorrectsHumidityWhenTempAbove70(t *testing.T) {
	db := setupTestDB(t)

	prior := WeatherReport{
		Time:                 time.Now().Add(-1 * time.Minute),
		DeviceModel:          "TestDevice-5",
		TemperatureInF:       75.0, // above 70
		HumidityInPercentage: 3,    // below 5
	}
	db.Create(&prior)

	newReport := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "TestDevice-5",
		TemperatureInF:       75.0,
		HumidityInPercentage: 99,
	}

	shouldIgnore, err := applyHumidityCorrection(db, &newReport)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shouldIgnore {
		t.Fatal("expected report to NOT be ignored")
	}
	if newReport.HumidityInPercentage != 1 {
		t.Fatalf("expected humidity corrected to 1, got %d", newReport.HumidityInPercentage)
	}
}

func TestApplyHumidityCorrection_IgnoresReportWhenTempAtOrBelow70(t *testing.T) {
	db := setupTestDB(t)

	prior := WeatherReport{
		Time:                 time.Now().Add(-1 * time.Minute),
		DeviceModel:          "TestDevice-6",
		TemperatureInF:       70.0, // exactly 70 (not above)
		HumidityInPercentage: 2,    // below 5
	}
	db.Create(&prior)

	newReport := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "TestDevice-6",
		TemperatureInF:       70.0,
		HumidityInPercentage: 99,
	}

	shouldIgnore, err := applyHumidityCorrection(db, &newReport)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !shouldIgnore {
		t.Fatal("expected report to be ignored when temp is <= 70")
	}
}

func TestApplyHumidityCorrection_DoesNothingWhenPriorHumidityAtOrAbove5(t *testing.T) {
	db := setupTestDB(t)

	prior := WeatherReport{
		Time:                 time.Now().Add(-1 * time.Minute),
		DeviceModel:          "TestDevice-7",
		TemperatureInF:       75.0,
		HumidityInPercentage: 5, // exactly 5 (not below)
	}
	db.Create(&prior)

	newReport := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "TestDevice-7",
		TemperatureInF:       75.0,
		HumidityInPercentage: 99,
	}

	shouldIgnore, err := applyHumidityCorrection(db, &newReport)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shouldIgnore {
		t.Fatal("expected report to NOT be ignored")
	}
	if newReport.HumidityInPercentage != 99 {
		t.Fatalf("expected humidity unchanged at 99, got %d", newReport.HumidityInPercentage)
	}
}

func TestApplyHumidityCorrection_DoesNothingWhenNewHumidityNot99(t *testing.T) {
	db := setupTestDB(t)

	prior := WeatherReport{
		Time:                 time.Now().Add(-1 * time.Minute),
		DeviceModel:          "TestDevice-8",
		TemperatureInF:       80.0,
		HumidityInPercentage: 3,
	}
	db.Create(&prior)

	newReport := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "TestDevice-8",
		TemperatureInF:       80.0,
		HumidityInPercentage: 50, // not 99
	}

	shouldIgnore, err := applyHumidityCorrection(db, &newReport)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shouldIgnore {
		t.Fatal("expected report to NOT be ignored")
	}
	if newReport.HumidityInPercentage != 50 {
		t.Fatalf("expected humidity unchanged at 50, got %d", newReport.HumidityInPercentage)
	}
}

func TestApplyHumidityCorrection_NoPriorReport(t *testing.T) {
	db := setupTestDB(t)

	newReport := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "TestDevice-9",
		TemperatureInF:       80.0,
		HumidityInPercentage: 99,
	}

	shouldIgnore, err := applyHumidityCorrection(db, &newReport)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shouldIgnore {
		t.Fatal("expected report to NOT be ignored when no prior report exists")
	}
	if newReport.HumidityInPercentage != 99 {
		t.Fatalf("expected humidity unchanged at 99, got %d", newReport.HumidityInPercentage)
	}
}

func TestApplyHumidityCorrection_DifferentDeviceModel(t *testing.T) {
	db := setupTestDB(t)

	prior := WeatherReport{
		Time:                 time.Now().Add(-1 * time.Minute),
		DeviceModel:          "TestDevice-10A",
		TemperatureInF:       75.0,
		HumidityInPercentage: 3,
	}
	db.Create(&prior)

	newReport := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "TestDevice-10B", // different model
		TemperatureInF:       75.0,
		HumidityInPercentage: 99,
	}

	shouldIgnore, err := applyHumidityCorrection(db, &newReport)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shouldIgnore {
		t.Fatal("expected report to NOT be ignored for different device model")
	}
	if newReport.HumidityInPercentage != 99 {
		t.Fatalf("expected humidity unchanged at 99, got %d", newReport.HumidityInPercentage)
	}
}

func TestCheckDuplicate_NoPriorReport(t *testing.T) {
	db := setupTestDB(t)

	report := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "TestDevice-11",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}

	shouldIgnore, err := checkDuplicate(db, &report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shouldIgnore {
		t.Fatal("expected report to NOT be ignored when no prior report exists")
	}
}

func TestCheckDuplicate_SameTimeModelButDifferentTemp(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now().Truncate(time.Second)

	existing := WeatherReport{
		Time:                 now,
		DeviceModel:          "TestDevice-12",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}
	db.Create(&existing)

	report := WeatherReport{
		Time:                 now,
		DeviceModel:          "TestDevice-12",
		TemperatureInF:       71.0, // different
		HumidityInPercentage: 40,
	}

	shouldIgnore, err := checkDuplicate(db, &report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shouldIgnore {
		t.Fatal("expected report with different temp to NOT be ignored")
	}
}

func TestCheckDuplicate_SameTimeModelButDifferentHumidity(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now().Truncate(time.Second)

	existing := WeatherReport{
		Time:                 now,
		DeviceModel:          "TestDevice-13",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}
	db.Create(&existing)

	report := WeatherReport{
		Time:                 now,
		DeviceModel:          "TestDevice-13",
		TemperatureInF:       70.0,
		HumidityInPercentage: 45, // different
	}

	shouldIgnore, err := checkDuplicate(db, &report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shouldIgnore {
		t.Fatal("expected report with different humidity to NOT be ignored")
	}
}

func TestCheckDuplicate_SameTimeModelSameValuesIsDuplicate(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now().Truncate(time.Second)

	existing := WeatherReport{
		Time:                 now,
		DeviceModel:          "TestDevice-14",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}
	db.Create(&existing)

	report := WeatherReport{
		Time:                 now,
		DeviceModel:          "TestDevice-14",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}

	shouldIgnore, err := checkDuplicate(db, &report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !shouldIgnore {
		t.Fatal("expected report with identical time/model/values to be a duplicate")
	}
}

func TestCheckDuplicate_TimeGapGreaterThan5sNotDuplicate(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now().Truncate(time.Second)

	existing := WeatherReport{
		Time:                 now,
		DeviceModel:          "TestDevice-15",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}
	db.Create(&existing)

	report := WeatherReport{
		Time:                 now.Add(6 * time.Second),
		DeviceModel:          "TestDevice-15",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}

	shouldIgnore, err := checkDuplicate(db, &report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shouldIgnore {
		t.Fatal("expected report with 6s time gap to NOT be a duplicate")
	}
}

func TestCheckDuplicate_TimeGapExactly5sNotDuplicate(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now().Truncate(time.Second)

	existing := WeatherReport{
		Time:                 now,
		DeviceModel:          "TestDevice-16",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}
	db.Create(&existing)

	report := WeatherReport{
		Time:                 now.Add(5 * time.Second),
		DeviceModel:          "TestDevice-16",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}

	shouldIgnore, err := checkDuplicate(db, &report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shouldIgnore {
		t.Fatal("expected report with 5s time gap to NOT be a duplicate (WHERE clause requires exact time match)")
	}
}

func TestSaveWeatherReport_WithHumidityCorrectionThenInsert(t *testing.T) {
	db := setupTestDB(t)

	prior := WeatherReport{
		Time:                 time.Now().Add(-1 * time.Minute),
		DeviceModel:          "TestDevice-17",
		TemperatureInF:       80.0,
		HumidityInPercentage: 2,
	}
	db.Create(&prior)

	report := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "TestDevice-17",
		TemperatureInF:       80.0,
		HumidityInPercentage: 99,
	}

	saved, inserted, err := saveWeatherReport(db, report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !inserted {
		t.Fatal("expected report to be inserted after humidity correction")
	}
	if saved.HumidityInPercentage != 1 {
		t.Fatalf("expected corrected humidity 1, got %d", saved.HumidityInPercentage)
	}

	var count int64
	db.Model(&WeatherReport{}).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 reports in database, got %d", count)
	}
}

func TestSaveWeatherReport_WithHumidityCorrectionThenIgnored(t *testing.T) {
	db := setupTestDB(t)

	prior := WeatherReport{
		Time:                 time.Now().Add(-1 * time.Minute),
		DeviceModel:          "TestDevice-18",
		TemperatureInF:       65.0,
		HumidityInPercentage: 3,
	}
	db.Create(&prior)

	report := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "TestDevice-18",
		TemperatureInF:       65.0,
		HumidityInPercentage: 99,
	}

	_, inserted, err := saveWeatherReport(db, report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inserted {
		t.Fatal("expected report to be ignored when temp <= 70")
	}

	var count int64
	db.Model(&WeatherReport{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 report in database, got %d", count)
	}
}

func TestSaveWeatherReport_DuplicateAfterHumidityCorrectionIgnored(t *testing.T) {
	db := setupTestDB(t)

	prior := WeatherReport{
		Time:                 time.Now().Add(-1 * time.Minute),
		DeviceModel:          "TestDevice-19",
		TemperatureInF:       80.0,
		HumidityInPercentage: 2,
	}
	db.Create(&prior)

	report := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "TestDevice-19",
		TemperatureInF:       80.0,
		HumidityInPercentage: 99,
	}

	// First save: humidity corrected to 1, inserted
	_, inserted, err := saveWeatherReport(db, report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !inserted {
		t.Fatal("expected first report to be inserted")
	}

	// Second save with same corrected values: duplicate
	duplicate := WeatherReport{
		Time:                 report.Time,
		DeviceModel:          "TestDevice-19",
		TemperatureInF:       80.0,
		HumidityInPercentage: 1, // same as corrected value
	}

	_, inserted, err = saveWeatherReport(db, duplicate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inserted {
		t.Fatal("expected duplicate after correction to be ignored")
	}

	var count int64
	db.Model(&WeatherReport{}).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 reports in database, got %d", count)
	}
}

func TestSaveWeatherReport_WithHumidityCorrectionThenDuplicateIgnored(t *testing.T) {
	db := setupTestDB(t)

	prior := WeatherReport{
		Time:                 time.Now().Add(-1 * time.Minute),
		DeviceModel:          "TestDevice-20",
		TemperatureInF:       80.0,
		HumidityInPercentage: 2,
	}
	db.Create(&prior)

	report := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "TestDevice-20",
		TemperatureInF:       80.0,
		HumidityInPercentage: 99,
	}

	// First save: humidity corrected to 1, inserted
	_, inserted, err := saveWeatherReport(db, report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !inserted {
		t.Fatal("expected first report to be inserted")
	}

	// Second save: same time, same corrected values -> duplicate
	duplicate := WeatherReport{
		Time:                 report.Time,
		DeviceModel:          "TestDevice-20",
		TemperatureInF:       80.0,
		HumidityInPercentage: 1, // same as corrected value
	}

	_, inserted, err = saveWeatherReport(db, duplicate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inserted {
		t.Fatal("expected duplicate after correction to be ignored")
	}
}

func TestSaveWeatherReport_MultipleDifferentDevices(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now()

	reports := []WeatherReport{
		{Time: now, DeviceModel: "Sensor-A", TemperatureInF: 70.0, HumidityInPercentage: 40},
		{Time: now, DeviceModel: "Sensor-B", TemperatureInF: 72.0, HumidityInPercentage: 45},
		{Time: now, DeviceModel: "Sensor-A", TemperatureInF: 71.0, HumidityInPercentage: 42},
	}

	for _, r := range reports {
		_, inserted, err := saveWeatherReport(db, r)
		if err != nil {
			t.Fatalf("expected no error for %s, got %v", r.DeviceModel, err)
		}
		if !inserted {
			t.Fatalf("expected report for %s to be inserted", r.DeviceModel)
		}
	}

	var count int64
	db.Model(&WeatherReport{}).Count(&count)
	if count != 3 {
		t.Fatalf("expected 3 reports, got %d", count)
	}

	var dmCount int64
	db.Model(&DeviceModel{}).Count(&dmCount)
	if dmCount != 2 {
		t.Fatalf("expected 2 device models, got %d", dmCount)
	}
}
