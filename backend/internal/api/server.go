package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/pquerna/otp/totp"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/config"
	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/sip"
	"gb-telemetry-collector/internal/state"
	"gb-telemetry-collector/internal/telegram"
	"gb-telemetry-collector/internal/ws"
)

type Server struct {
	stateManager state.DeviceStateManager
	logger       *slog.Logger
	db           *db.DB
	httpServer   *http.Server
	imagesDir    string
	config       *config.Config
	sipHandler   *sip.SIPHandler
	wsHub        *ws.Hub
	telegramBot  *telegram.Bot

	// P2P gateway integration
	p2pGatewayURL string
	p2pAPIKey     string
	httpClient    *http.Client
}

func NewServer(addr string, stateMgr state.DeviceStateManager, logger *slog.Logger, database *db.DB, imagesDir string, cfg *config.Config, sipHandler *sip.SIPHandler) *Server {
	r := chi.NewRouter()

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	s := &Server{
		stateManager:  stateMgr,
		logger:        logger,
		db:            database,
		imagesDir:     imagesDir,
		config:        cfg,
		sipHandler:    sipHandler,
		wsHub:         ws.NewHub(),
		p2pGatewayURL: cfg.P2PGatewayURL,
		p2pAPIKey:     cfg.P2PAPIKey,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
	go s.wsHub.Run()

	// Публичные маршруты (без JWT)
	r.Post("/api/v1/auth/login", s.handleLogin)
	if cfg.HTTPXMLEnabled {
		r.Post("/api/v1/external/alarm/xml", s.handleExternalAlarmXML)
	}
	if cfg.VigiEnabled {
		r.Post("/api/v1/external/alarm/vigi", s.handleExternalAlarmVigi)
	}
	// P2P alarm endpoint (protected by API key, not JWT)
	r.Post("/api/v1/external/alarm/p2p", s.handleP2PAlarm)

	// Защищённые маршруты (требуют JWT)
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware)
		r.Get("/api/v1/users/me", s.handleCurrentUser)
		r.Get("/api/v1/devices", s.listDevices)
		r.Get("/api/v1/devices/{id}", s.getDevice)
		r.Get("/api/v1/devices/{id}/status", s.getDeviceStatus)
		r.Post("/api/v1/external/alarm", s.handleExternalAlarm)
		r.Get("/api/v1/analytics/predictions", s.getPredictions)
		r.Get("/api/v1/logs/search", s.searchLogs)

		// Password management
		r.Put("/api/v1/users/me/password", s.changeMyPassword)
		r.Put("/api/v1/users/{id}/reset-password", s.resetUserPassword) // Admin only

		// Settings (Services)
		r.Get("/api/v1/settings/services", s.getServicesSettings)
		r.Put("/api/v1/settings/services", s.updateServicesSettings)

		// --- User Management (Admin Only) ---
		r.Get("/api/v1/users", s.listUsers)
		r.Post("/api/v1/users", s.createUser)
		r.Put("/api/v1/users/{id}", s.updateUser)
		r.Delete("/api/v1/users/{id}", s.deleteUser)

		// Изображения
		r.Get("/api/v1/images/{filename}", s.getImage)
		r.Get("/api/v1/images/device/{deviceId}", s.listDeviceImages)

		// P2P management endpoints
		r.Get("/api/v1/p2p/devices", s.listP2PDevices)
		r.Post("/api/v1/p2p/devices", s.registerP2PDevice)
		r.Get("/api/v1/p2p/status/{id}", s.getP2PDeviceStatus)
		r.Post("/api/v1/p2p/command/{id}", s.sendP2PCommand)
		r.Get("/api/v1/p2p/snapshot/{id}", s.getP2PSnapshot)

		// GB28181 API endpoints
		r.Post("/api/v1/gb28181/catalog/{id}", s.requestCatalog)
		r.Post("/api/v1/gb28181/ptz/{id}", s.sendPTZCommand)

		// WebSocket endpoint for real-time alarms
		r.Get("/api/v1/ws/alarms", s.handleWebSocket)

		// Session management endpoints
		r.Get("/api/v1/sessions", s.getUserSessions)
		r.Delete("/api/v1/sessions/{id}", s.revokeSession)
		r.Post("/api/v1/sessions/revoke-all", s.revokeAllOtherSessions)

		// 2FA endpoints
		r.Post("/api/v1/users/me/2fa/setup", s.handle2FASetup)
		r.Post("/api/v1/users/me/2fa/verify", s.handle2FAVerify)
		r.Post("/api/v1/users/me/2fa/disable", s.handle2FADisable)

		// Telegram endpoints
		r.Post("/api/v1/users/me/telegram/generate-link", s.handleTelegramGenerateLink)
		r.Post("/api/v1/users/me/telegram/settings", s.handleTelegramUpdateSettings)
		r.Get("/api/v1/users/me/telegram/status", s.handleTelegramStatus)

		// API Key Management (Admin only)
		r.Get("/api/v1/api-keys", s.handleListAPIKeys)
		r.Post("/api/v1/api-keys", s.handleCreateAPIKey)
		r.Delete("/api/v1/api-keys/{id}", s.handleRevokeAPIKey)

		// ═══════════════════════════════════════════════════════════════
		// CMMS Routes (Maintenance Schedules, Work Orders, Spare Parts)
		// ═══════════════════════════════════════════════════════════════

		// Maintenance Schedules
		r.Get("/api/v1/maintenance/schedules", s.listMaintenanceSchedules)
		r.Post("/api/v1/maintenance/schedules", s.createMaintenanceSchedule)
		r.Get("/api/v1/maintenance/schedules/due", s.getDueSchedules)
		r.Get("/api/v1/maintenance/schedules/{id}", s.getMaintenanceSchedule)
		r.Put("/api/v1/maintenance/schedules/{id}", s.updateMaintenanceSchedule)
		r.Delete("/api/v1/maintenance/schedules/{id}", s.deleteMaintenanceSchedule)
		r.Post("/api/v1/maintenance/schedules/{id}/complete", s.completeMaintenanceSchedule)

		// Work Orders
		r.Get("/api/v1/work-orders", s.listWorkOrders)
		r.Post("/api/v1/work-orders", s.createWorkOrder)
		r.Get("/api/v1/work-orders/{id}", s.getWorkOrder)
		r.Put("/api/v1/work-orders/{id}", s.updateWorkOrder)
		r.Delete("/api/v1/work-orders/{id}", s.deleteWorkOrder)
		r.Post("/api/v1/work-orders/{id}/assign", s.assignWorkOrder)
		r.Post("/api/v1/work-orders/{id}/start", s.startWorkOrder)
		r.Post("/api/v1/work-orders/{id}/complete", s.completeWorkOrder)
		r.Post("/api/v1/work-orders/{id}/cancel", s.cancelWorkOrder)
		r.Post("/api/v1/work-orders/{id}/photos", s.uploadWorkOrderPhotos)
		r.Post("/api/v1/work-orders/{id}/parts", s.addWorkOrderParts)

		// Spare Parts
		r.Get("/api/v1/spare-parts", s.listSpareParts)
		r.Post("/api/v1/spare-parts", s.createSparePart)
		r.Get("/api/v1/spare-parts/low-stock", s.getLowStockParts)
		r.Get("/api/v1/spare-parts/{id}", s.getSparePart)
		r.Put("/api/v1/spare-parts/{id}", s.updateSparePart)
		r.Delete("/api/v1/spare-parts/{id}", s.deleteSparePart)
		r.Post("/api/v1/spare-parts/{id}/adjust", s.adjustSparePartStock)

		// Technician Management
		r.Get("/api/v1/technicians/workload", s.getAllTechnicianWorkloads)
		r.Get("/api/v1/technicians/{id}/workload", s.getTechnicianWorkload)
		r.Put("/api/v1/technicians/{id}/skills", s.updateTechnicianSkills)

		// Technician Site Assignments (закрепление техников за объектами)
		r.Get("/api/v1/technician-assignments", s.listTechnicianSiteAssignments)
		r.Post("/api/v1/technician-assignments", s.createTechnicianSiteAssignment)
		r.Put("/api/v1/technician-assignments/{id}", s.updateTechnicianSiteAssignment)
		r.Delete("/api/v1/technician-assignments/{id}", s.deleteTechnicianSiteAssignment)

		// SLA & Reports
		r.Get("/api/v1/sla/config", s.getSLAConfig)
		r.Put("/api/v1/sla/config/{priority}", s.updateSLAConfig)
		r.Get("/api/v1/reports/maintenance", s.getMaintenanceReport)
		r.Get("/api/v1/reports/sla-compliance", s.getSLAComplianceReport)
	})

	// External endpoints with API key auth
	r.Group(func(r chi.Router) {
		r.Use(s.APIKeyMiddleware)
		r.Post("/api/v1/external/alarm", s.handleExternalAlarm)
	})

	// Public 2FA login endpoint
	r.Post("/api/v1/auth/login/2fa", s.handleLogin2FA)

	// Public Telegram login endpoints
	r.Post("/api/v1/auth/telegram/request-code", s.handleTelegramRequestCode)
	r.Post("/api/v1/auth/telegram/verify", s.handleTelegramVerify)

	// Public password reset endpoints
	r.Post("/api/v1/auth/forgot-password", s.handleForgotPassword)
	r.Post("/api/v1/auth/reset-password", s.handleResetPasswordWithToken)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: r,
	}
	return s
}

