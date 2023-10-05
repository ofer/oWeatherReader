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

type Rtl433WeatherReport struct {
	Time          time.Time
	Model         string
	Battery_ok    uint8
	Temperature_F float32
	Humidity      uint8
	Mic           string
}

func getWeatherReportsByModel(c *gin.Context, db *gorm.DB) {
	model := c.Param("model")
	var weatherReports []WeatherReport
	result := db.Where("device_model = ?", model).Find(&weatherReports)
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

	// // Get user value
	// r.GET("/user/:name", func(c *gin.Context) {
	// 	user := c.Params.ByName("name")
	// 	value, ok := db[user]
	// 	if ok {
	// 		c.JSON(http.StatusOK, gin.H{"user": user, "value": value})
	// 	} else {
	// 		c.JSON(http.StatusOK, gin.H{"user": user, "status": "no value"})
	// 	}
	// })

	// Authorized group (uses gin.BasicAuth() middleware)
	// Same than:
	// authorized := r.Group("/")
	// authorized.Use(gin.BasicAuth(gin.Credentials{
	//	  "foo":  "bar",
	//	  "manu": "123",
	//}))
	// authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
	// 	"foo":  "bar", // user:foo password:bar
	// 	"manu": "123", // user:manu password:123
	// }))

	/* example curl for /admin with basicauth header
	   Zm9vOmJhcg== is base64("foo:bar")

		curl -X POST \
	  	http://localhost:8080/admin \
	  	-H 'authorization: Basic Zm9vOmJhcg==' \
	  	-H 'content-type: application/json' \
	  	-d '{"value":"bar"}'
	*/
	// authorized.POST("admin", func(c *gin.Context) {
	// 	user := c.MustGet(gin.AuthUserKey).(string)

	// 	// Parse JSON
	// 	var json struct {
	// 		Value string `json:"value" binding:"required"`
	// 	}

	// 	if c.Bind(&json) == nil {
	// 		db[user] = json.Value
	// 		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	// 	}
	// })

	return r
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
	command := exec.Command("/usr/bin/rtl_433", "-F", "json", "-M", "time:iso:utc:tz")
	// var outBuffer, errBuffer bytes.Buffer
	// command.Stdout = &outBuffer
	// command.Stderr = &errBuffer
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
			log.Fatal("Unmarshal Error:", err)
			return
		}

		weatherReport.Time = rtl433WeatherReport.Time
		weatherReport.DeviceModel = rtl433WeatherReport.Model
		weatherReport.TemperatureInF = rtl433WeatherReport.Temperature_F
		weatherReport.HumidityInPercentage = rtl433WeatherReport.Humidity

		log.Println("Finished unmarshalling json")
		db.Create(&weatherReport)
		log.Println("Finished adding weather report")
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
