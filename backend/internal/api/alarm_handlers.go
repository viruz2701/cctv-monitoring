package api

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gb-telemetry-collector/internal/models"
)

// ---------- Структуры для парсинга Hikvision ISAPI XML ----------

// HikvisionEvent описывает XML-событие от Hikvision ISAPI.
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

// ---------- Вспомогательные ----------

// safeDeviceID очищает строку от недопустимых символов для использования в имени файла и device_id.
func safeDeviceID(raw string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9\-_\.]`)
	return re.ReplaceAllString(raw, "_")
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

// ---------- Внешние алермы ----------

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
		respondError(w, r, NewBadRequestError("Bad request"))
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
		respondError(w, r, NewBadRequestError("Failed to read body"))
		return
	}
	defer r.Body.Close()

	var event HikvisionEvent
	if err := xml.Unmarshal(body, &event); err != nil {
		s.logger.Error("XML parse error", "error", err)
		respondError(w, r, NewBadRequestError("Invalid XML"))
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
		respondError(w, r, NewBadRequestError("Bad request"))
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

// ---------- P2P Alarm ----------

func (s *Server) handleP2PAlarm(w http.ResponseWriter, r *http.Request) {
	// Проверка API-ключа (может быть в заголовке X-API-Key или в query)
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}
	if apiKey != s.p2pAPIKey {
		respondError(w, r, NewUnauthorizedError("Unauthorized"))
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
		respondError(w, r, NewBadRequestError("Bad request"))
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
			s.logger.Warn("Failed to save P2P image", "error", err)
		}
	}

	s.stateManager.AddAlarm(req.DeviceID, alarm)
	s.logger.Info("P2P alarm received", "device_id", req.DeviceID, "event", req.EventType)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}
