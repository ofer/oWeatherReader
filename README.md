# oWeatherReader

Weather monitoring system with RTL-SDR and AI-powered recommendations.

## Features

- **RTL-SDR Weather Monitoring**: Monitors weather sensors using RTL-433
- **Web API**: RESTful API for accessing weather data
- **Database Storage**: SQLite database for weather reports and recommendations
- **AI-Powered Recommendations**: Uses OpenAI-compatible API servers to provide smart home automation recommendations

## OpenAI-Compatible API Integration

The system queries an OpenAI-compatible server every 15 minutes (configurable) to get AI-powered recommendations for:
- Whether to operate the air conditioner
- What temperature to set the air conditioner to
- Whether windows should be open
- Weather description with reasoning

### Configuration

Configuration can be managed in two ways:

1. **Static Configuration**: Edit `config.json` file (loaded at startup)
2. **Dynamic Configuration**: Use the API endpoints to update configuration at runtime

Configuration schema:
```json
{
  "openAIBaseURL": "http://localhost:11434",
  "openAIModel": "Qwen3.6-35B-A3B-UD-Q4_K_XL",
  "openAIAPIKey": "",
  "indoorDeviceModel": "LaCrosse-TX141W",
  "outdoorDeviceModel": "LaCrosse-TX141W",
  "recommendationIntervalMinutes": 15
}
```

#### Dynamic Configuration Updates

You can now update the configuration without restarting the service:

```bash
# Get current configuration
curl http://localhost:6656/config

# Update configuration
curl -X POST http://localhost:6656/config \
  -H "Content-Type: application/json" \
  -d '{
    "openAIBaseURL": "http://localhost:11434",
    "openAIModel": "Qwen3.6-35B-A3B-UD-Q4_K_XL",
    "indoorDeviceModel": "New-Indoor-Sensor",
    "outdoorDeviceModel": "New-Outdoor-Sensor",
    "recommendationIntervalMinutes": 30
  }'
```

**Features of Dynamic Updates:**
- Changes take effect immediately
- Configuration is automatically saved to `config.json`
- If recommendation interval changes, the worker is automatically restarted
- Validation ensures all required fields are provided

### API Endpoints

- `GET /reports/latest` - Get the latest weather report
- `GET /reports/:model` - Get weather reports for a specific device model
- `GET /reports/events` - Server-Sent Events stream for new weather report notifications
- `GET /models` - Get all device models with report counts
- `GET /recommendations/latest` - Get the latest AI recommendation
- `GET /config` - Get current configuration
- `POST /config` - Update configuration dynamically

#### Server-Sent Events (`/reports/events`)

Subscribe to real-time notifications when the server accepts a new weather report. The response uses `text/event-stream` format.

Event payload:
```json
{
  "dbId": 123,
  "time": 1684516800,
  "deviceModel": "Bresser-3CH"
}
```

The UI subscribes to this endpoint and refreshes the weather history graph when an event matches the currently selected device model.

### AI Recommendation Response Format

The AI server is expected to return JSON in this format:

```json
{
  "shouldOperateAirConditioner": true,
  "temperatureToSetAirConditionerInF": 72,
  "shouldWindowBeOpen": false,
  "weatherDescription": "Indoor temperature is 75°F while outdoor is 85°F. AC recommended to maintain comfort."
}
```

### Setup Requirements

1. **OpenAI-Compatible Server**: Ensure an OpenAI-compatible API server (e.g., Ollama, vLLM) is running on the configured URL
2. **Model**: The specified model should be available on the server
3. **API Key**: If the server requires authentication, provide the API key in `openAIAPIKey` (leave empty for local servers)
4. **Device Models**: Configure the correct device model names for indoor/outdoor sensors

### Database Schema

The `OllamaRecommendation` table stores:
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

## Running Tests

```bash
# Go tests
go test ./...

# Angular tests (from ui/ directory)
cd ui && npm test -- --watch=false --browsers=ChromeHeadless

# Angular build
cd ui && npm run build
```
