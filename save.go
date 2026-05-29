package main

import (
	"errors"
	"log"

	"gorm.io/gorm"
)

// saveWeatherReport handles duplicate detection, humidity correction, and database insertion.
// Returns the saved report, whether it was inserted (true) or ignored (false), and any error.
// The caller is responsible for broadcasting events after successful inserts.
func saveWeatherReport(db *gorm.DB, report WeatherReport) (WeatherReport, bool, error) {
	// Ensure device model exists in database
	checkForDeviceModel(db, report)

	// Apply humidity correction if needed
	shouldIgnoreReport, err := applyHumidityCorrection(db, &report)
	if err != nil {
		return report, false, err
	}
	if shouldIgnoreReport {
		return report, false, nil
	}

	// Check for duplicates
	shouldIgnoreReport, err = checkDuplicate(db, &report)
	if err != nil {
		return report, false, err
	}
	if shouldIgnoreReport {
		return report, false, nil
	}

	// Insert into database
	result := db.Create(&report)
	if result.Error != nil {
		return report, false, result.Error
	}

	return report, true, nil
}

// applyHumidityCorrection checks the last report and corrects or ignores
// reports with anomalous humidity values (99 following humidity < 5).
func applyHumidityCorrection(db *gorm.DB, report *WeatherReport) (bool, error) {
	var lastReport WeatherReport
	result := db.Where("device_model = ?", report.DeviceModel).Order("db_id desc").First(&lastReport)
	if result.Error != nil {
		// No prior report exists, nothing to correct
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		// Other database errors are logged but not fatal
		log.Println("Failed to retrieve last weather report:", result.Error)
		return false, nil
	}

	// Humidity correction: if prior humidity < 5 and new humidity is 99
	if lastReport.HumidityInPercentage < 5 && report.HumidityInPercentage == 99 {
		log.Println("deciding on proper humidity due to erroneous humidity report")
		if lastReport.TemperatureInF > 70 {
			log.Println("temp is > 70, setting humidity to 1")
			report.HumidityInPercentage = 1
		} else {
			log.Println("temp is <= 70, ignoring report")
			return true, nil
		}
	}

	return false, nil
}

// checkDuplicate determines whether this report is a duplicate of an existing one.
// Returns true if the report should be ignored.
func checkDuplicate(db *gorm.DB, report *WeatherReport) (bool, error) {
	var existing WeatherReport
	result := db.Where("time = ? AND device_model = ?", report.Time, report.DeviceModel).First(&existing)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// No existing report with same time and model, not a duplicate
		return false, nil
	}

	// Found a report with same time and model — check if values differ or time gap > 5s
	if existing.TemperatureInF != report.TemperatureInF ||
		existing.HumidityInPercentage != report.HumidityInPercentage ||
		report.Time.Unix()-existing.Time.Unix() > 5 {
		// Values differ or significant time gap — treat as new report
		return false, nil
	}

	log.Println("Ignoring duplicate report")
	return true, nil
}
