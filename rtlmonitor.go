package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	"gorm.io/gorm"
)

// rtlMonitor monitors RTL-SDR device for weather data using rtl_433
func rtlMonitor(db *gorm.DB, hub *EventHub) {
	fmt.Println("Running rtl_433")
	command := exec.Command("/home/ofer/repos/rtl_433/build/src/rtl_433", "-f", "433000000", "-F", "json", "-M", "time:iso:utc:tz")
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

		savedReport, inserted, err := saveWeatherReport(db, weatherReport)
		if err != nil {
			log.Println("Failed to save weather report:", err)
			continue
		}

		if inserted {
			hub.Broadcast(WeatherReportEvent{
				DbId:        savedReport.DbId,
				Time:        savedReport.Time.Unix(),
				DeviceModel: savedReport.DeviceModel,
			})
		}

	}
}
