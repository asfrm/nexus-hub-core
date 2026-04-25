package api

import (
	"context"
	"sync"
	"time"
)

// NexusPacket represents a message in the system
type NexusPacket struct {
	EventType string      `json:"event_type"`
	Payload   interface{} `json:"payload"`
	Log       *LogPayload `json:"log,omitempty"`
}

// LogPayload contains log message data
type LogPayload struct {
	Module    string    `json:"module"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	Timestamp time.Time `json:"timestamp"`
}

// Hub manages pub-sub broadcasting to multiple subscribers
type Hub struct {
	subscribers  map[chan NexusPacket]bool
	internalChan chan NexusPacket
	mu           sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		subscribers:  make(map[chan NexusPacket]bool),
		internalChan: make(chan NexusPacket, 100),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		h.Close()
	}()
}

// Subscribe creates a new subscriber channel
func (h *Hub) Subscribe() <-chan NexusPacket {
	ch := make(chan NexusPacket, 100)
	h.mu.Lock()
	h.subscribers[ch] = true
	h.mu.Unlock()
	return ch
}

// Publish sends a packet to all subscribers
func (h *Hub) Publish(packet NexusPacket) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.subscribers {
		select {
		case ch <- packet:
		default:
			// Skip slow subscribers to avoid blocking
		}
	}
}

// GetInternalChan returns the internal channel for sending packets
func (h *Hub) GetInternalChan() <-chan NexusPacket {
	return h.internalChan
}

// Send publishes a packet to the internal channel and all subscribers
func (h *Hub) Send(packet NexusPacket) {
	h.internalChan <- packet
	h.Publish(packet)
}

// Close shuts down the hub and closes all subscriber channels
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for ch := range h.subscribers {
		close(ch)
		delete(h.subscribers, ch)
	}
	close(h.internalChan)
}
