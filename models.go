package main

import (
	"time"
)

// WeatherReport represents a weather measurement from a device
type WeatherReport struct {
	DbId                 uint      `gorm:"primaryKey;autoIncrement"`
	Time                 time.Time `gorm:"index"`
	DeviceModel          string    `gorm:"index"`
	TemperatureInF       float32
	HumidityInPercentage uint8
}

// DeviceModel represents a device model in the system
type DeviceModel struct {
	DbId        uint `gorm:"primaryKey;autoIncrement"`
	DeviceModel string
	Name        string
}

// DeviceModelCount represents a device model with its report count for API responses
type DeviceModelCount struct {
	DeviceModel string
	ReportCount uint64
	Name        string
}

// OllamaRecommendation represents AI-generated recommendations for climate control
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
