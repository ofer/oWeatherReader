# oWeatherReader

Weather monitoring system with RTL-SDR and AI-powered recommendations.

## Features

- **RTL-SDR Weather Monitoring**: Monitors weather sensors using RTL-433
- **Web API**: RESTful API for accessing weather data
- **Database Storage**: SQLite database for weather reports and recommendations
- **AI-Powered Recommendations**: Uses Ollama to provide smart home automation recommendations

## New Feature: Ollama Integration

The system now queries an Ollama server every 15 minutes (configurable) to get AI-powered recommendations for:
- Whether to operate the air conditioner
- What temperature to set the air conditioner to
- Whether windows should be open
- Weather description with reasoning

### Configuration

Edit `config.json` to configure the Ollama integration:

```json
{
  "ollamaServerURL": "http://localhost:11434",
  "ollamaModel": "llama3.2",
  "indoorDeviceModel": "LaCrosse-TX141W",
  "outdoorDeviceModel": "LaCrosse-TX141W", 
  "recommendationIntervalMinutes": 15
}
```

### API Endpoints

- `GET /reports/latest` - Get the latest weather report
- `GET /reports/:model` - Get weather reports for a specific device model
- `GET /models` - Get all device models with report counts
- `GET /recommendations/latest` - **NEW** Get the latest AI recommendation

### Ollama Response Format

The system expects Ollama to return JSON in this format:

```json
{
  "shouldOperateAirConditioner": true,
  "temperatureToSetAirConditionerInF": 72,
  "shouldWindowBeOpen": false,
  "weatherDescription": "Indoor temperature is 75°F while outdoor is 85°F. AC recommended to maintain comfort."
}
```

### Setup Requirements

1. **Ollama Server**: Ensure Ollama is running on the configured URL
2. **Model**: The specified model should be available in Ollama
3. **Device Models**: Configure the correct device model names for indoor/outdoor sensors

### Database Schema

The new `OllamaRecommendation` table stores:
- Timestamp of recommendation
- AI recommendations (AC operation, temperature, window status)
- Weather description with reasoning  
- Indoor and outdoor temperatures used for the recommendation

## Building and Running

```bash
go build -o oWeatherReader main.go
./oWeatherReader
```

The service runs on port 6656 by default.