func (s *Server) requestCatalog(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")

	if err := s.sipHandler.RequestCatalog(deviceID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":    "catalog_requested",
		"device_id": deviceID,
	})
}

func (s *Server) sendPTZCommand(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")

	var req struct {
		Command string `json:"command"`
		Speed   int    `json:"speed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Speed == 0 {
		req.Speed = 128
	}

	cmd := sip.PTZCommand{
		Action: req.Command,
		Speed:  req.Speed,
	}

	if err := s.sipHandler.SendPTZCommand(deviceID, cmd); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":    "command_sent",
		"device_id": deviceID,
		"command":   req.Command,
	})
}

func (s *Server) Start() error {
	s.logger.Info("API server started", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop() error {
	return s.httpServer.Close()
}

// SetTelegramBot sets the Telegram bot instance for the server
func (s *Server) SetTelegramBot(bot *telegram.Bot) {
	s.telegramBot = bot
}

// ---------- Аутентификация ----------

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	// Ищем пользователя по username или email
	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		// Пробуем найти по email
		user, err = s.db.GetUserByEmail(req.Username)
		if err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
	}
	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// If 2FA is enabled, return a temporary session token instead of the main JWT
	if user.TOTPEnabled {
		tempToken, err := auth.GenerateTempToken(user.ID, user.Username, user.Role)
		if err != nil {
			s.logger.Error("failed to generate temp token", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusAccepted, map[string]interface{}{
			"requires_2fa":  true,
			"session_token": tempToken,
		})
		return
	}

	token, err := auth.GenerateJWT(user.ID, user.Username, user.Role)
	if err != nil {
		s.logger.Error("failed to generate JWT", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	user.PasswordHash = ""
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

// handleLogin2FA handles the second step of 2FA login.
func (s *Server) handleLogin2FA(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionToken string `json:"session_token"`
		Code         string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	claims, err := auth.ValidateTempToken(req.SessionToken)
	if err != nil {
		http.Error(w, "invalid or expired session", http.StatusUnauthorized)
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	if !user.TOTPEnabled || user.TOTPSecret == "" {
		http.Error(w, "2FA not enabled", http.StatusBadRequest)
		return
	}

	if !totp.Validate(req.Code, user.TOTPSecret) {
		http.Error(w, "invalid 2FA code", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateJWT(user.ID, user.Username, user.Role)
	if err != nil {
		s.logger.Error("failed to generate JWT", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	user.PasswordHash = ""
	user.TOTPSecret = ""
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

// handle2FASetup generates a TOTP secret and returns the provisioning URI for QR code.
func (s *Server) handle2FASetup(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role != "admin" {
		http.Error(w, "forbidden: admin only", http.StatusForbidden)
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if user.TOTPEnabled {
		http.Error(w, "2FA already enabled", http.StatusBadRequest)
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "CCTV Monitor",
		AccountName: user.Username,
	})
	if err != nil {
		s.logger.Error("failed to generate TOTP key", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.db.UpdateTOTPSecret(user.ID, key.Secret()); err != nil {
		s.logger.Error("failed to save TOTP secret", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"secret":   key.Secret(),
		"uri":      key.URL(),
		"qr_image": nil, // Frontend will generate QR from URI
	})
}

// handle2FAVerify verifies the TOTP code and enables 2FA.
func (s *Server) handle2FAVerify(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role != "admin" {
		http.Error(w, "forbidden: admin only", http.StatusForbidden)
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if user.TOTPEnabled {
		http.Error(w, "2FA already enabled", http.StatusBadRequest)
		return
	}

	if user.TOTPSecret == "" {
		http.Error(w, "2FA not set up. Call /2fa/setup first", http.StatusBadRequest)
		return
	}

	if !totp.Validate(req.Code, user.TOTPSecret) {
		http.Error(w, "invalid 2FA code", http.StatusUnauthorized)
		return
	}

	if err := s.db.EnableTOTP(user.ID); err != nil {
		s.logger.Error("failed to enable TOTP", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "ENABLE_2FA", "user", claims.UserID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "2fa_enabled"})
}

// handle2FADisable disables 2FA for the current user.
func (s *Server) handle2FADisable(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role != "admin" {
		http.Error(w, "forbidden: admin only", http.StatusForbidden)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		http.Error(w, "invalid password", http.StatusUnauthorized)
		return
	}

	if err := s.db.DisableTOTP(user.ID); err != nil {
		s.logger.Error("failed to disable TOTP", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "DISABLE_2FA", "user", claims.UserID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "2fa_disabled"})
}

func (s *Server) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	user.PasswordHash = ""
	jsonResponse(w, http.StatusOK, user)
}

// ---------- Устройства ----------

func (s *Server) listDevices(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	devicesMap := s.stateManager.GetAll()
	allDevices := make([]*models.Device, 0, len(devicesMap))
	for _, dev := range devicesMap {
		allDevices = append(allDevices, dev)
	}

	var filtered []*models.Device
	switch claims.Role {
	case "admin", "support":
		filtered = allDevices
	case "owner":
		for _, dev := range allDevices {
			if dev.OwnerID != nil && *dev.OwnerID == claims.UserID {
				filtered = append(filtered, dev)
			}
		}
	default:
		filtered = []*models.Device{}
	}

	resp := make([]map[string]interface{}, len(filtered))
	for i, dev := range filtered {
		resp[i] = map[string]interface{}{
			"device_id":     dev.DeviceID,
			"owner_id":      dev.OwnerID,
			"name":          dev.Name,
			"location":      dev.Location,
			"vendor_type":   dev.VendorType,
			"status":        dev.Status,
			"last_seen":     dev.LastSeen,
			"registered_at": dev.RegisteredAt,
			"user_agent":    dev.UserAgent,
			// P2P fields if present
			"p2p_brand":    dev.P2PBrand,
			"p2p_serial":   dev.P2PSerial,
			"cloud_status": dev.CloudStatus,
		}
	}
	jsonResponse(w, http.StatusOK, resp)
}

func (s *Server) getDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dev, ok := s.stateManager.Get(id)
	if !ok {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}
	claims := auth.GetClaims(r)
	if claims.Role == "owner" {
		if dev.OwnerID == nil || *dev.OwnerID != claims.UserID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"device_id":     dev.DeviceID,
		"owner_id":      dev.OwnerID,
		"name":          dev.Name,
		"location":      dev.Location,
		"vendor_type":   dev.VendorType,
		"status":        dev.Status,
		"last_seen":     dev.LastSeen,
		"registered_at": dev.RegisteredAt,
		"user_agent":    dev.UserAgent,
		"p2p_brand":     dev.P2PBrand,
		"p2p_serial":    dev.P2PSerial,
		"cloud_status":  dev.CloudStatus,
	})
}

func (s *Server) getDeviceStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dev, ok := s.stateManager.Get(id)
	if !ok {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}
	claims := auth.GetClaims(r)
	if claims.Role == "owner" {
		if dev.OwnerID == nil || *dev.OwnerID != claims.UserID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"device_id": dev.DeviceID,
		"status":    dev.Status,
		"last_seen": dev.LastSeen.Format(time.RFC3339),
	})
}

// ---------- Внешние алермы (существующие) ----------

// Структуры для парсинга Hikvision ISAPI XML
type HikvisionEvent struct {
	XMLName     xml.Name `xml:"EventNotificationAlert"`
	EventType   string   `xml:"eventType"`
	EventState  string   `xml:"eventState"`
	ChannelID   int      `xml:"channelID"`
	DateTime    string   `xml:"dateTime"`
	Description string   `xml:"eventDescription"`
	PicName     string   `xml:"picName"`
	PicUrl      string   `xml:"picUrl"`
	ImageBase64 string   `xml:"image"`
}

// safeDeviceID очищает строку от недопустимых символов для использования в имени файла и device_id
func safeDeviceID(raw string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9\-_\.]`)
	return re.ReplaceAllString(raw, "_")
}

