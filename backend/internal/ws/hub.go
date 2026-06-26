package ws

import (
	"log/slog"
	"sync"
	"time"
)

// Hub manages WebSocket client connections and broadcasts messages to all connected clients.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop for handling client registration, unregistration, and message broadcasting.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client is slow, disconnect it
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// GetClientCount returns the number of connected clients.
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ─── Presence Hub (P3-3.3: Real-time Collaboration) ─────────────────────────

// PresenceInfo represents a user's presence on a resource (e.g. work order).
type PresenceInfo struct {
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	Resource  string    `json:"resource"` // "work_order:wo-123"
	Action    string    `json:"action"`   // "viewing", "editing"
	UpdatedAt time.Time `json:"updated_at"`
}

// PresenceHub tracks which users are currently viewing or editing a resource.
// Used for real-time collaboration features:
//   - WebSocket presence indicators
//   - "User is editing this WO" indicator
//   - Conflict warning on concurrent edit
type PresenceHub struct {
	mu        sync.RWMutex
	presences map[string]*PresenceInfo // key: "user_id:resource"
	logger    *slog.Logger
	timeout   time.Duration
}

// NewPresenceHub creates a new PresenceHub with the given logger.
func NewPresenceHub(logger *slog.Logger) *PresenceHub {
	return &PresenceHub{
		presences: make(map[string]*PresenceInfo),
		logger:    logger.With("component", "presence-hub"),
		timeout:   30 * time.Second,
	}
}

// SetPresence records or refreshes a user's presence on a resource.
func (h *PresenceHub) SetPresence(userID, userName, resource, action string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	key := userID + ":" + resource
	h.presences[key] = &PresenceInfo{
		UserID:    userID,
		UserName:  userName,
		Resource:  resource,
		Action:    action,
		UpdatedAt: time.Now(),
	}
}

// RemovePresence removes a user's presence from a resource.
func (h *PresenceHub) RemovePresence(userID, resource string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.presences, userID+":"+resource)
}

// GetResourcePresences returns all active presences for a given resource
// that have not timed out.
func (h *PresenceHub) GetResourcePresences(resource string) []*PresenceInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []*PresenceInfo
	now := time.Now()
	for _, p := range h.presences {
		if p.Resource == resource && now.Sub(p.UpdatedAt) < h.timeout {
			result = append(result, p)
		}
	}
	return result
}

// Cleanup removes all expired presences that have exceeded the timeout.
func (h *PresenceHub) Cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	for key, p := range h.presences {
		if now.Sub(p.UpdatedAt) >= h.timeout {
			delete(h.presences, key)
		}
	}
}
