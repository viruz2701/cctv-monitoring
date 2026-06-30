// Package ws — WebSocket chat per Work Order with @mentions, read receipts, reactions.
//
// P2-CHAT: Real-Time Chat per Work Order
//
// Архитектура:
//   - ChatHub — глобальный менеджер комнат (map[woID]*ChatRoom)
//   - ChatRoom — комната для одного Work Order (map[userID]*ChatClient)
//   - ChatClient — обёртка над WebSocket.Client с user info
//
// Compliance:
//   - ISO 27001 A.12.4 (Audit trail — все сообщения логируются)
//   - ISO 27001 A.9.2 (Access control — только участники WO)
//   - IEC 62443-3-3 SR 2.1 (Account management)
//   - OWASP ASVS V3 (Session management)
//   - Приказ ОАЦ №66 п. 7.18.3 (Аудит)
package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ── Message Types ──────────────────────────────────────────────────────────

const (
	MsgTypeChat        = "chat"
	MsgTypeSystem      = "system"
	MsgTypeTyping      = "typing"
	MsgTypeReadReceipt = "read_receipt"
	MsgTypeReaction    = "reaction"
	MsgTypePresence    = "presence"
	MsgTypeError       = "error"
)

// ChatMessage represents a chat message exchanged via WebSocket.
type ChatMessage struct {
	ID          string       `json:"id"`
	WOID        string       `json:"wo_id"`
	UserID      string       `json:"user_id"`
	UserName    string       `json:"user_name"`
	Text        string       `json:"text"`
	MessageType string       `json:"message_type"` // text, system, voice, image
	Attachments []Attachment `json:"attachments,omitempty"`
	Mentions    []string     `json:"mentions,omitempty"`
	Reaction    string       `json:"reaction,omitempty"`
	ReadBy      []string     `json:"read_by,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

// Attachment represents a file attached to a chat message.
type Attachment struct {
	ID            string `json:"id"`
	FileName      string `json:"file_name"`
	FileSize      int64  `json:"file_size"`
	MimeType      string `json:"mime_type"`
	StoragePath   string `json:"storage_path,omitempty"`
	ThumbnailPath string `json:"thumbnail_path,omitempty"`
}

// WSEnvelope is the generic WebSocket message envelope.
type WSEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	WOID    string          `json:"wo_id"`
	UserID  string          `json:"user_id,omitempty"`
	Time    time.Time       `json:"time"`
}

// ── ChatClient ─────────────────────────────────────────────────────────────

// ChatClient represents a single WebSocket connection in a chat room.
type ChatClient struct {
	UserID   string
	UserName string
	Role     string
	conn     *websocket.Conn
	send     chan []byte
	room     *ChatRoom
	mu       sync.Mutex
	closed   bool
}

// NewChatClient creates a new ChatClient.
func NewChatClient(userID, userName, role string, conn *websocket.Conn, room *ChatRoom) *ChatClient {
	return &ChatClient{
		UserID:   userID,
		UserName: userName,
		Role:     role,
		conn:     conn,
		send:     make(chan []byte, 256),
		room:     room,
	}
}

// ReadPump reads messages from WebSocket connection and publishes to room.
func (c *ChatClient) ReadPump() {
	defer func() {
		c.room.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(65536) // 64KB max message
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("chat ws read error", "user_id", c.UserID, "error", err)
			}
			break
		}

		var envelope WSEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			c.sendError("invalid message format")
			continue
		}

		envelope.UserID = c.UserID
		envelope.Time = time.Now()

		switch envelope.Type {
		case MsgTypeChat:
			c.room.HandleChatMessage(c, envelope.Payload)
		case MsgTypeTyping:
			c.room.BroadcastTyping(c)
		case MsgTypeReadReceipt:
			c.room.HandleReadReceipt(c, envelope.Payload)
		case MsgTypeReaction:
			c.room.HandleReaction(c, envelope.Payload)
		default:
			c.sendError("unknown message type: " + envelope.Type)
		}
	}
}

// WritePump writes messages from the send channel to WebSocket connection.
func (c *ChatClient) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send sends a JSON-encoded message to the client.
func (c *ChatClient) Send(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	select {
	case c.send <- data:
	default:
		c.room.Unregister(c)
	}
	return nil
}

func (c *ChatClient) sendError(msg string) {
	_ = c.Send(WSEnvelope{
		Type: MsgTypeError,
		Time: time.Now(),
	})
}

// Close marks the client as closed and closes the send channel.
func (c *ChatClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

// ── ChatRoom ───────────────────────────────────────────────────────────────

// ChatRoom represents a single chat room for one Work Order.
type ChatRoom struct {
	WOID    string
	clients map[string]*ChatClient // userID -> client
	logger  *slog.Logger
	mu      sync.RWMutex
	hub     *ChatHub
}

// NewChatRoom creates a new ChatRoom for the given Work Order.
func NewChatRoom(woID string, hub *ChatHub, logger *slog.Logger) *ChatRoom {
	return &ChatRoom{
		WOID:    woID,
		clients: make(map[string]*ChatClient),
		logger:  logger.With("component", "chat-room", "wo_id", woID),
		hub:     hub,
	}
}

// Register adds a client to the room and broadcasts join event.
func (r *ChatRoom) Register(client *ChatClient) {
	r.mu.Lock()
	r.clients[client.UserID] = client
	userCount := len(r.clients)
	r.mu.Unlock()

	r.logger.Info("client joined chat room",
		"user_id", client.UserID,
		"user_name", client.UserName,
		"total_clients", userCount,
	)

	// Broadcast presence update
	r.broadcastPresence()
}

// Unregister removes a client from the room and broadcasts leave event.
func (r *ChatRoom) Unregister(client *ChatClient) {
	r.mu.Lock()
	if _, ok := r.clients[client.UserID]; ok {
		delete(r.clients, client.UserID)
		client.Close()
	}
	userCount := len(r.clients)
	r.mu.Unlock()

	r.logger.Info("client left chat room",
		"user_id", client.UserID,
		"total_clients", userCount,
	)

	r.broadcastPresence()

	// If room is empty, remove it from hub
	if userCount == 0 {
		r.hub.RemoveRoom(r.WOID)
	}
}

// GetActiveUsers returns list of user IDs currently in the room.
func (r *ChatRoom) GetActiveUsers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	users := make([]string, 0, len(r.clients))
	for userID := range r.clients {
		users = append(users, userID)
	}
	return users
}

// Broadcast sends a message to all clients in the room.
func (r *ChatRoom) Broadcast(msg WSEnvelope) {
	data, err := json.Marshal(msg)
	if err != nil {
		r.logger.Error("failed to marshal broadcast", "error", err)
		return
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, client := range r.clients {
		select {
		case client.send <- data:
		default:
			r.logger.Warn("client send buffer full, unregistering",
				"user_id", client.UserID,
			)
			go r.Unregister(client)
		}
	}
}

// BroadcastExcept sends a message to all clients except the sender.
func (r *ChatRoom) BroadcastExcept(senderID string, msg WSEnvelope) {
	data, err := json.Marshal(msg)
	if err != nil {
		r.logger.Error("failed to marshal broadcast", "error", err)
		return
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for userID, client := range r.clients {
		if userID == senderID {
			continue
		}
		select {
		case client.send <- data:
		default:
			go r.Unregister(client)
		}
	}
}

// HandleChatMessage processes an incoming chat message from a client.
func (r *ChatRoom) HandleChatMessage(sender *ChatClient, payload json.RawMessage) {
	var msg ChatMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		sender.sendError("invalid chat message format")
		return
	}

	// Inject sender info (server-authoritative)
	msg.UserID = sender.UserID
	msg.UserName = sender.UserName
	msg.WOID = r.WOID
	msg.CreatedAt = time.Now()

	// Store in database via hub's store callback
	if r.hub.onMessage != nil {
		saved, err := r.hub.onMessage(&msg)
		if err != nil {
			r.logger.Error("failed to store chat message", "error", err)
			sender.sendError("failed to save message")
			return
		}
		msg.ID = saved.ID
	}

	r.Broadcast(WSEnvelope{
		Type:    MsgTypeChat,
		Payload: mustMarshal(msg),
		WOID:    r.WOID,
		Time:    time.Now(),
	})

	// Send push notifications for @mentions
	if len(msg.Mentions) > 0 && r.hub.onNotify != nil {
		r.hub.onNotify(&msg)
	}
}

// HandleReadReceipt processes a read receipt from a client.
func (r *ChatRoom) HandleReadReceipt(client *ChatClient, payload json.RawMessage) {
	var receipt struct {
		MessageID string `json:"message_id"`
	}
	if err := json.Unmarshal(payload, &receipt); err != nil {
		client.sendError("invalid read receipt format")
		return
	}

	// Store read receipt via hub
	if r.hub.onReadReceipt != nil {
		r.hub.onReadReceipt(receipt.MessageID, client.UserID)
	}

	r.Broadcast(WSEnvelope{
		Type: MsgTypeReadReceipt,
		Payload: mustMarshal(map[string]string{
			"message_id": receipt.MessageID,
			"user_id":    client.UserID,
		}),
		WOID: r.WOID,
		Time: time.Now(),
	})
}

// HandleReaction processes a reaction to a message.
func (r *ChatRoom) HandleReaction(client *ChatClient, payload json.RawMessage) {
	var reaction struct {
		MessageID string `json:"message_id"`
		Emoji     string `json:"emoji"`
	}
	if err := json.Unmarshal(payload, &reaction); err != nil {
		client.sendError("invalid reaction format")
		return
	}

	// Store reaction via hub
	if r.hub.onReaction != nil {
		r.hub.onReaction(reaction.MessageID, client.UserID, reaction.Emoji)
	}

	r.Broadcast(WSEnvelope{
		Type: MsgTypeReaction,
		Payload: mustMarshal(map[string]string{
			"message_id": reaction.MessageID,
			"user_id":    client.UserID,
			"emoji":      reaction.Emoji,
		}),
		WOID: r.WOID,
		Time: time.Now(),
	})
}

// BroadcastTyping sends a typing indicator to other clients in the room.
func (r *ChatRoom) BroadcastTyping(client *ChatClient) {
	r.BroadcastExcept(client.UserID, WSEnvelope{
		Type: MsgTypeTyping,
		Payload: mustMarshal(map[string]string{
			"user_id":   client.UserID,
			"user_name": client.UserName,
		}),
		WOID: r.WOID,
		Time: time.Now(),
	})
}

func (r *ChatRoom) broadcastPresence() {
	r.mu.RLock()
	users := make([]map[string]string, 0, len(r.clients))
	for _, client := range r.clients {
		users = append(users, map[string]string{
			"user_id":   client.UserID,
			"user_name": client.UserName,
		})
	}
	r.mu.RUnlock()

	r.Broadcast(WSEnvelope{
		Type:    MsgTypePresence,
		Payload: mustMarshal(map[string]interface{}{"users": users}),
		WOID:    r.WOID,
		Time:    time.Now(),
	})
}

// ── ChatHub ────────────────────────────────────────────────────────────────

// MessageStore is the callback for persisting chat messages.
type MessageStore func(msg *ChatMessage) (*ChatMessage, error)

// ReadReceiptStore is the callback for persisting read receipts.
type ReadReceiptStore func(messageID, userID string) error

// ReactionStore is the callback for persisting reactions.
type ReactionStore func(messageID, userID, emoji string) error

// NotifyCallback is the callback for sending push notifications.
type NotifyCallback func(msg *ChatMessage)

// ChatHub manages chat rooms for all Work Orders.
type ChatHub struct {
	rooms  map[string]*ChatRoom // woID -> room
	logger *slog.Logger
	mu     sync.RWMutex
	hub    *Hub // reference to the main WS hub for shared resources

	// Store callbacks (injected by api layer)
	onMessage     MessageStore
	onReadReceipt ReadReceiptStore
	onReaction    ReactionStore
	onNotify      NotifyCallback
}

// NewChatHub creates a new ChatHub.
func NewChatHub(logger *slog.Logger) *ChatHub {
	return &ChatHub{
		rooms:  make(map[string]*ChatRoom),
		logger: logger.With("component", "chat-hub"),
	}
}

// SetStoreCallbacks injects database storage callbacks.
func (h *ChatHub) SetStoreCallbacks(
	onMessage MessageStore,
	onReadReceipt ReadReceiptStore,
	onReaction ReactionStore,
	onNotify NotifyCallback,
) {
	h.onMessage = onMessage
	h.onReadReceipt = onReadReceipt
	h.onReaction = onReaction
	h.onNotify = onNotify
}

// GetOrCreateRoom returns the chat room for a Work Order, creating it if needed.
func (h *ChatHub) GetOrCreateRoom(woID string) *ChatRoom {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.rooms[woID]; ok {
		return room
	}

	room := NewChatRoom(woID, h, h.logger)
	h.rooms[woID] = room
	return room
}

// GetRoom returns the chat room for a Work Order without creating one.
func (h *ChatHub) GetRoom(woID string) *ChatRoom {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rooms[woID]
}

// RemoveRoom removes an empty room from the hub.
func (h *ChatHub) RemoveRoom(woID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, woID)
	h.logger.Debug("chat room removed", "wo_id", woID)
}

// RoomCount returns the number of active rooms.
func (h *ChatHub) RoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic("ws chat: marshal failed: " + err.Error())
	}
	return data
}