func (s *Server) handleExternalAlarm(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID    string `json:"device_id"`
		EventType   string `json:"event_type"`
		Priority    int    `json:"priority"`
		Method      int    `json:"method"`
		Description string `json:"description"`
		RawData     string `json:"raw_data"`
		Protocol    string `json:"protocol"`
		Timestamp   string `json:"timestamp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Error("Invalid external alarm JSON", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	alarmTime := time.Now()
	if req.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			alarmTime = t
		}
	}
	alarm := &models.Alarm{
		DeviceID:    req.DeviceID,
		Priority:    models.AlarmPriority(req.Priority),
		Method:      models.AlarmMethod(req.Method),
		Timestamp:   alarmTime,
		Description: req.Description,
	}
	s.stateManager.AddAlarm(req.DeviceID, alarm)
	s.logger.Info("External alarm received", "device_id", req.DeviceID, "event", req.EventType)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleExternalAlarmXML(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Failed to read XML body", "error", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var event HikvisionEvent
	if err := xml.Unmarshal(body, &event); err != nil {
		s.logger.Error("XML parse error", "error", err)
		http.Error(w, "Invalid XML", http.StatusBadRequest)
		return
	}

	if event.EventState != "active" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	clientIP := strings.Split(r.RemoteAddr, ":")[0]
	clientIP = strings.Trim(clientIP, "[]")
	deviceID := fmt.Sprintf("hikvision_%s_ch%d", safeDeviceID(clientIP), event.ChannelID)

	priority := mapHikvisionPriority(event.EventType)

	alarm := &models.Alarm{
		DeviceID:    deviceID,
		Priority:    priority,
		Method:      models.AlarmMethodMotionDetection,
		Timestamp:   time.Now(),
		Description: fmt.Sprintf("%s: %s", event.EventType, event.Description),
	}

	if s.config.SaveEventImages && event.ImageBase64 != "" {
		imagePath, err := s.saveBase64Image(deviceID, event.ImageBase64)
		if err == nil {
			alarm.ImagePath = imagePath
		} else {
			s.logger.Warn("Failed to save base64 image", "error", err)
		}
	}

	s.stateManager.AddAlarm(deviceID, alarm)
	s.logger.Info("XML alarm received", "device_id", deviceID, "event", event.EventType)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleExternalAlarmVigi(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID    string `json:"device_id"`
		Channel     int    `json:"channel"`
		ChannelName string `json:"channel_name"`
		MessageType int    `json:"message_type"`
		SubType     int    `json:"sub_type"`
		LocalTime   string `json:"localtime"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Error("Invalid Vigi JSON", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	deviceID := fmt.Sprintf("vigi_%s_%d", safeDeviceID(req.DeviceID), req.Channel)
	priority := mapVigiPriority(req.MessageType)
	description := fmt.Sprintf("Vigi event type %d (subtype %d) on channel %s", req.MessageType, req.SubType, req.ChannelName)

	alarm := &models.Alarm{
		DeviceID:    deviceID,
		Priority:    priority,
		Method:      models.AlarmMethodMotionDetection,
		Timestamp:   time.Now(),
		Description: description,
	}

	s.stateManager.AddAlarm(deviceID, alarm)
	s.logger.Info("Vigi alarm received", "device_id", deviceID, "type", req.MessageType, "subtype", req.SubType)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func mapHikvisionPriority(eventType string) models.AlarmPriority {
	switch {
	case strings.Contains(eventType, "Motion"), strings.Contains(eventType, "VMD"):
		return models.AlarmPriorityHigh
	case strings.Contains(eventType, "VideoLoss"), strings.Contains(eventType, "Tamper"):
		return models.AlarmPriorityHigh
	case strings.Contains(eventType, "HDD"), strings.Contains(eventType, "Storage"):
		return models.AlarmPriorityMedium
	default:
		return models.AlarmPriorityLow
	}
}

func mapVigiPriority(messageType int) models.AlarmPriority {
	switch messageType {
	case 1:
		return models.AlarmPriorityHigh
	case 2, 3:
		return models.AlarmPriorityHigh
	case 4, 5:
		return models.AlarmPriorityMedium
	default:
		return models.AlarmPriorityLow
	}
}

func (s *Server) saveBase64Image(deviceID, base64Data string) (string, error) {
	if idx := strings.Index(base64Data, ","); idx != -1 {
		base64Data = base64Data[idx+1:]
	}
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", err
	}

	safeID := safeDeviceID(deviceID)
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("%s_%d.jpg", safeID, timestamp)
	fullPath := filepath.Join(s.imagesDir, filename)

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", err
	}

	return "/api/v1/images/" + filename, nil
}

