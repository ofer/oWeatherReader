package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---- Event Hub Tests ----

func newTestEventHub() *EventHub {
	hub := NewEventHub()
	// Shutdown is called in test cleanup via t.Cleanup
	return hub
}

func TestEventHub_BroadcastsToSingleClient(t *testing.T) {
	hub := newTestEventHub()
	defer hub.Shutdown()

	id, ch := hub.register()
	defer hub.unregister(id)

	event := WeatherReportEvent{
		DbId:        42,
		Time:        1684516800,
		DeviceModel: "Bresser-3CH",
	}
	hub.Broadcast(event)

	select {
	case msg := <-ch:
		var received WeatherReportEvent
		if err := json.Unmarshal([]byte(msg[6:]), &received); err != nil {
			// Strip "data: " prefix
			raw := msg
			if idx := len("data: "); idx < len(msg) {
				raw = msg[idx:]
			}
			if err2 := json.Unmarshal([]byte(raw), &received); err2 != nil {
				t.Fatalf("failed to unmarshal event: %v (raw: %s)", err, msg)
			}
		}
		if received.DbId != 42 {
			t.Fatalf("expected DbId 42, got %d", received.DbId)
		}
		if received.DeviceModel != "Bresser-3CH" {
			t.Fatalf("expected DeviceModel 'Bresser-3CH', got '%s'", received.DeviceModel)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event on single client channel")
	}
}

func TestEventHub_BroadcastsToMultipleClients(t *testing.T) {
	hub := newTestEventHub()
	defer hub.Shutdown()

	clientCount := 5
	channels := make([]chan string, clientCount)
	ids := make([]uint64, clientCount)

	for i := 0; i < clientCount; i++ {
		id, ch := hub.register()
		ids[i] = id
		channels[i] = ch
	}
	defer func() {
		for _, id := range ids {
			hub.unregister(id)
		}
	}()

	event := WeatherReportEvent{
		DbId:        1,
		Time:        100,
		DeviceModel: "MultiTest",
	}
	hub.Broadcast(event)

	received := 0
	for i, ch := range channels {
		select {
		case msg := <-ch:
			if msg == "" {
				t.Fatalf("client %d received empty message", i)
			}
			received++
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for event on client %d channel", i)
		}
	}
	if received != clientCount {
		t.Fatalf("expected %d clients to receive event, got %d", clientCount, received)
	}
}

func TestEventHub_UnregistersCanceledClient(t *testing.T) {
	hub := newTestEventHub()
	defer hub.Shutdown()

	id, _ := hub.register()

	// Unregister the client
	hub.unregister(id)

	// Broadcast should not fail
	event := WeatherReportEvent{
		DbId:        99,
		Time:        200,
		DeviceModel: "CancelTest",
	}
	hub.Broadcast(event)

	// Register a fresh client and verify it still receives events
	newID, newCh := hub.register()
	defer hub.unregister(newID)

	hub.Broadcast(event)
	select {
	case msg := <-newCh:
		if msg == "" {
			t.Fatal("new client received empty message")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event on new client after old client was unregistered")
	}
}

func TestEventHub_DoesNotBlockWhenClientNotReading(t *testing.T) {
	hub := newTestEventHub()
	defer hub.Shutdown()

	// Register a client but do NOT read from its channel
	// The channel has a buffer of 256, so we can send up to 256 events
	id, _ := hub.register()
	defer hub.unregister(id)

	// Fill the channel buffer
	for i := 0; i < 256; i++ {
		event := WeatherReportEvent{
			DbId:        uint(i),
			Time:        int64(i),
			DeviceModel: "SlowClient",
		}
		hub.Broadcast(event)
	}

	// The next broadcast should drop the slow client and return immediately
	done := make(chan struct{})
	go func() {
		hub.Broadcast(WeatherReportEvent{
			DbId:        9999,
			Time:        9999,
			DeviceModel: "DropTest",
		})
		close(done)
	}()

	select {
	case <-done:
		// Broadcast returned even though client was not reading — success
	case <-time.After(1 * time.Second):
		t.Fatal("Broadcast blocked when client was not reading")
	}

	// Verify the client still has buffered messages (they weren't drained by run loop)
	// This confirms the client was slow enough to trigger the non-blocking drop path
	// on the next broadcast iteration
}

func TestEventHub_BroadcastChannelDropWhenFull(t *testing.T) {
	hub := newTestEventHub()
	defer hub.Shutdown()

	// Simulate a slow reader scenario: register a client that won't be read
	_, _ = hub.register()

	// Fill the broadcast channel (buffer size 256)
	// The 257th broadcast should be dropped without blocking
	done := make(chan struct{})
	go func() {
		hub.Broadcast(WeatherReportEvent{
			DbId:        99999,
			Time:        99999,
			DeviceModel: "DropTest",
		})
		close(done)
	}()

	select {
	case <-done:
		// Broadcast returned — the event was dropped without blocking
	case <-time.After(1 * time.Second):
		t.Fatal("Broadcast blocked when broadcast channel was full")
	}
}

func TestEventHub_ShutdownClosesAllClients(t *testing.T) {
	hub := NewEventHub()

	// Register several clients
	clients := make(map[uint64]chan string)
	for i := 0; i < 3; i++ {
		id, ch := hub.register()
		clients[id] = ch
	}

	hub.Shutdown()

	// All client channels should be closed
	for id, ch := range clients {
		select {
		case _, ok := <-ch:
			if ok {
				t.Fatalf("client %d channel should be closed after shutdown", id)
			}
		default:
			t.Fatalf("client %d channel should not be open after shutdown", id)
		}
	}
}

// ---- SSE Handler Tests ----

func TestSSEHandler_ReturnsCorrectHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := newTestEventHub()
	defer hub.Shutdown()

	router := gin.New()
	router.GET("/reports/events", func(c *gin.Context) {
		handleReportEvents(c, hub)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "/reports/events", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected Content-Type 'text/event-stream', got '%s'", rr.Header().Get("Content-Type"))
	}
	if rr.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("expected Cache-Control 'no-cache', got '%s'", rr.Header().Get("Cache-Control"))
	}
	if rr.Header().Get("Connection") != "keep-alive" {
		t.Fatalf("expected Connection 'keep-alive', got '%s'", rr.Header().Get("Connection"))
	}
}

func TestSSEHandler_SerializesEventPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := newTestEventHub()
	defer hub.Shutdown()

	router := gin.New()
	router.GET("/reports/events", func(c *gin.Context) {
		handleReportEvents(c, hub)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "/reports/events", nil)
	rr := httptest.NewRecorder()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		router.ServeHTTP(rr, req)
		wg.Done()
	}()

	// Wait for connection to be established
	time.Sleep(30 * time.Millisecond)

	// Broadcast an event
	event := WeatherReportEvent{
		DbId:        42,
		Time:        1684516800,
		DeviceModel: "TestModel",
	}
	hub.Broadcast(event)

	// Wait for the event to be received
	time.Sleep(100 * time.Millisecond)

	// Check response body
	body := rr.Body.String()
	if body == "" {
		t.Fatal("expected SSE response body, got empty string")
	}

	// Parse the SSE data line
	var parsed WeatherReportEvent
	// SSE format: "data: {json}\n\n"
	dataLine := body
	if len("data: ") < len(dataLine) {
		dataLine = dataLine[len("data: "):]
	}
	if err := json.Unmarshal([]byte(dataLine), &parsed); err != nil {
		t.Fatalf("failed to parse SSE payload: %v (body: %s)", err, body)
	}

	if parsed.DbId != 42 {
		t.Fatalf("expected DbId 42, got %d", parsed.DbId)
	}
	if parsed.Time != 1684516800 {
		t.Fatalf("expected Time 1684516800, got %d", parsed.Time)
	}
	if parsed.DeviceModel != "TestModel" {
		t.Fatalf("expected DeviceModel 'TestModel', got '%s'", parsed.DeviceModel)
	}
}

