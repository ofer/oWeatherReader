# Coding Conventions

**Analysis Date:** 2026-05-19

## Overview

oWeatherReader is a Go (backend) + Angular (frontend) application. The Go backend uses a flat package structure with one `main` package. The Angular frontend follows Angular CLI conventions within `ui/`.

---

## Go Conventions

### Naming Patterns

**Files:**
- **snake_case** — `handlers.go`, `rtlmonitor.go`, `recommendationprocess.go`
- **File name matches the primary type or function it contains** — e.g., `models.go` holds all domain types; `config.go` holds `Config` struct.

**Types:**
- **PascalCase (exported)** — `WeatherReport`, `DeviceModel`, `OllamaRecommendation`, `Rtl433WeatherReport`
- **Lowercase (unexported)** — none currently exist; all types are in the `main` package so everything is effectively package-scoped.

**Functions:**
- **PascalCase (exported)** — none currently.
- **camelCase (unexported)** — `rtlMonitor`, `setupDatabase`, `checkForDeviceModel`, `setupRouter`, `queryOllamaForRecommendation`, `getConfig`, `updateConfig`

**Variables:**
- **camelCase** — `db`, `config`, `globalDB`, `err`, `recommendation`

**Constants:**
- No package-level constants found. Port `6656` and path literals are hardcoded inline.

### Structural Patterns

**Package Organization:**
- **Single-package flat structure.** All Go files live in the root `main` package.
- Co-location by *concept* — `models.go` for types, `database.go` for DB setup, `handlers.go` for HTTP handlers, `config.go` for config structs, `utils.go` for helpers, `rtlmonitor.go` for RTL-SDR monitoring, `recommendationprocess.go` for the Ollama worker.

**Imports:**
- Go standard library grouped together.
- Third-party / external groups are **not** separated by blank lines (not enforced):
  ```go
  import (
      "context"
      "fmt"
      "log"
      "net/http"
      "strings"
      "sync"
      "time"

      "gorm.io/gorm"
  )
  ```

**Error Handling — `if err != nil` Pattern:**
- **Every `err` is explicitly checked** after function calls that return `error`.
- **Inline early-return style** — errors are handled immediately with a log/alert and return from the function.
- **Verbose, context-rich error messages** using `fmt.Errorf(...)` for wrapping; `log.Printf(...)` for non-fatal logging.

---

## Angular Conventions

### Naming Patterns

**Files (kebab-case):**
- `weather-report.ts`, `api.service.ts`, `app.component.ts`, `home-summary-report-component.component.ts`
- Angular CLI generated naming for components: `<name>.component.ts/.html/.scss`

**Types/Classes (PascalCase):**
- `WeatherReport`, `DeviceModel`, `HouseHvacRecommendation`, `ApiService`, `SettingsService`

**Functions/Methods (camelCase):**
- `getLatestRecommendedReport()`, `getModels()`, `getHistoricDataForDeviceModel()`
- Component methods: `ngOnInit()`, `ngOnDestroy()` (Angular lifecycle hooks always PascalCase)

**Variables (camelCase):**
- `weatherReports`, `temperatureCelsius`, `weatherReport`

### Structural Patterns

**Component Structure:**
- `@Component` decorator with metadata: `selector`, `templateUrl`, `styleUrls`.
- `@Input()` and `@Output()` used for component-to-component communication.
- **`ngOnInit()`** is the primary initialization hook.
- **`ngOnDestroy()`** used for cleanup (e.g., `subscription.unsubscribe()`, clearing `setInterval`).

**Service Structure:**
- `@Injectable({ providedIn: 'root' })` for singleton-scoped services.
- Methods **always return `Observable<T>`** — `HttpClient` methods are not subscribed to inside the service; subscribers live in components.
- `timer` / `interval` used with RxJS for polling (e.g., the 30-second live data observer in `ApiService`).

**Imports:**
- Standard group separation: Angular/core → Angular/common → RxJS → Local modules.
  ```typescript
  import { Injectable } from '@angular/core';
  import { HttpClient } from '@angular/common/http';
  import { Observable, retry, switchMap, timer } from 'rxjs';
  ```

---

## Testing Conventions

### Go

| Aspect | Finding |
|--------|-----|
| **Test files** | None found — the Go backend has **no `*_test.go` files**. |
| **Test framework** | N/A |
| **Coverage** | No `go.mod` coverage flags or CI test jobs present. |

### Angular

| Aspect | Finding |
|--------|-----|
| **Test framework** | Jasmine + Karma (standard Angular CLI) |
| **Test directory** | `*.spec.ts` files co-located alongside source files in `ui/src/app/` and subdirectories. |
| **Test files found** | 8 spec files across the project |

Test file list:

- `app.component.spec.ts`
- `api.service.spec.ts`
- `home-summary-report-component.component.spec.ts`
- `high-low-history.component.spec.ts`
- `latest-reports.component.spec.ts`
- `device-report-history.component.spec.ts`
- `navigation.component.spec.ts`
- `settings.component.spec.ts`
- `weather-report-display.component.spec.ts`

**Typical test pattern:** Minimal scaffolding — `TestBed.configureTestingModule({})` with an empty providers array, `beforeEach` injecting the service, and a single smoke test:

```typescript
it('should be created', () => {
  expect(service).toBeTruthy();
});
```

**Mocking:** No tests mock `HttpClient`. The `ApiService` spec configures an empty module — in practice, tests would fail at runtime without mocking. This indicates tests are currently **placeholder-only** and not run in CI.

---

## Summary — Quality Observations

| Category | Status |
|----------|------|
| Naming consistency | **Good** — both Go and Angular sections follow their respective community standards |
| Error handling | **Good** — Go backend checks every error; no `panic` or `recover` usage |
| Code structure | **Adequate** — single-package Go; Angular follows standard module pattern |
| Testing coverage | **Critical gap** — Go has zero tests. Angular has placeholder-only specs that do not validate functionality |
| Constants / magic numbers | **Improvement needed** — Port `6656` and paths are hardcoded rather than extracted to constants |
| Imports grouping | **Mixed** — Go does not enforce group separation; Angular does separate them by blank lines |
