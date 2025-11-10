package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"

	"gorm.io/gorm"
)

// rtlMonitor monitors RTL-SDR device for weather data using rtl_433
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
