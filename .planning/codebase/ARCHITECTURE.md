---
focus: arch
---

<!-- refreshed: 2026-05-19 -->
# Architecture

**Analysis Date:** 2026-05-19

## Pattern
Monolith — single Go HTTP server serving static Angular UI. Go handles API + backend logic, Angular handles frontend.

## Layers

```text
┌────────────────────────────────────────────────────────────┐
│                    Angular UI (ui/src/app/)                │
│  Components: home-summary-report, latest-reports,          │
│  device-report-history, high-low-history, settings,         │
│  weather-report-display, navigation                         │
├────────────────────────────────────────────────────────────┤
│            Go HTTP Server (handlers.go)                    │
│  /reports/latest, /reports/:model, /models                 │
│  /recommendations/latest, /config (GET+POST)               │
│  Static file serving (ui/dist/ui/)                         │
├────────────────────────────────────────────────────────────┤
│            Business Logic (Go)                             │
│  recommendationprocess.go — Ollama AI worker               │
│  rtlmonitor.go — rtl_433 subprocess monitor               │
├────────────────────────────────────────────────────────────┤
│            Data Layer (Go)                                 │
│  database.go — GORM + SQLite setup + migrations           │
│  models.go — WeatherReport, DeviceModel, OllamaRec        │
│  api_types.go — DTOs (OllamaRequest/Response, etc.)       │
└────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|---|------------|
| App Entry | Config load, DB init, workers startup, server listen | `main.go` |
| HTTP Routes | Gin router setup, static file serving, all API endpoints | `handlers.go` |
| Config | Config struct + global instance | `config.go` |
| Config I/O | Load/save config.json, defaults | `utils.go` |
| Database | GORM setup, SQLite, migrations, device model check | `database.go` |
| Domain Models | WeatherReport, DeviceModel, DeviceModelCount, OllamaRecommendation | `models.go` |
| API DTOs | OllamaRequest/Response, OllamaRecommendationResponse, Rtl433WeatherReport | `api_types.go` |
| AI Worker | queryOllama + periodic worker + restart logic | `recommendationprocess.go` |
| Device Monitor | rtl_433 subprocess, JSON parse, dedup, humidity correction, DB write | `rtlmonitor.go` |

## Data Flow

### rtl_433 Ingestion
1. `rtlmonitor.go` exec's rtl_433 binary (hardcoded path: `/home/ofer/repos/rtl_433/build/src/rtl_433`) at `433000000` Hz
2. Lines read via `bufio.Scanner`, unmarshaled to `Rtl433WeatherReport`
3. Temperature converted to Fahrenheit, humidity converted to uint8
4. `checkForDeviceModel()` (database.go:24) creates entries for unknown devices
5. Duplicate detection: timestamp + device_model match, or same temp+humidity for same device
6. Humidity correction: if last humidity < 5 and new humidity is 99, treat as error
7. Valid reports written to `weather_reports` table via GORM

### Recommendation Engine
1. `ollamaRecommendationWorker()` (recommendationprocess.go:129) starts as goroutine with configurable tick interval
2. Runs immediately on startup, then on each tick
3. Queries Ollama via HTTP POST to `{OllamaServerURL}/api/generate` (recommendationprocess.go:74)
4. Prompt includes latest indoor + outdoor temps, humidity, time
5. Parses JSON from Ollama's text response (extracts content between `{` and `}`)
6. Creates `OllamaRecommendation` record in SQLite
7. Config changes via POST /config trigger worker restart via `restartRecommendationWorker()`

### API Response Path
1. Client polls `GET /reports/latest` or `GET /reports/:model`
2. Handler queries SQLite via GORM (handlers.go:31-92)
3. JSON response returned, UI re-renders via `api.service.ts` every 30 seconds
4. `GET /recommendations/latest` serves latest AI recommendation
5. `GET /config` / `POST /config` serve and update runtime config

### Static UI Serving
1. `r.NoRoute()` in handlers.go (line 20) catches all non-API requests
2. Serves files from `./ui/dist/ui/`
3. SPA fallback: unknown paths → `index.html`

## Abstractions

- **WeatherReport** (models.go:8) — core domain entity representing one weather sensor reading
- **DeviceModel** (models.go:17) — tracks known device types, auto-populated via `checkForDeviceModel`
- **Config** (config.go:4) — global mutable config, hot-reloadable via POST /config
- **Global DB** (main.go:11) — `globalDB` package var enables config update hook access to DB
- **Worker Goroutines** — rtl monitor and recommendation both run as long-lived goroutines

## Entry Points

- **main.go:14** — entry point, orchestrates initialization
- **handlers.go:14** — `setupRouter()` registers all HTTP routes
- **rtlmonitor.go:15** — `rtlMonitor(db)` goroutine
- **recommendationprocess.go:129** — `ollamaRecommendationWorker(db)` goroutine

## Architectural Constraints

- **Single-threaded event loop** — Go's goroutine model with blocking I/O (SQLite, HTTP, subprocess)
- **Global mutable state** — `config` (config.go:13) and `globalDB` (main.go:11) are package-level variables accessible from any handler
- **Hardcoded binary path** — rtl_433 path `/home/ofer/repos/rtl_433/build/src/rtl_433` is not configurable (rtlmonitor.go:17)
- **No TLS** — the prompt mentioned TLS but main.go:30 only calls `r.Run(":6656")`; no TLS configuration present
- **No graceful shutdown** — no signal handling; the process exits abruptly

## Anti-Patterns

### Hardcoded Binary Path

**What happens:** rtl_433 binary path is hardcoded as `/home/ofer/repos/rtl_433/build/src/rtl_433` (rtlmonitor.go:17)
**Why it's wrong:** Cannot run on any machine other than the author's; not portable; deploy fails everywhere else
**Do this instead:** Add an `Rtl433BinaryPath` field to Config (config.go:4) and read it from config.json

### Global Mutable Config

**What happens:** `config` variable (config.go:13) is a global package-level variable modified by updateConfig handler
**Why it's wrong:** No mutex protection — concurrent reads/writes during config update cause race conditions; the recommendation interval change detection reads old value then writes, then the goroutine sees the new value but ticker was already created with the old duration
**Do this instead:** Wrap config in a `sync.RWMutex` or use `atomic.Value` for thread-safe access

### Subprocess Error Handling

**What happens:** If rtl_433 exits (command.Start error, or ReadError), `log.Fatal` kills the entire process (rtlmonitor.go:25-38)
**Why it's wrong:** A transient rtl_433 failure (device unplugged, permission error) brings down the entire application
**Do this instead:** Implement subprocess restart logic with exponential backoff, log the error, and continue

---

*Architecture analysis: 2026-05-19*
