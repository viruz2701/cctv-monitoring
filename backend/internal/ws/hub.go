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

// GetAllPresences returns all active presences across all resources.
func (h *PresenceHub) GetAllPresences() []*PresenceInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []*PresenceInfo
	now := time.Now()
	for _, p := range h.presences {
		if now.Sub(p.UpdatedAt) < h.timeout {
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

// ─── Cursor Sharing ────────────────────────────────────────────────────────

// CursorPosition represents a user's cursor position on a resource.
type CursorPosition struct {
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	Resource  string    `json:"resource"`
	Field     string    `json:"field"`  // which field is focused
	Offset    int       `json:"offset"` // cursor offset in text
	Line      int       `json:"line"`   // line number
	Col       int       `json:"col"`    // column number
	UpdatedAt time.Time `json:"updated_at"`
}

// CursorHub tracks real-time cursor positions for collaboration.
type CursorHub struct {
	mu      sync.RWMutex
	cursors map[string]*CursorPosition // key: "user_id:resource"
	logger  *slog.Logger
	timeout time.Duration
}

// NewCursorHub creates a new CursorHub.
func NewCursorHub(logger *slog.Logger) *CursorHub {
	return &CursorHub{
		cursors: make(map[string]*CursorPosition),
		logger:  logger.With("component", "cursor-hub"),
		timeout: 10 * time.Second,
	}
}

// SetCursor updates a user's cursor position on a resource.
func (h *CursorHub) SetCursor(userID, userName, resource, field string, offset, line, col int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	key := userID + ":" + resource
	h.cursors[key] = &CursorPosition{
		UserID:    userID,
		UserName:  userName,
		Resource:  resource,
		Field:     field,
		Offset:    offset,
		Line:      line,
		Col:       col,
		UpdatedAt: time.Now(),
	}
}

// RemoveCursor removes a user's cursor from a resource.
func (h *CursorHub) RemoveCursor(userID, resource string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.cursors, userID+":"+resource)
}

// GetResourceCursors returns all active cursors for a resource.
func (h *CursorHub) GetResourceCursors(resource string) []*CursorPosition {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []*CursorPosition
	now := time.Now()
	for _, c := range h.cursors {
		if c.Resource == resource && now.Sub(c.UpdatedAt) < h.timeout {
			result = append(result, c)
		}
	}
	return result
}

// Cleanup removes all expired cursors.
func (h *CursorHub) Cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	for key, c := range h.cursors {
		if now.Sub(c.UpdatedAt) >= h.timeout {
			delete(h.cursors, key)
		}
	}
}

// ─── Collaboration Hub (composite) ─────────────────────────────────────────

// CollaborationHub combines presence and cursor tracking for real-time collaboration.
// It also manages room-based subscriptions so users only receive relevant updates.
type CollaborationHub struct {
	presence *PresenceHub
	cursor   *CursorHub
	rooms    map[string]map[string]bool // room -> set of userIDs
	mu       sync.RWMutex
	logger   *slog.Logger
}

// NewCollaborationHub creates a new CollaborationHub.
func NewCollaborationHub(logger *slog.Logger) *CollaborationHub {
	return &CollaborationHub{
		presence: NewPresenceHub(logger),
		cursor:   NewCursorHub(logger),
		rooms:    make(map[string]map[string]bool),
		logger:   logger.With("component", "collab-hub"),
	}
}

// JoinRoom adds a user to a room.
func (h *CollaborationHub) JoinRoom(room, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[string]bool)
	}
	h.rooms[room][userID] = true
}

// LeaveRoom removes a user from a room.
func (h *CollaborationHub) LeaveRoom(room, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[room] != nil {
		delete(h.rooms[room], userID)
		if len(h.rooms[room]) == 0 {
			delete(h.rooms, room)
		}
	}
}

// GetRoomUsers returns all users in a room.
func (h *CollaborationHub) GetRoomUsers(room string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var result []string
	for userID := range h.rooms[room] {
		result = append(result, userID)
	}
	return result
}

// SetPresence records presence and notifies the room.
func (h *CollaborationHub) SetPresence(userID, userName, resource, action string) {
	h.presence.SetPresence(userID, userName, resource, action)
}

// RemovePresence removes presence and notifies the room.
func (h *CollaborationHub) RemovePresence(userID, resource string) {
	h.presence.RemovePresence(userID, resource)
}

// GetResourcePresences returns all presences for a resource.
func (h *CollaborationHub) GetResourcePresences(resource string) []*PresenceInfo {
	return h.presence.GetResourcePresences(resource)
}

// SetCursor updates cursor position.
func (h *CollaborationHub) SetCursor(userID, userName, resource, field string, offset, line, col int) {
	h.cursor.SetCursor(userID, userName, resource, field, offset, line, col)
}

// RemoveCursor removes cursor.
func (h *CollaborationHub) RemoveCursor(userID, resource string) {
	h.cursor.RemoveCursor(userID, resource)
}

// GetResourceCursors returns all cursors for a resource.
func (h *CollaborationHub) GetResourceCursors(resource string) []*CursorPosition {
	return h.cursor.GetResourceCursors(resource)
}

// Cleanup removes all expired presences and cursors.
func (h *CollaborationHub) Cleanup() {
	h.presence.Cleanup()
	h.cursor.Cleanup()
}
