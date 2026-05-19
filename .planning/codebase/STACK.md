---
focus: tech
last_mapped_commit: 1bc4aa72d752dfd67991b3a8dfd8de597ce16c7d
---

# Stack

## Languages

- **Go** (1.21.1) — primary language, backend server
- **TypeScript** (~5.1.3) — Angular UI layer
- **HTML/CSS** — Angular component templates

## Frameworks & Tools

### Backend (Go)
- **Gin** v1.9.1 — HTTP web framework and router
- **GORM** v1.25.4 — ORM for SQLite
- **sqlite3 driver (mattn/go-sqlite3)** v1.14.17 — GORM SQLite driver (CGO-based)

### Frontend (Angular)
- **Angular** v16.2.0 — frontend framework
  - `@angular/core`, `@angular/common`, `@angular/compiler`, `@angular/platform-browser`, `@angular/platform-browser-dynamic`, `@angular/forms`, `@angular/router`
- **@angular/material** v16.2.7 — Material Design UI components
- **ngx-echarts** v16.0.0 — Angular wrapper for ECharts
- **echarts** v5.4.3 — charting library
- **rxjs** ~7.8.0 — reactive extensions

### Testing
- **karma** ~6.4.0 + **jasmine-core** ~4.6.0 — Angular unit test runner
- **go test** (standard library) — Go (not yet used; plan.md notes this as a gap)

### Build Tools
- **Angular CLI** ^16.2.4 — Angular build, serve, test
- **`go build`** — Go compilation
- **run.sh** — convenience script: builds Angular, then runs Go server

## Runtime

- **Go** — runs as a compiled binary (`oWeatherReader`), listens on port **6656** (hardcoded in `main.go:30`)
- **Angular** — builds to `./ui/dist/ui/`, served as static files by the Go Gin router
- **Docker** — not detected
- **Process model** — single Go binary; spawns `rtl_433` via `exec.Command` and runs two background goroutines (`rtlMonitor`, `ollamaRecommendationWorker`)

## Dependencies

### Go (go.mod)
**Direct dependencies:**
- `github.com/gin-gonic/gin v1.9.1` — HTTP framework
- `gorm.io/driver/sqlite v1.5.3` — SQLite driver for GORM
- `gorm.io/gorm v1.25.4` — ORM

**Key indirect dependencies:**
- `github.com/mattn/go-sqlite3 v1.14.17` — CGO SQLite driver
- `golang.org/x/net v0.15.0` — HTTP utilities used by Gin
- `github.com/ugorji/go/codec v1.2.11` — fast JSON codec used by Gin
- `google.golang.org/protobuf v1.31.0` — protocol buffer runtime

### Frontend (ui/package.json)
- Angular 16 full framework
- @angular/material + @angular/cdk
- ngx-echarts + echarts (charting)
- jasmine-core + karma (testing)

## Configuration

### Runtime configuration (`config.json`)
```json
{
  "ollamaServerURL": "http://10.0.0.17:11434",
  "ollamaModel": "gemma4:26b",
  "indoorDeviceModel": "inFactory-TH",
  "outdoorDeviceModel": "Bresser-3CH",
  "recommendationIntervalMinutes": 15
}
```
- Loaded at startup via `loadConfig()` in `utils.go`
- Can also be updated at runtime via `POST /config` endpoint (persists back to file)

### Server configuration
- **Port:** `6656` (hardcoded in `main.go:30`)
- **Database:** SQLite file `weather.db` (hardcoded in `database.go:13`)
- **rtl_433 binary path:** `/home/ofer/repos/rtl_433/build/src/rtl_433` (hardcoded in `rtlmonitor.go:17`)
- **Frequency:** `433000000` Hz (hardcoded in rtl_433 command line)
- **Static UI files:** `./ui/dist/ui/` (hardcoded in `handlers.go:24`)

### Build/deployment configuration
- `run.sh` — builds Angular with `ng build --base-href=/weather/ -c development` then runs Go with `go run .`
- `go.mod` — Go module configuration (module name: `oWeatherReader/main`)
- `ui/angular.json` — Angular workspace config
- `ui/tsconfig.json` / `ui/tsconfig.app.json` — TypeScript compilation settings

## Platform Requirements

- **Linux** — rtl_433 binary is Linux-specific, CGO sqlite3 requires build tools
- **RTL-SDR hardware** — required for rtl_433 to receive weather sensor data (433 MHz)
- **Ollama server** — required on the configured network URL (default `http://10.0.0.17:11434`)
- **Port 6656** — must be available for the HTTP server

---

## Architecture Summary

```
┌─────────────────────────────────────┐
│  Angular UI (ui/dist/ui/)          │  ←  SPA: Material components + ngx-echarts
│  TypeScript / RxJS / ECharts        │
└──────────────┬──────────────────────┘
               │  REST API (HTTP JSON)
               ▼
┌─────────────────────────────────────┐
│  Go Server (oWeatherReader)        │  ←  Gin router on :6656
│  ├── handlers.go   (REST endpoints) │
│  ├── database.go   (GORM/SQLite)    │
│  ├── rtlmonitor.go   (rtl_433 proc) │
│  ├── recommendationprocess.go       │
│  └── utils.go      (config I/O)     │
└───────┬──────────┬──────────────────┘
        │          │
   ┌────▼───┐  ┌──▼──────────┐
   │SQLite  │  │ Ollama HTTP │  ← http.Post to ollamaServerURL + /api/generate
   │weather │  │ api/generate│
   │.db     │  └─────────────┘
   └────────┘
        │
   ┌────▼──────────────┐
   │ rtl_433 (exec.Cmd)│  ← 433 MHz SDR weather sensor receiver
   └───────────────────┘
```

---

*Stack analysis: 2026-05-19*