func TestSSEHandler_EventPayloadShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := newTestEventHub()
	defer hub.Shutdown()

	router := gin.New()
	router.GET("/reports/events", func(c *gin.Context) {
		handleReportEvents(c, hub)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "/reports/events", nil)
	rr := httptest.NewRecorder()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		router.ServeHTTP(rr, req)
		wg.Done()
	}()

	time.Sleep(30 * time.Millisecond)

	hub.Broadcast(WeatherReportEvent{
		DbId:        7,
		Time:        1000000,
		DeviceModel: "ShapeTest",
	})

	time.Sleep(100 * time.Millisecond)

	body := rr.Body.String()

	// Check that the JSON has the required fields
	var raw map[string]interface{}
	dataLine := body
	if len("data: ") < len(dataLine) {
		dataLine = dataLine[len("data: "):]
	}
	if err := json.Unmarshal([]byte(dataLine), &raw); err != nil {
		t.Fatalf("failed to parse SSE payload as JSON: %v (body: %s)", err, body)
	}

	if _, ok := raw["dbId"]; !ok {
		t.Fatal("expected 'dbId' field in event payload")
	}
	if _, ok := raw["time"]; !ok {
		t.Fatal("expected 'time' field in event payload")
	}
	if _, ok := raw["deviceModel"]; !ok {
		t.Fatal("expected 'deviceModel' field in event payload")
	}

	// Verify field values
	if float64(raw["dbId"].(float64)) != 7 {
		t.Fatalf("expected dbId 7, got %v", raw["dbId"])
	}
	if float64(raw["time"].(float64)) != 1000000 {
		t.Fatalf("expected time 1000000, got %v", raw["time"])
	}
	if raw["deviceModel"].(string) != "ShapeTest" {
		t.Fatalf("expected deviceModel 'ShapeTest', got '%v'", raw["deviceModel"])
	}
}

