# Testing Strategy

**Analysis Date:** 2026-05-19

---

## Go Backend Testing Status

**Current state: No automated tests exist for the Go backend.**

### What is missing

| Layer | Tests needed |
|-------|--------------|
| **Configuration** | Unit tests for `loadConfig()` / `saveConfig()` — verify defaults, file I/O errors, and malformed JSON handling |
| **Database** | Unit tests for `setupDatabase()` / `checkForDeviceModel()` — verify migrations, duplicate prevention, and device model tracking |
| **API handlers** | Unit tests for every handler in `handlers.go` — verify JSON responses, error codes, and missing-resource cases |
| **Recommendation worker** | Unit / integration tests for `queryOllamaForRecommendation()` — verify prompt formatting, JSON parsing, and Ollama response handling |
| **RTL-SDR monitor** | Integration tests for `rtlMonitor()` — verify parsing, duplicate detection, and database writes |

### Recommended testing framework

- **Package:** `testing` (Go standard library, no external dependencies)
- **Run:** `go test ./...`
- **CI integration:** Add `go test -race` (race detector) to catch concurrency bugs in the recommendation worker

---

## Angular Frontend Testing Status

**Current state: Placeholder tests only.** All 8 `*.spec.ts` files contain the same boilerplate smoke test:

```typescript
it('should be created', () => {
  expect(service).toBeTruthy();
});
```

These tests pass without mocking `HttpClient` or any Angular providers, so they **do not validate any actual functionality**.

### What is missing

| Area | Tests needed |
|------|--------------|
| **ApiService** | Tests for `getHistoricDataForDeviceModel()`, `getModels()`, `getLatestRecommendedReport()` — all require `HttpClientTestingModule` with `HttpTestingController` to assert HTTP requests and responses |
| **WeatherReport / DeviceModel types** | No tests exist; these types are simple data structures |
| **Settings Service** | Tests for persistence, config loading/saving |
| **Components** | No `DebugElement`, `By.css()` queries, or interaction tests exist; currently only the smoke test |
| **Live polling logic** (30 s interval in `ApiService`) | Timer-based tests using `fakeAsync` and `tick()` |

### Recommended testing infrastructure

| Concern | Recommendation |
|---------|----------------|
| **HTTP testing** | Use `HttpClientTestingModule` with `MockBackend` (or `HttpTestingController`) for all API service tests |
| **Component testing** | Use `TestBed.createComponent()` with `By.css()` queries; test `@Input()` bindings and user interactions |
| **Timer / Observable logic** | Use `fakeAsync` / `tick()` from `@angular/core/testing` to test `timer()` / `interval()` flows |
| **Coverage target** | Set a minimum of **>70% line coverage per file** as a CI gate |

---

## Recommended Next Steps

### Immediate (critical)

1. **Add Go backend tests** for `handlers.go` at a minimum — the HTTP API is the public surface and needs verification before any deployment.
2. **Replace placeholder Angular specs** with real tests for `ApiService.getHistoricDataForDeviceModel()` and `getLatestRecommendedReport()`. Without real HTTP tests, the frontend is effectively untested.

### Short-term (high-value)

3. Add a `Makefile` or GitHub Actions workflow that runs `go test` and `ng test --watch=false --browsers=ChromeHeadless` on every PR.
4. Add race detector to Go tests: `go test -race ./...` (the recommendation worker runs in a goroutine and uses shared state — very prone to race conditions).

### Medium-term (robustness)

5. Mock Ollama API responses with a local HTTP server (`httptest.NewServer`) in Go tests so they can run CI without an external AI model dependency.
6. Use `ng e2e` (Protractor/Cypress) for integration tests that cover the full user flow: view weather report → check recommendation → update settings.
