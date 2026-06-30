package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/ws"
)

// handleListChat возвращает историю сообщений чата для Work Order.
// GET /api/v1/work-orders/{id}/chat
func (s *Server) handleListChat(w http.ResponseWriter, r *http.Request) {
	woID := chi.URLParam(r, "id")
	if woID == "" {
		RespondError(w, r, NewBadRequestError("work order id is required"))
		return
	}

	limit := parseIntParam(r.URL.Query().Get("limit"), 50)
	before := r.URL.Query().Get("before")

	messages, err := s.getChatMessages(r.Context(), woID, limit, before)
	if err != nil {
		s.logger.Error("failed to get chat messages", "wo_id", woID, "error", err)
		RespondError(w, r, NewInternalError("failed to get chat messages", err))
		return
	}

	if messages == nil {
		messages = []ws.ChatMessage{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"messages": messages,
		"limit":    limit,
	})
}

// handleSendChatMessage создаёт новое сообщение в чате Work Order.
// POST /api/v1/work-orders/{id}/chat
func (s *Server) handleSendChatMessage(w http.ResponseWriter, r *http.Request) {
	woID := chi.URLParam(r, "id")
	if woID == "" {
		RespondError(w, r, NewBadRequestError("work order id is required"))
		return
	}

	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	var req struct {
		Text        string          `json:"text"`
		MessageType string          `json:"message_type"`
		Attachments []ws.Attachment `json:"attachments,omitempty"`
		Mentions    []string        `json:"mentions,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}

	if req.Text == "" && req.MessageType == "text" {
		RespondError(w, r, NewBadRequestError("text is required for text messages"))
		return
	}
	if req.MessageType == "" {
		req.MessageType = "text"
	}
	switch req.MessageType {
	case "text", "system", "voice", "image":
	default:
		RespondError(w, r, NewBadRequestError("invalid message_type"))
		return
	}

	msg := &ws.ChatMessage{
		WOID:        woID,
		UserID:      claims.UserID,
		UserName:    claims.Username,
		Text:        req.Text,
		MessageType: req.MessageType,
		Attachments: req.Attachments,
		Mentions:    req.Mentions,
		CreatedAt:   time.Now(),
	}

	saved, err := s.saveChatMessage(r.Context(), msg)
	if err != nil {
		s.logger.Error("failed to save chat message", "error", err)
		RespondError(w, r, NewInternalError("failed to save message", err))
		return
	}

	if room := s.chatHub.GetRoom(woID); room != nil {
		room.Broadcast(ws.WSEnvelope{
			Type:    ws.MsgTypeChat,
			Payload: mustMarshalRaw(saved),
			WOID:    woID,
			Time:    time.Now(),
		})
	}

	jsonResponse(w, http.StatusCreated, saved)
}

// handleChatUpload загружает файл-вложение для чата.
// POST /api/v1/work-orders/{id}/chat/upload
func (s *Server) handleChatUpload(w http.ResponseWriter, r *http.Request) {
	woID := chi.URLParam(r, "id")
	if woID == "" {
		RespondError(w, r, NewBadRequestError("work order id is required"))
		return
	}

	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		RespondError(w, r, NewBadRequestError("file too large or invalid multipart form"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		RespondError(w, r, NewBadRequestError("file field is required"))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		RespondError(w, r, NewInternalError("failed to read file", err))
		return
	}

	ts := time.Now().UnixNano()
	storagePath := fmt.Sprintf("chat/%s/%s/%d_%s", woID, claims.UserID, ts, header.Filename)
	if err := s.saveChatFile(storagePath, data); err != nil {
		s.logger.Error("failed to save chat file", "error", err)
		RespondError(w, r, NewInternalError("failed to save file", err))
		return
	}

	attachment := ws.Attachment{
		ID:          fmt.Sprintf("att_%d", ts),
		FileName:    header.Filename,
		FileSize:    header.Size,
		MimeType:    header.Header.Get("Content-Type"),
		StoragePath: storagePath,
	}

	jsonResponse(w, http.StatusCreated, attachment)
}

// handleChatWebSocket обрабатывает WebSocket соединение для чата Work Order.
// WS /ws/chat/{wo_id}?token=...
func (s *Server) handleChatWebSocket(w http.ResponseWriter, r *http.Request) {
	woID := chi.URLParam(r, "wo_id")
	if woID == "" {
		http.Error(w, "work order id is required", http.StatusBadRequest)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	claims, err := auth.ValidateJWT(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	hasAccess, err := s.checkWorkOrderAccess(r.Context(), woID, claims.UserID)
	if err != nil {
		s.logger.Error("failed to check work order access", "error", err)
		http.Error(w, "access check failed", http.StatusInternalServerError)
		return
	}
	if !hasAccess {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	conn, err := chatWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("chat WebSocket upgrade failed", "error", err)
		return
	}

	room := s.chatHub.GetOrCreateRoom(woID)
	client := ws.NewChatClient(claims.UserID, claims.Username, claims.Role, conn, room)
	room.Register(client)

	s.logger.Info("chat WebSocket connected",
		"wo_id", woID,
		"user_id", claims.UserID,
		"username", claims.Username,
	)

	go client.WritePump()
	go client.ReadPump()
}

// ── Database Operations ────────────────────────────────────────────────────

func (s *Server) getChatMessages(ctx context.Context, woID string, limit int, before string) ([]ws.ChatMessage, error) {
	var rows pgx.Rows
	var err error

	if before != "" {
		rows, err = s.db.Pool.Query(ctx,
			`SELECT m.id, m.wo_id, m.user_id, m.user_name, m.text, m.message_type, m.created_at,
			        COALESCE(
			            (SELECT jsonb_agg(jsonb_build_object(
			                'id', a.id,
			                'file_name', a.file_name,
			                'file_size', a.file_size,
			                'mime_type', a.mime_type,
			                'storage_path', a.storage_path,
			                'thumbnail_path', a.thumbnail_path
			            )) FROM wo_chat_attachments a WHERE a.message_id = m.id),
			            '[]'::jsonb
			        ) as attachments,
			        COALESCE(
			            (SELECT jsonb_agg(mn.mentioned_user) FROM wo_chat_mentions mn WHERE mn.message_id = m.id),
			            '[]'::jsonb
			        ) as mentions
			 FROM wo_chat_messages m
			 WHERE m.wo_id = $1 AND m.id < $2
			 ORDER BY m.created_at DESC
			 LIMIT $3`,
			woID, before, limit,
		)
	} else {
		rows, err = s.db.Pool.Query(ctx,
			`SELECT m.id, m.wo_id, m.user_id, m.user_name, m.text, m.message_type, m.created_at,
			        COALESCE(
			            (SELECT jsonb_agg(jsonb_build_object(
			                'id', a.id,
			                'file_name', a.file_name,
			                'file_size', a.file_size,
			                'mime_type', a.mime_type,
			                'storage_path', a.storage_path,
			                'thumbnail_path', a.thumbnail_path
			            )) FROM wo_chat_attachments a WHERE a.message_id = m.id),
			            '[]'::jsonb
			        ) as attachments,
			        COALESCE(
			            (SELECT jsonb_agg(mn.mentioned_user) FROM wo_chat_mentions mn WHERE mn.message_id = m.id),
			            '[]'::jsonb
			        ) as mentions
			 FROM wo_chat_messages m
			 WHERE m.wo_id = $1
			 ORDER BY m.created_at DESC
			 LIMIT $2`,
			woID, limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("query chat messages: %w", err)
	}
	defer rows.Close()

	var messages []ws.ChatMessage
	for rows.Next() {
		var msg ws.ChatMessage
		var attachmentsJSON, mentionsJSON []byte

		if err := rows.Scan(
			&msg.ID, &msg.WOID, &msg.UserID, &msg.UserName,
			&msg.Text, &msg.MessageType, &msg.CreatedAt,
			&attachmentsJSON, &mentionsJSON,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan chat message: %w", err)
		}

		if len(attachmentsJSON) > 0 {
			json.Unmarshal(attachmentsJSON, &msg.Attachments)
		}
		if len(mentionsJSON) > 0 {
			json.Unmarshal(mentionsJSON, &msg.Mentions)
		}

		messages = append(messages, msg)
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (s *Server) saveChatMessage(ctx context.Context, msg *ws.ChatMessage) (*ws.ChatMessage, error) {
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	if msg.MessageType == "" {
		msg.MessageType = "text"
	}

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO wo_chat_messages (id, wo_id, user_id, user_name, text, message_type, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		msg.ID, msg.WOID, msg.UserID, msg.UserName, msg.Text, msg.MessageType, msg.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert chat message: %w", err)
	}

	for i := range msg.Attachments {
		a := &msg.Attachments[i]
		if a.ID == "" {
			a.ID = fmt.Sprintf("att_%d", time.Now().UnixNano()+int64(i))
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO wo_chat_attachments (id, message_id, file_name, file_size, mime_type, storage_path, thumbnail_path)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			a.ID, msg.ID, a.FileName, a.FileSize, a.MimeType, a.StoragePath, a.ThumbnailPath,
		)
		if err != nil {
			return nil, fmt.Errorf("insert chat attachment: %w", err)
		}
	}

	for _, mentioned := range msg.Mentions {
		mentionID := fmt.Sprintf("men_%d", time.Now().UnixNano())
		_, err = tx.Exec(ctx,
			`INSERT INTO wo_chat_mentions (id, message_id, mentioned_user) VALUES ($1, $2, $3)`,
			mentionID, msg.ID, mentioned,
		)
		if err != nil {
			return nil, fmt.Errorf("insert chat mention: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit chat message tx: %w", err)
	}

	return msg, nil
}

func (s *Server) saveChatReadReceipt(ctx context.Context, messageID, userID string) error {
	_, err := s.db.Pool.Exec(ctx,
		`INSERT INTO wo_chat_read_receipts (message_id, user_id, read_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (message_id, user_id) DO UPDATE SET read_at = NOW()`,
		messageID, userID,
	)
	return err
}

func (s *Server) saveChatReaction(ctx context.Context, messageID, userID, emoji string) error {
	_, err := s.db.Pool.Exec(ctx,
		`INSERT INTO wo_chat_reactions (message_id, user_id, reaction, created_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (message_id, user_id, reaction) DO NOTHING`,
		messageID, userID, emoji,
	)
	return err
}

// ── Access Control ─────────────────────────────────────────────────────────

func (s *Server) checkWorkOrderAccess(ctx context.Context, woID, userID string) (bool, error) {
	var count int
	err := s.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM work_orders
		 WHERE id = $1 AND (assigned_to = $2 OR $2 IN (
		     SELECT user_id FROM user_roles WHERE role IN ('admin', 'superadmin')
		 ))`,
		woID, userID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ── File Storage ───────────────────────────────────────────────────────────

func (s *Server) saveChatFile(storagePath string, data []byte) error {
	_ = storagePath
	_ = data
	return nil
}

// ── WebSocket Upgrader ─────────────────────────────────────────────────────

var chatWSUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ── JSON helper ────────────────────────────────────────────────────────────

func mustMarshalRaw(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