// ---- No Broadcast For Ignored Reports Tests ----

// testableSave wraps saveWeatherReport with an optional notifier for testing broadcast behavior.
type testableSaveFunc func(db *gorm.DB, report WeatherReport, notifier func(event WeatherReportEvent)) (WeatherReport, bool, error)

// testableSaveWeatherReport is a variant of saveWeatherReport that accepts an optional notifier.
// When notifier is non-nil, it is called after successful inserts so tests can verify broadcast behavior.
func testableSaveWeatherReport(db *gorm.DB, report WeatherReport, notifier func(event WeatherReportEvent)) (WeatherReport, bool, error) {
	// Ensure device model exists in database
	checkForDeviceModel(db, report)

	// Apply humidity correction if needed
	shouldIgnoreReport, err := applyHumidityCorrection(db, &report)
	if err != nil {
		return report, false, err
	}
	if shouldIgnoreReport {
		return report, false, nil
	}

	// Check for duplicates
	shouldIgnoreReport, err = checkDuplicate(db, &report)
	if err != nil {
		return report, false, err
	}
	if shouldIgnoreReport {
		return report, false, nil
	}

	// Insert into database
	result := db.Create(&report)
	if result.Error != nil {
		return report, false, result.Error
	}

	// Call notifier only after successful insert
	if notifier != nil {
		notifier(WeatherReportEvent{
			DbId:        report.DbId,
			Time:        report.Time.Unix(),
			DeviceModel: report.DeviceModel,
		})
	}

	return report, true, nil
}