// ---------- P2P Alarm endpoint ----------
func (s *Server) handleP2PAlarm(w http.ResponseWriter, r *http.Request) {
	// Проверка API-ключа (может быть в заголовке X-API-Key или в query)
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}
	if apiKey != s.p2pAPIKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		DeviceID    string `json:"device_id"`
		EventType   string `json:"event_type"`
		Priority    int    `json:"priority"`
		Method      int    `json:"method"`
		Description string `json:"description"`
		Timestamp   string `json:"timestamp"`
		ImageBase64 string `json:"image_base64"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Error("Invalid P2P alarm JSON", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	alarmTime := time.Now()
	if req.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			alarmTime = t
		}
	}

	alarm := &models.Alarm{
		DeviceID:    req.DeviceID,
		Priority:    models.AlarmPriority(req.Priority),
		Method:      models.AlarmMethod(req.Method),
		Timestamp:   alarmTime,
		Description: req.Description,
	}

	if s.config.SaveEventImages && req.ImageBase64 != "" {
		imagePath, err := s.saveBase64Image(req.DeviceID, req.ImageBase64)
		if err == nil {
			alarm.ImagePath = imagePath
		} else {
			s.logger.Warn("Failed to save base64 image from P2P alarm", "error", err)
		}
	}

	s.stateManager.AddAlarm(req.DeviceID, alarm)
	s.logger.Info("P2P alarm received", "device_id", req.DeviceID, "event", req.EventType)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------- P2P Management ----------
func (s *Server) listP2PDevices(w http.ResponseWriter, r *http.Request) {
	if s.p2pGatewayURL == "" {
		http.Error(w, "P2P gateway not configured", http.StatusServiceUnavailable)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), "GET", s.p2pGatewayURL+"/p2p/devices", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("X-API-Key", s.p2pAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Failed to fetch P2P devices", "error", err)
		http.Error(w, "Failed to fetch P2P devices", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) registerP2PDevice(w http.ResponseWriter, r *http.Request) {
	if s.p2pGatewayURL == "" {
		http.Error(w, "P2P gateway not configured", http.StatusServiceUnavailable)
		return
	}

	// Читаем тело запроса для валидации
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Восстанавливаем тело для повторного использования
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Проксируем запрос к p2p-gateway
	req, err := http.NewRequestWithContext(r.Context(), "POST", s.p2pGatewayURL+"/p2p/register", bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", s.p2pAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Failed to register P2P device", "error", err)
		http.Error(w, "Failed to register P2P device", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Если регистрация успешна, можно также сохранить устройство в локальной БД (опционально)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) getP2PDeviceStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if s.p2pGatewayURL == "" {
		http.Error(w, "P2P gateway not configured", http.StatusServiceUnavailable)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), "GET", s.p2pGatewayURL+"/p2p/status/"+id, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("X-API-Key", s.p2pAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Failed to get P2P device status", "error", err)
		http.Error(w, "Failed to get P2P device status", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) sendP2PCommand(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if s.p2pGatewayURL == "" {
		http.Error(w, "P2P gateway not configured", http.StatusServiceUnavailable)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	req, err := http.NewRequestWithContext(r.Context(), "POST", s.p2pGatewayURL+"/p2p/command/"+id, bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", s.p2pAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Failed to send P2P command", "error", err)
		http.Error(w, "Failed to send P2P command", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) getP2PSnapshot(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if s.p2pGatewayURL == "" {
		http.Error(w, "P2P gateway not configured", http.StatusServiceUnavailable)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), "GET", s.p2pGatewayURL+"/p2p/snapshot/"+id, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("X-API-Key", s.p2pAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Failed to get P2P snapshot", "error", err)
		http.Error(w, "Failed to get P2P snapshot", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Предполагаем, что p2p-gateway возвращает изображение напрямую (JPEG)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ---------- Аналитика ----------

func (s *Server) getPredictions(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	limit := 10
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	claims := auth.GetClaims(r)
	if claims.Role == "owner" {
		if deviceID == "" {
			jsonResponse(w, http.StatusOK, []interface{}{})
			return
		}
		dev, ok := s.stateManager.Get(deviceID)
		if !ok || dev.OwnerID == nil || *dev.OwnerID != claims.UserID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}
	predictions, err := s.db.GetPredictions(deviceID, limit)
	if err != nil {
		s.logger.Error("failed to get predictions", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, http.StatusOK, predictions)
}

// ---------- Поиск логов ----------

func (s *Server) searchLogs(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "support" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	deviceID := r.URL.Query().Get("device_id")
	level := r.URL.Query().Get("level")
	keyword := r.URL.Query().Get("keyword")
	timeFrom := r.URL.Query().Get("time_from")
	timeTo := r.URL.Query().Get("time_to")

	logs, err := s.db.SearchLogs(deviceID, level, keyword, timeFrom, timeTo)
	if err != nil {
		s.logger.Error("failed to search logs", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, http.StatusOK, logs)
}

// ---------- Изображения ----------

func (s *Server) getImage(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if strings.Contains(filename, "..") || strings.ContainsAny(filename, "/\\") {
		http.Error(w, "invalid filename", http.StatusBadRequest)
		return
	}
	filePath := filepath.Join(s.imagesDir, filename)
	http.ServeFile(w, r, filePath)
}

func (s *Server) listDeviceImages(w http.ResponseWriter, r *http.Request) {
	deviceId := chi.URLParam(r, "deviceId")
	claims := auth.GetClaims(r)
	if claims.Role == "owner" {
		dev, ok := s.stateManager.Get(deviceId)
		if !ok || dev.OwnerID == nil || *dev.OwnerID != claims.UserID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}
	pattern := filepath.Join(s.imagesDir, safeDeviceID(deviceId)+"_*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	baseNames := make([]string, len(files))
	for i, f := range files {
		baseNames[i] = filepath.Base(f)
	}
	jsonResponse(w, http.StatusOK, baseNames)
}

// ---------- Вспомогательные ----------

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// backend/internal/api/server.go (добавить в конец файла)

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" {
		http.Error(w, "forbidden: admin only", http.StatusForbidden)
		return
	}
	users, err := s.db.GetUsers()
	if err != nil {
		s.logger.Error("failed to get users", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, http.StatusOK, users)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" {
		http.Error(w, "forbidden: admin only", http.StatusForbidden)
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Валидация роли
	validRoles := map[string]bool{"admin": true, "manager": true, "technician": true, "viewer": true, "support": true, "owner": true}
	if !validRoles[req.Role] {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	user, err := s.db.CreateUser(req.Username, hashed, req.Role, req.Email, nil)
	if err != nil {
		s.logger.Error("failed to create user", "error", err)
		http.Error(w, "user already exists or db error", http.StatusConflict)
		return
	}

	// Аудит
	_ = s.db.SaveAudit(claims.UserID, "CREATE_USER", "user", user.ID, nil, map[string]string{"username": req.Username, "role": req.Role})

	user.PasswordHash = ""
	jsonResponse(w, http.StatusCreated, user)
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" {
		http.Error(w, "forbidden: admin only", http.StatusForbidden)
		return
	}
	id := chi.URLParam(r, "id")
	var req struct {
		Role   string `json:"role"`
		Status string `json:"status"`
		Email  string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	updates := make(map[string]interface{})
	if req.Role != "" {
		validRoles := map[string]bool{"admin": true, "manager": true, "technician": true, "viewer": true, "support": true, "owner": true}
		if !validRoles[req.Role] {
			http.Error(w, "invalid role", http.StatusBadRequest)
			return
		}
		updates["role"] = req.Role
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}

	if err := s.db.UpdateUser(id, updates); err != nil {
		http.Error(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "UPDATE_USER", "user", id, nil, updates)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" {
		http.Error(w, "forbidden: admin only", http.StatusForbidden)
		return
	}
	id := chi.URLParam(r, "id")

	// Защита от удаления самого себя
	if id == claims.UserID {
		http.Error(w, "cannot delete yourself", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteUser(id); err != nil {
		http.Error(w, "failed to delete user", http.StatusInternalServerError)
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "DELETE_USER", "user", id, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// backend/internal/api/server.go — добавить в конец файла

// ---------- Settings (Services) ----------
func (s *Server) getServicesSettings(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "manager" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	settings, err := s.db.GetSystemSettings()
	if err != nil {
		s.logger.Error("failed to get services settings", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, http.StatusOK, settings)
}

func (s *Server) updateServicesSettings(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" {
		http.Error(w, "forbidden: admin only", http.StatusForbidden)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := s.db.UpdateMultipleSettings(req, claims.UserID); err != nil {
		s.logger.Error("failed to update services settings", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = s.db.SaveAudit(claims.UserID, "UPDATE_SERVICES_SETTINGS", "system_settings", "services", nil, req)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

// ---------- Password Management ----------

// changeMyPassword — пользователь меняет свой пароль (с проверкой текущего)
func (s *Server) changeMyPassword(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Валидация нового пароля
	if len(req.NewPassword) < 6 {
		http.Error(w, "new password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	// Проверяем текущий пароль
	currentHash, err := s.db.GetPasswordHash(claims.UserID)
	if err != nil {
		s.logger.Error("failed to get password hash", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !auth.CheckPasswordHash(req.CurrentPassword, currentHash) {
		http.Error(w, "current password is incorrect", http.StatusUnauthorized)
		return
	}

	// Хешируем новый пароль
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		s.logger.Error("failed to hash new password", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Обновляем в БД
	if err := s.db.UpdatePassword(claims.UserID, newHash); err != nil {
		s.logger.Error("failed to update password", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "CHANGE_PASSWORD", "user", claims.UserID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "password_changed"})
}

// resetUserPassword — админ сбрасывает пароль пользователю (без проверки текущего)
func (s *Server) resetUserPassword(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" {
		http.Error(w, "forbidden: admin only", http.StatusForbidden)
		return
	}

	targetUserID := chi.URLParam(r, "id")
	if targetUserID == "" {
		http.Error(w, "user id required", http.StatusBadRequest)
		return
	}

	// Защита от сброса пароля самому себе через этот эндпоинт
	if targetUserID == claims.UserID {
		http.Error(w, "use /users/me/password to change your own password", http.StatusBadRequest)
		return
	}

	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 6 {
		http.Error(w, "new password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		s.logger.Error("failed to hash password", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.db.UpdatePassword(targetUserID, newHash); err != nil {
		s.logger.Error("failed to reset password", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "RESET_PASSWORD", "user", targetUserID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "password_reset"})
}

// handleWebSocket handles WebSocket connections for real-time alarm notifications.
// JWT token is passed via query parameter ?token=...
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
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

	s.logger.Info("WebSocket client connected", "user_id", claims.UserID, "username", claims.Username)

	_, err = ws.ServeWs(s.wsHub, w, r)
	if err != nil {
		s.logger.Error("WebSocket upgrade failed", "error", err, "user_id", claims.UserID)
		return
	}
}

// BroadcastAlarm sends an alarm to all connected WebSocket clients.
func (s *Server) BroadcastAlarm(alarm *models.Alarm) {
	if s.wsHub == nil {
		return
	}

	data, err := json.Marshal(map[string]interface{}{
		"type":  "alarm",
		"alarm": alarm,
	})
	if err != nil {
		s.logger.Error("Failed to marshal alarm for WebSocket", "error", err)
		return
	}

	s.wsHub.Broadcast(data)
}

// getUserSessions returns all active sessions for the current user.
func (s *Server) getUserSessions(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sessions, err := s.db.GetUserSessions(claims.UserID)
	if err != nil {
		s.logger.Error("failed to get user sessions", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, sessions)
}

// revokeSession revokes a specific session.
func (s *Server) revokeSession(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		http.Error(w, "session id required", http.StatusBadRequest)
		return
	}

	if err := s.db.RevokeSession(sessionID); err != nil {
		s.logger.Error("failed to revoke session", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "REVOKE_SESSION", "session", sessionID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// revokeAllOtherSessions revokes all sessions for the current user except the current one.
func (s *Server) revokeAllOtherSessions(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// We need the current session ID. For simplicity, we can pass it in the request body or header,
	// but since we don't track it in the JWT, we'll just revoke all sessions for this user.
	// A better approach is to store the session ID in the JWT or pass it in the request.
	// Let's assume the frontend passes the current session ID in the request body.
	var req struct {
		CurrentSessionID string `json:"current_session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := s.db.RevokeAllOtherSessions(claims.UserID, req.CurrentSessionID); err != nil {
		s.logger.Error("failed to revoke all other sessions", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "REVOKE_ALL_SESSIONS", "user", claims.UserID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "revoked_all"})
}

