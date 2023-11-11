package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type WeatherReport struct {
	DbId                 uint      `gorm:"primaryKey;autoIncrement"`
	Time                 time.Time `gorm:"index"`
	DeviceModel          string    `gorm:"index"`
	TemperatureInF       float32
	HumidityInPercentage uint8
}

type DeviceModel struct {
	DeviceModel string
	ReportCount uint64
}

type Rtl433WeatherReport struct {
	Time          time.Time
	Model         string
	Id            uint32
	Channel       uint8
	Battery_ok    uint8
	Temperature_F *float32
	Temperature_C *float32
	Humidity      float32
	Button        *uint8
	Mic           string
}

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

	// Ping test
	r.GET("/reports/latest", func(c *gin.Context) {
		getLatestWeatherReport(c, db)
	})

	r.GET("/reports/:model", func(c *gin.Context) {
		getWeatherReportsByModel(c, db)
	})

	r.GET("/models", func(c *gin.Context) {
		getModels(c, db)
	})

	return r
}

func getModels(c *gin.Context, db *gorm.DB) {
	var deviceModels []DeviceModel
	result := db.Model(&WeatherReport{}).Select("device_model, COUNT(*) as report_count").Group("device_model").Find(&deviceModels)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve weather reports"})
		return
	}
	c.JSON(http.StatusOK, deviceModels)
}

func getLatestWeatherReport(c *gin.Context, db *gorm.DB) {
	var weatherReport WeatherReport
	result := db.Order("time desc").First(&weatherReport)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve latest weather report"})
		return
	}
	c.JSON(http.StatusOK, weatherReport)
}

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

		db.Create(&weatherReport)
	}
}

func setupDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("weather.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database", err)
	}

	db.AutoMigrate(&WeatherReport{})

	return db
}

func main() {
	fmt.Println("Starting oWeatherReader")

	db := setupDatabase()
	go rtlMonitor(db)

	r := setupRouter(db)
	// Listen and Server in 0.0.0.0:8080
	r.Run(":6656")
}
