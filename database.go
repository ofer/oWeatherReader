package main

import (
	"errors"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDatabase initializes the database connection and runs migrations
func setupDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("weather.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database", err)
	}

	db.AutoMigrate(&WeatherReport{}, &DeviceModel{}, &OllamaRecommendation{})

	return db
}

// checkForDeviceModel checks whether the device model exists in the database, if it doesn't, adds it
func checkForDeviceModel(db *gorm.DB, weatherReport WeatherReport) {
	var deviceModelInfo DeviceModel
	deviceModelInfo.DeviceModel = weatherReport.DeviceModel
	deviceModelInfo.Name = weatherReport.DeviceModel

	result := db.Where("device_model = ?", weatherReport.DeviceModel).First(&deviceModelInfo)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		db.Create(&deviceModelInfo)
	}
}
