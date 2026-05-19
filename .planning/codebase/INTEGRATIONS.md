---
focus: tech - integrations
last_mapped_commit: 1bc4aa72d752dfd67991b3a8dfd8de597ce16c7d
---

# Integrations

## External APIs & Services

### Ollama AI Service
- **Endpoint:** `{ollamaServerURL}/api/generate` — configurable via `config.json`
- **Default URL:** `http://10.0.0.17:11434` (network host, localhost defaults in `utils.go:14`)
- **Transport:** HTTP POST with JSON body (`application/json`)
- **SDK:** `net/http.Post` directly (no Ollama SDK)
- **Model:** configurable via `config.ollamaModel` (default: `gemma4:26b`)
- **Called from:** `recommendationprocess.go:74` — `http.Post(config.OllamaServerURL+"/api/generate", ...)`
- **Response parsing:** `OllamaResponse` struct (`api_types.go:15-19`); response body is further parsed as JSON to produce `OllamaRecommendationResponse`
- **Called by:** `ollamaRecommendationWorker` goroutine, triggered every `RecommendationIntervalMinutes` (default 15 min)

### rtl_433 (Local Process — Hardware Sensor Integration)
- **Binary path:** `/home/ofer/repos/rtl_433/build/src/rtl_433` (hardcoded in `rtlmonitor.go:17`)
- **Command-line arguments:** `-f 433000000 -F json -M time:iso:utc:tz`
- **Protocol:** 433 MHz RF — receives wireless weather sensor telemetry
- **Output format:** JSON per line on stdout (parsed via `bufio.NewReader` + `json.Unmarshal`)
- **Sensor models configured:**
  - Indoor: `inFactory-TH` (per `config.json`)
  - Outdoor: `Bresser-3CH` (per `config.json`)
- **Called from:** `rtlmonitor.go:17` — `exec.Command("/home/ofer/repos/rtl_433/build/src/rtl_433", ...)`
- **Error handling:** `log.Fatal` on startup failures (no automatic restart); continuous read loop on stderr

## Databases

### SQLite
- **File:** `weather.db` (local filesystem, path hardcoded in `database.go:13`)
- **ORM:** GORM (`gorm.io/gorm v1.25.4`) with `gorm.io/driver/sqlite v1.5.3`
- **Migration:** AutoMigrate for three tables at startup (`database.go:18`):
  - `weather_reports` — weather sensor readings
  - `device_models` — sensor device model catalog
  - `ollama_recommendations` — AI-generated climate recommendations
- **Schema (Go models → tables):**
  - `WeatherReport` (`models.go:8-14`): `db_id`, `time`, `device_model`, `temperature_in_f`, `humidity_in_percentage`
  - `DeviceModel` (`models.go:17-21`): `db_id`, `device_model`, `name`
  - `OllamaRecommendation` (`models.go:31-40`): `db_id`, `time`, `should_operate_air_conditioner`, `temperature_to_set_air_conditioner_in_f`, `should_window_be_open`, `weather_description`, `indoor_temperature_f`, `outdoor_temperature_f`
- **Connection:** Single global `*gorm.DB` instance stored in `main.go:11` as `globalDB`

## Auth

- **No authentication detected.** All API endpoints are open without auth middleware.
- The server serves a static Angular SPA that communicates directly with the Go HTTP server on port 6656.

## Webhooks & Events

### Inbound (no webhooks)
- No webhook endpoints or callback URLs configured.

### Outbound events
- **Scheduled HTTP calls to Ollama:** `POST {ollamaServerURL}/api/generate` every `RecommendationIntervalMinutes` (background goroutine in `recommendationprocess.go:129`)
- **No SSE/WebSocket support in current code** — `plan.md` documents a planned Phase 1 feature (`GET /reports/events` SSE) that has not yet been implemented.

## Planned Integrations (from plan.md)

| Phase | Feature | Status |
|-------|---------|--------|
| Phase 1 | SSE endpoint `GET /reports/events` | Not implemented |
| Phase 2 | Extract report save/filter logic for testability | Not implemented |
| Phase 3 | Go unit tests for SSE and insert logic | Not implemented |
| Phase 4 | Angular `ApiService` SSE stream client | Not implemented |
| Phase 5 | Graph auto-refresh on server events | Not implemented |
| Phase 6 | Angular unit tests for events | Not implemented |
| Phase 7 | Integration verification | Not implemented |

## Environment Configuration

### Required environment
- **No external environment variables** — all configuration is in `config.json` (runtime-writable)
- **rtl_433 binary** must exist at the hardcoded path `/home/ofer/repos/rtl_433/build/src/rtl_433`
- **RTL-SDR USB dongle** attached and accessible to rtl_433 process

### config.json fields

| Field | Type | Required | Default | Purpose |
|-------|------|----------|---------|---------|
| `ollamaServerURL` | string | No (validated at POST time) | `http://localhost:11434` | Ollama service base URL |
| `ollamaModel` | string | No (validated at POST time) | `llama3.2` | Model name for Ollama requests |
| `indoorDeviceModel` | string | No (validated at POST time) | `LaCrosse-TX141W` | rtl_433 model name for indoor sensor |
| `outdoorDeviceModel` | string | No (validated at POST time) | `LaCrosse-TX141W` | rtl_433 model name for outdoor sensor |
| `recommendationIntervalMinutes` | int | No (validated at POST time: >0) | 15 | How often to query Ollama (minutes) |

### Runtime API endpoints (Go server)

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| `GET` | `/reports/latest` | None | Latest weather report |
| `GET` | `/reports/:model` | None | Weather history for a device model (5 days) |
| `GET` | `/models` | None | All device models with report counts |
| `GET` | `/recommendations/latest` | None | Latest AI recommendation |
| `GET` | `/config` | None | Current configuration |
| `POST` | `/config` | None | Update configuration dynamically |

---

*Integration audit: 2026-05-19*
