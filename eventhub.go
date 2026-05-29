package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
)

// EventHub broadcasts weather report events to SSE subscribers
type EventHub struct {
	mu        sync.RWMutex
	clients   map[uint64]chan string
	nextID    uint64
	done      chan struct{}
	broadcast chan WeatherReportEvent
}

// NewEventHub creates and starts a new EventHub
func NewEventHub() *EventHub {
	hub := &EventHub{
		clients:   make(map[uint64]chan string),
		done:      make(chan struct{}),
		broadcast: make(chan WeatherReportEvent, 256),
	}
	go hub.run()
	return hub
}

// run is the main event loop that forwards events to all connected clients
func (hub *EventHub) run() {
	for {
		select {
		case <-hub.done:
			return
		case event := <-hub.broadcast:
			hub.broadcastToClients(event)
		}
	}
}

// broadcastToClients sends an event to all connected clients concurrently
func (hub *EventHub) broadcastToClients(event WeatherReportEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("EventHub: failed to marshal event: %v\n", err)
		return
	}
	message := fmt.Sprintf("data: %s\n\n", string(data))

	// Collect live client IDs to avoid holding the lock during writes
	var clientIDs []uint64
	hub.mu.RLock()
	clientIDs = make([]uint64, 0, len(hub.clients))
	for id := range hub.clients {
		clientIDs = append(clientIDs, id)
	}
	hub.mu.RUnlock()

	for _, id := range clientIDs {
		hub.mu.RLock()
		ch, ok := hub.clients[id]
		hub.mu.RUnlock()
		if !ok {
			continue
		}

		// Non-blocking send: if client buffer is full, remove them
		select {
		case ch <- message:
			// sent successfully
		default:
			// client is slow or disconnected, remove it
			hub.unregister(id)
		}
	}
}

// register adds a new client and returns its channel
func (hub *EventHub) register() (uint64, chan string) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	id := atomic.AddUint64(&hub.nextID, 1)
	ch := make(chan string, 256)
	hub.clients[id] = ch
	return id, ch
}

// unregister removes a client by ID
func (hub *EventHub) unregister(id uint64) {
	hub.mu.Lock()
	if ch, ok := hub.clients[id]; ok {
		close(ch)
		delete(hub.clients, id)
	}
	hub.mu.Unlock()
}

// Shutdown stops the event hub and closes all client connections
func (hub *EventHub) Shutdown() {
	close(hub.done)
	hub.mu.Lock()
	for id, ch := range hub.clients {
		close(ch)
		delete(hub.clients, id)
	}
	hub.mu.Unlock()
}

// Broadcast sends an event to all subscribers
func (hub *EventHub) Broadcast(event WeatherReportEvent) {
	select {
	case hub.broadcast <- event:
	default:
		// broadcast channel is full, drop the event to avoid blocking rtl_433
	}
}
