---
focus: arch - structure
---

# Structure

**Analysis Date:** 2026-05-19

## Directory Layout

```
oWeatherReader/
├── main.go                    # Application entry point
├── handlers.go                # HTTP routes and API handlers
├── config.go                  # Config struct definition + global var
├── database.go                # GORM setup, migrations, device model helper
├── models.go                  # Domain model structs
├── api_types.go               # API DTO structs (Ollama, rtl_433)
├── utils.go                   # Config load/save JSON helpers
├── recommendationprocess.go   # Ollama recommendation engine + worker
├── rtlmonitor.go              # rtl_433 subprocess monitor
├── config.json                # Runtime configuration file
├── go.mod                     # Go module definition (go 1.21.1)
├── go.sum                     # Go dependency checksums
├── weather.db                 # SQLite database (runtime artifact)
├── oWeatherReader.log         # Runtime log file
├── ui/                        # Angular frontend
│   ├── package.json           # Angular 16 + echarts dependencies
│   └── src/
│       ├── app/               # Angular app module
│       │   ├── app.module.ts           # Root module
│       │   ├── app-routing.module.ts   # Route definitions
│       │   ├── app.component.ts/.html/.scss   # Root component
│       │   ├── api.service.ts            # HTTP API client
│       │   ├── settings.service.ts       # Config management service
│       │   ├── latest-weather-reporter.service.ts  # Polling service
│       │   ├── weather-report.ts                 # WeatherReport TS interface
│       │   ├── device-model.ts               # DeviceModel TS interface
│       │   ├── home-summary-report-component/  # Dashboard summary
│       │   ├── latest-reports/               # Latest reports list
│       │   ├── weather-report-display/       # Report detail card
│       │   ├── device-report-history/        # Historical data table
│       │   ├── high-low-history/             # Min/max temperature chart
│       │   ├── settings/                     # Config editing form
│       │   └── navigation/                   # App navigation bar
│       ├── assets/           # Static assets
│       ├── styles.scss       # Global styles
│       ├── index.html        # SPA entry HTML
│       ├── main.ts           # Angular bootstrap
│       └── manifest.webmanifest  # PWA manifest
└── dist/ui/                  # Compiled frontend (ng build output)
```

## Directory Purposes

**Root (`/`):**
- Everything Go: all .go source files, config, module definition, runtime artifacts
- All files in a single Go package (`package main`)

**`ui/src/app/`:**
- Angular component library serving the UI layer
- 8 component directories + 5 service/interface files
- Components follow Angular file-per-component pattern (.component.ts/.html/.scss)

**`ui/src/app/settings/`:**
- Configuration editing page (form to update config via POST /config)

## Key File Locations

**Entry Points:**
- `main.go`: HTTP server on port 6656, orchestrates all subsystems
- `ui/src/main.ts`: Angular bootstrap

**Configuration:**
- `config.go`: Config struct + global `config` var
- `config.json`: Runtime config (Ollama, device models, interval)

**Backend:**
- `handlers.go`: All HTTP routes — `/reports/latest`, `/reports/:model`, `/models`, `/recommendations/latest`, `/config` (GET + POST)
- `database.go`: SQLite via GORM, auto-migrates 3 tables
- `models.go`: `WeatherReport`, `DeviceModel`, `OllamaRecommendation`
- `recommendationprocess.go`: `queryOllamaForRecommendation()`, `ollamaRecommendationWorker()`
- `rtlmonitor.go`: `rtlMonitor()`, subprocess + JSON parse + dedup

**Frontend:**
- `ui/src/app/api.service.ts`: HTTP client wrapping backend endpoints
- `ui/src/app/settings.service.ts`: Config fetch/update
- `ui/src/app/latest-weather-reporter.service.ts`: Polling service (30s interval)
- `ui/src/app/weather-report.ts`: TypeScript interface for WeatherReport
- `ui/src/app/device-model.ts`: TypeScript interface for DeviceModel

## Naming Conventions

**Go:**
- Types: PascalCase — `WeatherReport`, `DeviceModel`, `OllamaRecommendation`
- Variables: camelCase — `weatherReport`, `recommendationWorkerCancel`
- Function names: camelCase — `getLatestWeatherReport`, `queryOllamaForRecommendation`
- Files: lowercase.go — `main.go`, `handlers.go`, `recommendationprocess.go`

**Angular:**
- Components: kebab-case directory + PascalCase file — `latest-reports/`, `LatestReportsComponent`
- Services: PascalCase — `ApiService`, `SettingsService`
- Interfaces/Classes: PascalCase — `WeatherReport`, `DeviceModel`
- TypeScript variables: camelCase — `weatherReports`, `recommendation`
- HTML templates: PascalCase component names in templates

## Module Boundaries

- **Go backend**: Single package (`package main`), all 9 .go files in root
- **Angular frontend**: Separate Angular 16 project in `ui/`
- **API boundary**: HTTP REST between Go handlers and Angular `api.service.ts`
- **No shared types**: TypeScript interfaces in the frontend are independent copies (weather-report.ts, device-model.ts) — not code-generated from Go types

## Where to Add New Code

**New API endpoint:** Add route in `setupRouter()` (handlers.go:14) and handler function in handlers.go or a new file in the same package.

**New Go model:** Add struct to `models.go` (or `api_types.go` for DTO-only types).

**New business logic:** New file in root package following the existing pattern (single package, all files at top level).

**New UI component:** Create directory in `ui/src/app/` with the standard Angular four-file pattern (.component.ts, .component.html, .component.scss, .component.spec.ts). Register in `app-routing.module.ts` or `app.module.ts` as needed.

**New config field:** Add to `Config` struct in `config.go`, to `config.json`, to `updateConfig` validation in `handlers.go:111`, to `loadConfig` defaults in `utils.go:9`, and to `SettingsService` in frontend.

## Special Directories

**`ui/dist/ui/`:** Angular build output directory. Not in source control. Generated by `ng build`. Served by the Go HTTP server's NoRoute handler.

**`weather.db`:** SQLite database file. Runtime artifact, not in source control. Created automatically on first run via GORM AutoMigrate.

---

*Structure analysis: 2026-05-19*