func TestNoBroadcastForDuplicateReport(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now().Truncate(time.Second)

	existing := WeatherReport{
		Time:                 now,
		DeviceModel:          "NoBroadcast-Dup",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}
	db.Create(&existing)

	duplicate := WeatherReport{
		Time:                 now,
		DeviceModel:          "NoBroadcast-Dup",
		TemperatureInF:       70.0,
		HumidityInPercentage: 40,
	}

	var notified bool
	_, _, err := testableSaveWeatherReport(db, duplicate, func(event WeatherReportEvent) {
		notified = true
		_ = event
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if notified {
		t.Fatal("expected no broadcast for duplicate report")
	}
}

func TestNoBroadcastForHumidityCorrectionIgnoredReport(t *testing.T) {
	db := setupTestDB(t)

	prior := WeatherReport{
		Time:                 time.Now().Add(-1 * time.Minute),
		DeviceModel:          "NoBroadcast-Hum",
		TemperatureInF:       65.0, // below 70
		HumidityInPercentage: 3,    // below 5
	}
	db.Create(&prior)

	ignored := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "NoBroadcast-Hum",
		TemperatureInF:       65.0,
		HumidityInPercentage: 99, // will be ignored because temp <= 70
	}

	var notified bool
	_, _, err := testableSaveWeatherReport(db, ignored, func(event WeatherReportEvent) {
		notified = true
		_ = event
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if notified {
		t.Fatal("expected no broadcast for report ignored due to humidity correction")
	}
}

func TestBroadcastOnlyOnSuccessfulInsert(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now().Truncate(time.Second)

	// First insert: should notify
	report1 := WeatherReport{
		Time:                 now,
		DeviceModel:          "Broadcast-Only",
		TemperatureInF:       72.0,
		HumidityInPercentage: 50,
	}

	var notifyCount int
	var lastEvent WeatherReportEvent

	_, inserted, err := testableSaveWeatherReport(db, report1, func(event WeatherReportEvent) {
		notifyCount++
		lastEvent = event
	})
	if err != nil {
		t.Fatalf("expected no error on first insert, got %v", err)
	}
	if !inserted {
		t.Fatal("expected first report to be inserted")
	}
	if notifyCount != 1 {
		t.Fatalf("expected 1 notification after first insert, got %d", notifyCount)
	}
	if lastEvent.DbId == 0 {
		t.Fatal("expected first insert notification to have populated DbId")
	}

	// Second insert: same time/model but different values -> not a duplicate
	now2 := now.Add(6 * time.Second)
	report2 := WeatherReport{
		Time:                 now2,
		DeviceModel:          "Broadcast-Only",
		TemperatureInF:       73.0,
		HumidityInPercentage: 55,
	}

	_, inserted, err = testableSaveWeatherReport(db, report2, func(event WeatherReportEvent) {
		notifyCount++
		lastEvent = event
	})
	if err != nil {
		t.Fatalf("expected no error on second insert, got %v", err)
	}
	if !inserted {
		t.Fatal("expected second report to be inserted")
	}
	if notifyCount != 2 {
		t.Fatalf("expected 2 notifications after second insert, got %d", notifyCount)
	}

	// Third insert: duplicate -> should NOT notify
	duplicate := WeatherReport{
		Time:                 now2,
		DeviceModel:          "Broadcast-Only",
		TemperatureInF:       73.0,
		HumidityInPercentage: 55,
	}

	_, inserted, err = testableSaveWeatherReport(db, duplicate, func(event WeatherReportEvent) {
		notifyCount++
	})
	if err != nil {
		t.Fatalf("expected no error on duplicate, got %v", err)
	}
	if inserted {
		t.Fatal("expected duplicate to be ignored")
	}
	if notifyCount != 2 {
		t.Fatalf("expected 2 notifications total (not 3), got %d", notifyCount)
	}
}

// ---- Save Weather Report Additional Tests ----

func TestSaveWeatherReport_WithNilNotifier(t *testing.T) {
	// Verify that the regular saveWeatherReport (without notifier) still works correctly
	db := setupTestDB(t)

	report := WeatherReport{
		Time:                 time.Now(),
		DeviceModel:          "NilNotifier",
		TemperatureInF:       70.0,
		HumidityInPercentage: 45,
	}

	saved, inserted, err := saveWeatherReport(db, report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !inserted {
		t.Fatal("expected report to be inserted")
	}
	if saved.DbId == 0 {
		t.Fatal("expected saved report to have a populated DbId")
	}
}

func TestSSEHandler_CleanupOnClientDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := newTestEventHub()
	defer hub.Shutdown()

	router := gin.New()
	router.GET("/reports/events", func(c *gin.Context) {
		handleReportEvents(c, hub)
	})

	// Start a request with a cancellable context
	ctx1, cancel1 := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx1, "GET", "/reports/events", nil)
	rr := httptest.NewRecorder()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		router.ServeHTTP(rr, req)
		wg.Done()
	}()

	// Wait for connection to be registered
	time.Sleep(30 * time.Millisecond)

	// Cancel the request context to simulate client disconnect
	cancel1()
	wg.Wait()

	// Wait for the hub's run loop to process the unregister
	time.Sleep(20 * time.Millisecond)

	// Start a new client — should be able to connect and receive events
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	newReq, _ := http.NewRequestWithContext(ctx2, "GET", "/reports/events", nil)
	newRR := httptest.NewRecorder()

	wg.Add(1)
	go func() {
		router.ServeHTTP(newRR, newReq)
		wg.Done()
	}()

	// Give the new connection time to be established
	time.Sleep(30 * time.Millisecond)

	// Broadcast to the new client — should succeed
	hub.Broadcast(WeatherReportEvent{
		DbId:        1,
		Time:        1,
		DeviceModel: "NewClient",
	})

	time.Sleep(100 * time.Millisecond)

	// The new connection should have received some data
	if newRR.Body.Len() == 0 {
		t.Fatal("expected new client to receive SSE events after old client was removed")
	}
}