// ---------- Password Reset (Forgot Password) ----------

// handleForgotPassword generates a reset token and returns it (in production, send via email).
func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "email required", http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUserByEmail(req.Email)
	if err != nil {
		// Не раскрываем, существует ли пользователь — всегда возвращаем успех
		jsonResponse(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"message": "If the email exists, a reset link has been sent",
		})
		return
	}

	// Генерируем токен
	token := auth.GenerateResetToken()
	expiresAt := time.Now().Add(1 * time.Hour)

	if err := s.db.CreatePasswordResetToken(user.ID, token, expiresAt); err != nil {
		s.logger.Error("failed to create reset token", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// В production здесь нужно отправить email с ссылкой.
	// Для простоты возвращаем токен в ответе (только для dev/test).
	s.logger.Info("Password reset token generated", "user_id", user.ID, "email", req.Email)

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"message":     "Reset token generated (check logs for production)",
		"reset_token": token, // В production убрать! Отправлять только по email.
		"expires_at":  expiresAt,
	})
}

// handleResetPasswordWithToken resets password using a valid reset token.
func (s *Server) handleResetPasswordWithToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		http.Error(w, "token and new_password required", http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 6 {
		http.Error(w, "new password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	userID, expiresAt, err := s.db.GetPasswordResetToken(req.Token)
	if err != nil {
		http.Error(w, "invalid or expired token", http.StatusBadRequest)
		return
	}

	if time.Now().After(expiresAt) {
		_ = s.db.DeletePasswordResetToken(req.Token)
		http.Error(w, "token expired", http.StatusBadRequest)
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		s.logger.Error("failed to hash password", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.db.UpdatePassword(userID, newHash); err != nil {
		s.logger.Error("failed to update password", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_ = s.db.DeletePasswordResetToken(req.Token)
	_ = s.db.SaveAudit(userID, "RESET_PASSWORD_WITH_TOKEN", "user", userID, nil, nil)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "password_reset"})
}
