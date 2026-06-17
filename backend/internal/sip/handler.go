package sip

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"gb-telemetry-collector/internal/config" // ДОБАВЛЕНО
	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/state"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// ═══════════════════════════════════════════════════════════════════════
// GB28181 XML Structures
// ═══════════════════════════════════════════════════════════════════════

// GB28181Notify — стандартный контейнер для уведомлений устройства
// (Keepalive, Alarm, MobilePosition, Catalog и т.д.)
type GB28181Notify struct {
	XMLName  xml.Name `xml:"Notify"`
	CmdType  string   `xml:"CmdType"`
	SN       int      `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
	// Keepalive
	Status string `xml:"Status,omitempty"`
	// Alarm
	AlarmPriority    int    `xml:"AlarmPriority,omitempty"`
	AlarmMethod      int    `xml:"AlarmMethod,omitempty"`
	AlarmTime        string `xml:"AlarmTime,omitempty"`
	AlarmDescription string `xml:"AlarmDescription,omitempty"`
	AlarmType        string `xml:"AlarmType,omitempty"`
	// MobilePosition
	Longitude string `xml:"Longitude,omitempty"`
	Latitude  string `xml:"Latitude,omitempty"`
	Speed     string `xml:"Speed,omitempty"`
}

// GB28181Response — ответ устройства на запрос сервера (Catalog, DeviceInfo)
type GB28181Response struct {
	XMLName    xml.Name      `xml:"Response"`
	CmdType    string        `xml:"CmdType"`
	SN         int           `xml:"SN"`
	DeviceID   string        `xml:"DeviceID"`
	Num        int           `xml:"Num,omitempty"`
	DeviceList []CatalogItem `xml:"DeviceList>Item,omitempty"`
	// DeviceInfo
	DeviceName   string `xml:"DeviceName,omitempty"`
	Manufacturer string `xml:"Manufacturer,omitempty"`
	Model        string `xml:"Model,omitempty"`
	Firmware     string `xml:"Firmware,omitempty"`
	Channel      int    `xml:"Channel,omitempty"`
}

// CatalogItem — элемент каталога устройств NVR
type CatalogItem struct {
	DeviceID     string `xml:"DeviceID"`
	Name         string `xml:"Name"`
	Manufacturer string `xml:"Manufacturer"`
	Model        string `xml:"Model"`
	Owner        string `xml:"Owner"`
	CivilCode    string `xml:"CivilCode"`
	Address      string `xml:"Address"`
	Parental     int    `xml:"Parental"`
	ParentID     string `xml:"ParentID"`
	SafetyWay    int    `xml:"SafetyWay"`
	RegisterWay  int    `xml:"RegisterWay"`
	Secrecy      int    `xml:"Secrecy"`
	IPAddress    string `xml:"IPAddress"`
	Port         int    `xml:"Port"`
	Password     string `xml:"Password"`
	Status       string `xml:"Status"` // ON, OFF, VLOST, FAULT
}

// ═══════════════════════════════════════════════════════════════════════
// SIP Message Parser
// ═══════════════════════════════════════════════════════════════════════

// SIPMessage — разобранный SIP-запрос или ответ
type SIPMessage struct {
	IsRequest  bool
	Method     string              // Для запросов: REGISTER, MESSAGE, INVITE...
	URI        string              // Для запросов: SIP URI
	StatusCode int                 // Для ответов: 200, 401...
	StatusText string              // Для ответов: "OK", "Unauthorized"
	Headers    map[string][]string // Массив значений (Via может быть несколько)
	Body       string
}

func parseSIPMessage(data []byte) *SIPMessage {
	msg := &SIPMessage{
		Headers: make(map[string][]string),
	}

	// Нормализуем \r\n → \n для упрощения парсинга
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return nil
	}

	// Первая строка — Request-Line или Status-Line
	parts := strings.Fields(lines[0])
	if len(parts) < 3 {
		return nil
	}

	if parts[0] == "SIP/2.0" {
		// Ответ: SIP/2.0 200 OK
		msg.IsRequest = false
		if code, err := strconv.Atoi(parts[1]); err == nil {
			msg.StatusCode = code
		}
		msg.StatusText = strings.Join(parts[2:], " ")
	} else {
		// Запрос: REGISTER sip:server SIP/2.0
		msg.IsRequest = true
		msg.Method = parts[0]
		msg.URI = parts[1]
	}

	// Парсим заголовки с поддержкой folded headers (RFC 3261 §7.3.1)
	bodyStart := -1
	var lastKey string
	for i := 1; i < len(lines); i++ {
		line := lines[i]

		// Пустая строка = начало body
		if line == "" {
			bodyStart = i + 1
			break
		}

		// Folded header: продолжение начинается с пробела/табуляции
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			if lastKey != "" {
				vals := msg.Headers[lastKey]
				if len(vals) > 0 {
					vals[len(vals)-1] += " " + strings.TrimSpace(line)
				}
			}
			continue
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		key := normalizeHeaderKey(strings.TrimSpace(line[:colonIdx]))
		val := strings.TrimSpace(line[colonIdx+1:])
		lastKey = key
		msg.Headers[key] = append(msg.Headers[key], val)
	}

	// Извлекаем body
	if bodyStart > 0 && bodyStart < len(lines) {
		msg.Body = strings.Join(lines[bodyStart:], "\n")
	}

	return msg
}

// normalizeHeaderKey — разворачивает компактные формы SIP-заголовков (RFC 3261 §7.3.3)
func normalizeHeaderKey(key string) string {
	switch strings.ToLower(key) {
	case "v":
		return "Via"
	case "f":
		return "From"
	case "t":
		return "To"
	case "i":
		return "Call-ID"
	case "m":
		return "Contact"
	case "l":
		return "Content-Length"
	case "c":
		return "Content-Type"
	case "e":
		return "Content-Encoding"
	case "k":
		return "Supported"
	case "s":
		return "Subject"
	case "r":
		return "Refer-To"
	case "o":
		return "Event"
	default:
		return key
	}
}

func (m *SIPMessage) getHeader(key string) string {
	if vals, ok := m.Headers[key]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func (m *SIPMessage) getHeaderAll(key string) []string {
	return m.Headers[key]
}

// ═══════════════════════════════════════════════════════════════════════
// GB2312 / GBK Decoding
// ═══════════════════════════════════════════════════════════════════════

func decodeBody(data []byte) string {
	if bytes.Contains(data, []byte("GB2312")) || bytes.Contains(data, []byte("gb2312")) ||
		bytes.Contains(data, []byte("GBK")) || bytes.Contains(data, []byte("gbk")) {
		decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(data)
		if err == nil {
			return string(decoded)
		}
	}
	return string(data)
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

var (
	tagCounter   uint64
	privateCIDRs []*net.IPNet
)

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"100.64.0.0/10", // CGNAT
	} {
		_, network, _ := net.ParseCIDR(cidr)
		privateCIDRs = append(privateCIDRs, network)
	}
}

func generateTag() string {
	val := atomic.AddUint64(&tagCounter, 1)
	return fmt.Sprintf("gb%x%x", time.Now().UnixNano()%0xFFFF, val%0xFFFF)
}

func generateBranch() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "z9hG4bK" + hex.EncodeToString(b)
}

func generateCallID(host string) string {
	b := make([]byte, 12)
	rand.Read(b)
	return fmt.Sprintf("%s@%s", hex.EncodeToString(b), host)
}

func isPrivateIP(ip net.IP) bool {
	for _, cidr := range privateCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════════
// SIPHandler
// ═══════════════════════════════════════════════════════════════════════

type SIPHandler struct {
	stateManager state.DeviceStateManager
	logger       *slog.Logger
	conn         *net.UDPConn
	host         string
	port         int
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	snCounter    uint64
	config       config.GB28181Config // <-- новое поле
}

func NewSIPHandler(stateMgr state.DeviceStateManager, logger *slog.Logger, cfg config.GB28181Config) *SIPHandler {
	return &SIPHandler{
		stateManager: stateMgr,
		logger:       logger,
		host:         cfg.Host,
		port:         cfg.Port,
		config:       cfg,
	}
}

func (h *SIPHandler) Start(ctx context.Context) error {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(h.host),
		Port: h.port,
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen UDP: %w", err)
	}
	h.conn = conn

	h.ctx, h.cancel = context.WithCancel(ctx)
	h.wg.Add(1)
	go h.udpReceiver()

	h.logger.Info("SIP/GB28181 UDP server started", "addr", addr.String())
	return nil
}

// Stop — остановка (совместимо с ProtocolHandler interface)
func (h *SIPHandler) Stop() error {
	if h.cancel != nil {
		h.cancel()
	}
	if h.conn != nil {
		h.conn.Close()
	}
	h.wg.Wait()
	return nil
}

// GetStateManager — экспортируемый геттер для stateManager
func (h *SIPHandler) GetStateManager() state.DeviceStateManager {
	return h.stateManager
}

// MessageContext — контекст SIP-сообщения для worker pool
type MessageContext struct {
	Data   []byte
	Remote *net.UDPAddr
}

// MessageProcessor — интерфейс обработчика SIP-сообщений
type MessageProcessor interface {
	ProcessMessage(msg *MessageContext)
}

func (h *SIPHandler) udpReceiver() {
	defer h.wg.Done()
	buf := make([]byte, 65535)
	for {
		select {
		case <-h.ctx.Done():
			return
		default:
		}

		h.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, remoteAddr, err := h.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			select {
			case <-h.ctx.Done():
				return
			default:
				h.logger.Error("UDP read error", "error", err)
				continue
			}
		}

		data := make([]byte, n)
		copy(data, buf[:n])
		go h.processMessage(data, remoteAddr)
	}
}

func (h *SIPHandler) processMessage(data []byte, remote *net.UDPAddr) {
	msg := parseSIPMessage(data)
	if msg == nil {
		h.logger.Debug("Failed to parse SIP message", "remote", remote.String())
		return
	}

	if !msg.IsRequest {
		h.handleResponse(msg, remote)
		return
	}

	h.logger.Debug("SIP request",
		"method", msg.Method,
		"remote", remote.String(),
	)

	switch msg.Method {
	case "REGISTER":
		h.handleRegister(msg, remote)
	case "MESSAGE":
		h.handleMessage(msg, remote)
	case "NOTIFY":
		h.handleNotify(msg, remote)
	case "INVITE":
		h.handleInvite(msg, remote)
	case "BYE":
		h.sendResponse(remote, msg, 200, "OK", nil)
	case "ACK":
		// ACK не требует ответа
	case "SUBSCRIBE":
		h.handleSubscribe(msg, remote)
	case "INFO":
		h.sendResponse(remote, msg, 200, "OK", nil)
	default:
		h.logger.Warn("Unhandled SIP method", "method", msg.Method)
		h.sendResponse(remote, msg, 405, "Method Not Allowed", nil)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// REGISTER
// ═══════════════════════════════════════════════════════════════════════

func (h *SIPHandler) handleRegister(msg *SIPMessage, remote *net.UDPAddr) {
	deviceID := h.extractDeviceID(msg)
	if deviceID == "" {
		h.logger.Warn("REGISTER without device ID", "remote", remote.String())
		h.sendResponse(remote, msg, 400, "Bad Request", nil)
		return
	}

	gbInfo := ParseGB28181DeviceID(deviceID)
	if gbInfo.IsValid {
		h.logger.Debug("GB28181 DeviceID parsed",
			"device_id", deviceID,
			"info", gbInfo.String(),
		)
	}

	expires := 3600
	if expStr := msg.getHeader("Expires"); expStr != "" {
		if e, err := strconv.Atoi(strings.TrimSpace(expStr)); err == nil {
			expires = e
		}
	}

	userAgent := msg.getHeader("User-Agent")
	contactAddr := h.extractContactAddr(msg, remote)

	dev, exists := h.stateManager.Get(deviceID)
	if !exists {
		dev = &models.Device{
			DeviceID:          deviceID,
			Status:            models.StatusOnline,
			LastSeen:          time.Now(),
			RegisteredAt:      time.Now(),
			ContactAddr:       contactAddr,
			HeartbeatInterval: expires,
			UserAgent:         userAgent,
			VendorType:        h.detectVendor(gbInfo),
			Name:              fmt.Sprintf("GB28181-%s", deviceID[len(deviceID)-6:]),
			Location:          remote.IP.String(),
		}
		h.logger.Info("New GB28181 device registered",
			"device_id", deviceID,
			"contact", contactAddr.String(),
			"vendor", dev.VendorType,
		)
	} else {
		dev.LastSeen = time.Now()
		dev.ContactAddr = contactAddr
		dev.HeartbeatInterval = expires
		dev.UserAgent = userAgent
		if dev.Status == models.StatusOffline {
			dev.Status = models.StatusOnline
		}
		h.logger.Debug("Device re-registered", "device_id", deviceID)
	}
	h.stateManager.Set(dev)

	extra := map[string]string{"Expires": strconv.Itoa(expires)}
	h.sendResponse(remote, msg, 200, "OK", extra)

	// Запрашиваем каталог после регистрации (асинхронно)
	go func() {
		time.Sleep(500 * time.Millisecond)
		h.requestCatalog(deviceID)
		time.Sleep(200 * time.Millisecond)
		h.requestDeviceInfo(deviceID)
	}()
}

// ═══════════════════════════════════════════════════════════════════════
// MESSAGE (Keepalive, Alarm, Catalog Response)
// ═══════════════════════════════════════════════════════════════════════

func (h *SIPHandler) handleMessage(msg *SIPMessage, remote *net.UDPAddr) {
	deviceID := h.extractDeviceID(msg)
	if deviceID == "" {
		h.logger.Warn("MESSAGE without device ID", "remote", remote.String())
		h.sendResponse(remote, msg, 400, "Bad Request", nil)
		return
	}

	h.stateManager.UpdateLastSeen(deviceID)
	h.stateManager.SetOnline(deviceID)

	// Отвечаем 200 OK сразу
	h.sendResponse(remote, msg, 200, "OK", nil)

	// Парсим body
	body := decodeBody([]byte(msg.Body))
	if body == "" {
		return
	}

	// Основной парсер: GB28181 <Notify>
	var notify GB28181Notify
	if err := xml.Unmarshal([]byte(body), &notify); err == nil && notify.CmdType != "" {
		h.handleGB28181Notify(deviceID, &notify)
		return
	}

	// Fallback: <Response> (может прийти в MESSAGE)
	var resp GB28181Response
	if err := xml.Unmarshal([]byte(body), &resp); err == nil && resp.CmdType != "" {
		h.handleGB28181Response(&resp)
		return
	}

	// Fallback: legacy форматы (не-GB28181)
	if strings.Contains(body, "Keepalive") {
		var k KeepaliveXML
		if xml.Unmarshal([]byte(body), &k) == nil {
			h.handleKeepaliveFallback(deviceID, &k)
		}
	} else if strings.Contains(body, "Alarm") {
		var a AlarmXML
		if xml.Unmarshal([]byte(body), &a) == nil {
			h.handleAlarmFallback(deviceID, &a)
		}
	}
}

func (h *SIPHandler) handleGB28181Notify(deviceID string, notify *GB28181Notify) {
	switch notify.CmdType {
	case "Keepalive":
		h.logger.Debug("GB28181 Keepalive",
			"device_id", deviceID,
			"status", notify.Status,
			"sn", notify.SN,
		)
		if notify.Status != "" && notify.Status != "OK" && notify.Status != "NORMAL" {
			if dev, ok := h.stateManager.Get(deviceID); ok {
				dev.LastError = fmt.Sprintf("keepalive: %s", notify.Status)
				h.stateManager.Set(dev)
			}
		}

	case "Alarm":
		priority := models.AlarmPriority(notify.AlarmPriority)
		if priority == 0 {
			priority = models.AlarmPriorityMedium
		}

		desc := notify.AlarmDescription
		if desc == "" {
			desc = fmt.Sprintf("GB28181 Alarm: method=%d, priority=%d", notify.AlarmMethod, notify.AlarmPriority)
		}

		alarm := &models.Alarm{
			DeviceID:    deviceID,
			Priority:    priority,
			Method:      models.AlarmMethod(notify.AlarmMethod),
			Timestamp:   time.Now(),
			Description: desc,
		}
		h.stateManager.AddAlarm(deviceID, alarm)
		h.logger.Info("GB28181 Alarm",
			"device_id", deviceID,
			"priority", notify.AlarmPriority,
			"method", notify.AlarmMethod,
		)

	case "MobilePosition":
		h.logger.Debug("GB28181 MobilePosition",
			"device_id", deviceID,
			"lon", notify.Longitude,
			"lat", notify.Latitude,
		)

	default:
		h.logger.Debug("GB28181 Notify (unhandled)",
			"device_id", deviceID,
			"cmd_type", notify.CmdType,
		)
	}
}

func (h *SIPHandler) handleKeepaliveFallback(deviceID string, k *KeepaliveXML) {
	h.logger.Debug("Keepalive (fallback)", "device_id", deviceID, "status", k.Status)
	if k.Status != "" && k.Status != "OK" {
		if dev, ok := h.stateManager.Get(deviceID); ok {
			dev.LastError = fmt.Sprintf("keepalive: %s", k.Status)
			h.stateManager.Set(dev)
		}
	}
}

func (h *SIPHandler) handleAlarmFallback(deviceID string, a *AlarmXML) {
	h.logger.Info("Alarm (fallback)", "device_id", deviceID, "method", a.AlarmMethod)
	alarm := &models.Alarm{
		DeviceID:    deviceID,
		Priority:    models.AlarmPriority(a.AlarmPriority),
		Method:      models.AlarmMethod(a.AlarmMethod),
		Timestamp:   time.Now(),
		Description: a.AlarmDescription,
	}
	h.stateManager.AddAlarm(deviceID, alarm)
}

// ═══════════════════════════════════════════════════════════════════════
// NOTIFY / SUBSCRIBE / INVITE
// ═══════════════════════════════════════════════════════════════════════

func (h *SIPHandler) handleNotify(msg *SIPMessage, remote *net.UDPAddr) {
	h.sendResponse(remote, msg, 200, "OK", nil)
	// NOTIFY содержит те же данные, что MESSAGE — переиспользуем обработку
	deviceID := h.extractDeviceID(msg)
	if deviceID != "" {
		h.stateManager.UpdateLastSeen(deviceID)
		h.stateManager.SetOnline(deviceID)
	}
}

func (h *SIPHandler) handleInvite(msg *SIPMessage, remote *net.UDPAddr) {
	// INVITE — запрос на видеопоток. Мы не медиа-сервер — отклоняем.
	deviceID := h.extractDeviceID(msg)
	h.logger.Info("INVITE received (media request, declining)",
		"device_id", deviceID,
		"remote", remote.String(),
	)
	h.sendResponse(remote, msg, 488, "Not Acceptable Here", nil)
}

func (h *SIPHandler) handleSubscribe(msg *SIPMessage, remote *net.UDPAddr) {
	h.sendResponse(remote, msg, 200, "OK", map[string]string{
		"Expires": "3600",
	})
}

// ═══════════════════════════════════════════════════════════════════════
// Response Handler (ответы на наши запросы)
// ═══════════════════════════════════════════════════════════════════════

func (h *SIPHandler) handleResponse(msg *SIPMessage, _ *net.UDPAddr) {
	if msg.StatusCode >= 400 {
		h.logger.Warn("SIP error response",
			"status", msg.StatusCode,
			"text", msg.StatusText,
		)
		return
	}

	body := decodeBody([]byte(msg.Body))
	if body == "" {
		return
	}

	var resp GB28181Response
	if err := xml.Unmarshal([]byte(body), &resp); err == nil && resp.CmdType != "" {
		h.handleGB28181Response(&resp)
	}
}

func (h *SIPHandler) handleGB28181Response(resp *GB28181Response) {
	switch resp.CmdType {
	case "Catalog":
		h.handleCatalogResponse(resp)
	case "DeviceInfo":
		h.handleDeviceInfoResponse(resp)
	case "DeviceStatus":
		h.logger.Debug("GB28181 DeviceStatus",
			"device_id", resp.DeviceID,
		)
	default:
		h.logger.Debug("GB28181 Response",
			"cmd_type", resp.CmdType,
			"device_id", resp.DeviceID,
		)
	}
}

func (h *SIPHandler) handleCatalogResponse(resp *GB28181Response) {
	h.logger.Info("GB28181 Catalog response",
		"device_id", resp.DeviceID,
		"expected_num", resp.Num,
		"received_items", len(resp.DeviceList),
	)

	for _, item := range resp.DeviceList {
		childID := item.DeviceID
		if childID == "" {
			continue
		}

		status := models.StatusOnline
		switch strings.ToUpper(item.Status) {
		case "OFF", "VLOST", "FAULT":
			status = models.StatusOffline
		}

		if dev, ok := h.stateManager.Get(childID); !ok {
			dev = &models.Device{
				DeviceID:     childID,
				Status:       status,
				LastSeen:     time.Now(),
				RegisteredAt: time.Now(),
				VendorType:   item.Manufacturer,
				Name:         item.Name,
				Location:     item.IPAddress,
			}
			h.stateManager.Set(dev)
			h.logger.Info("Catalog child registered",
				"device_id", childID,
				"name", item.Name,
				"manufacturer", item.Manufacturer,
				"status", item.Status,
			)
		} else {
			dev.LastSeen = time.Now()
			dev.Status = status
			if item.Name != "" {
				dev.Name = item.Name
			}
			if item.Manufacturer != "" {
				dev.VendorType = item.Manufacturer
			}
			if item.IPAddress != "" {
				dev.Location = item.IPAddress
			}
			h.stateManager.Set(dev)
		}
	}
}

func (h *SIPHandler) handleDeviceInfoResponse(resp *GB28181Response) {
	h.logger.Info("GB28181 DeviceInfo",
		"device_id", resp.DeviceID,
		"name", resp.DeviceName,
		"manufacturer", resp.Manufacturer,
		"model", resp.Model,
		"firmware", resp.Firmware,
	)

	if dev, ok := h.stateManager.Get(resp.DeviceID); ok {
		if resp.DeviceName != "" {
			dev.Name = resp.DeviceName
		}
		if resp.Manufacturer != "" {
			dev.VendorType = resp.Manufacturer
		}
		h.stateManager.Set(dev)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Outbound Requests (Catalog, DeviceInfo)
// ═══════════════════════════════════════════════════════════════════════

// PTZCommand — команда PTZ для GB28181
type PTZCommand struct {
	Action string
	Speed  int
}

// RequestCatalog — экспортируемый метод запроса каталога
func (h *SIPHandler) RequestCatalog(deviceID string) error {
	dev, ok := h.stateManager.Get(deviceID)
	if !ok || dev.ContactAddr == nil {
		return fmt.Errorf("device %s not found or no contact address", deviceID)
	}
	h.requestCatalog(deviceID)
	return nil
}

// SendPTZCommand — экспортируемый метод отправки PTZ-команды
func (h *SIPHandler) SendPTZCommand(deviceID string, cmd PTZCommand) error {
	dev, ok := h.stateManager.Get(deviceID)
	if !ok || dev.ContactAddr == nil {
		return fmt.Errorf("device %s not found or no contact address", deviceID)
	}

	sn := int(atomic.AddUint64(&h.snCounter, 1))
	ptzXML := fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Control>
<CmdType>DeviceControl</CmdType>
<SN>%d</SN>
<DeviceID>%s</DeviceID>
<PTZCmd>%s %d</PTZCmd>
</Control>`, sn, deviceID, cmd.Action, cmd.Speed)

	h.sendSIPRequest("MESSAGE", deviceID, dev.ContactAddr, sn, "Application/MANSCDP+xml", ptzXML)
	h.logger.Info("PTZ command sent", "device_id", deviceID, "action", cmd.Action, "sn", sn)
	return nil
}

func (h *SIPHandler) requestCatalog(deviceID string) {
	dev, ok := h.stateManager.Get(deviceID)
	if !ok || dev.ContactAddr == nil {
		return
	}

	sn := int(atomic.AddUint64(&h.snCounter, 1))
	queryXML := fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>Catalog</CmdType>
<SN>%d</SN>
<DeviceID>%s</DeviceID>
</Query>`, sn, deviceID)

	h.sendSIPRequest("MESSAGE", deviceID, dev.ContactAddr, sn, "Application/MANSCDP+xml", queryXML)
	h.logger.Info("Catalog request sent", "device_id", deviceID, "sn", sn)
}

func (h *SIPHandler) requestDeviceInfo(deviceID string) {
	dev, ok := h.stateManager.Get(deviceID)
	if !ok || dev.ContactAddr == nil {
		return
	}

	sn := int(atomic.AddUint64(&h.snCounter, 1))
	queryXML := fmt.Sprintf(`<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>DeviceInfo</CmdType>
<SN>%d</SN>
<DeviceID>%s</DeviceID>
</Query>`, sn, deviceID)

	h.sendSIPRequest("MESSAGE", deviceID, dev.ContactAddr, sn, "Application/MANSCDP+xml", queryXML)
}

func (h *SIPHandler) sendSIPRequest(method, deviceID string, contact *net.UDPAddr, sn int, contentType, body string) {
	callID := generateCallID(h.host)
	branch := generateBranch()
	tag := generateTag()

	msg := fmt.Sprintf("%s sip:%s@%s SIP/2.0\r\n"+
		"Via: SIP/2.0/UDP %s:%d;branch=%s\r\n"+
		"From: <sip:%s@%s>;tag=%s\r\n"+
		"To: <sip:%s@%s>\r\n"+
		"Call-ID: %s\r\n"+
		"CSeq: %d %s\r\n"+
		"Content-Type: %s\r\n"+
		"Max-Forwards: 70\r\n"+
		"User-Agent: GB-Telemetry-Collector/1.0\r\n"+
		"Content-Length: %d\r\n"+
		"\r\n"+
		"%s",
		method, deviceID, h.host,
		h.host, h.port, branch,
		h.host, h.host, tag,
		deviceID, h.host,
		callID,
		sn, method,
		contentType,
		len(body),
		body,
	)

	_, err := h.conn.WriteToUDP([]byte(msg), contact)
	if err != nil {
		h.logger.Error("Failed to send SIP request",
			"method", method,
			"device_id", deviceID,
			"error", err,
		)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Send SIP Response (RFC 3261 compliant)
// ═══════════════════════════════════════════════════════════════════════

func (h *SIPHandler) sendResponse(remote *net.UDPAddr, req *SIPMessage, statusCode int, statusText string, extraHeaders map[string]string) {
	var resp strings.Builder
	resp.WriteString(fmt.Sprintf("SIP/2.0 %d %s\r\n", statusCode, statusText))

	// Копируем ВСЕ Via (RFC 3261 §8.2.6.2) + добавляем received
	for _, via := range req.getHeaderAll("Via") {
		if !strings.Contains(via, "received=") {
			via += ";received=" + remote.IP.String()
		}
		resp.WriteString(fmt.Sprintf("Via: %s\r\n", via))
	}

	// From — копируем как есть
	if from := req.getHeader("From"); from != "" {
		resp.WriteString(fmt.Sprintf("From: %s\r\n", from))
	}

	// To — добавляем тег если нет (RFC 3261 §8.2.6.4)
	if to := req.getHeader("To"); to != "" {
		if !strings.Contains(to, "tag=") {
			to += ";tag=" + generateTag()
		}
		resp.WriteString(fmt.Sprintf("To: %s\r\n", to))
	}

	// Call-ID — копируем
	if callID := req.getHeader("Call-ID"); callID != "" {
		resp.WriteString(fmt.Sprintf("Call-ID: %s\r\n", callID))
	}

	// CSeq — копируем
	if cseq := req.getHeader("CSeq"); cseq != "" {
		resp.WriteString(fmt.Sprintf("CSeq: %s\r\n", cseq))
	}

	// Contact — наш адрес
	resp.WriteString(fmt.Sprintf("Contact: <sip:%s:%d>\r\n", h.host, h.port))
	resp.WriteString("User-Agent: GB-Telemetry-Collector/1.0\r\n")

	// Дополнительные заголовки
	for k, v := range extraHeaders {
		resp.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	resp.WriteString("Content-Length: 0\r\n\r\n")

	if _, err := h.conn.WriteToUDP([]byte(resp.String()), remote); err != nil {
		h.logger.Error("Failed to send SIP response",
			"remote", remote.String(),
			"status", statusCode,
			"error", err,
		)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Extract DeviceID
// ═══════════════════════════════════════════════════════════════════════

func (h *SIPHandler) extractDeviceID(msg *SIPMessage) string {
	// Ищем в From, To, URI, Contact — в порядке приоритета
	for _, hdr := range []string{"From", "To"} {
		if val := msg.getHeader(hdr); val != "" {
			if id := extractSIPURIUser(val); id != "" {
				return id
			}
		}
	}
	if msg.URI != "" {
		if id := extractSIPURIUser(msg.URI); id != "" {
			return id
		}
	}
	if contact := msg.getHeader("Contact"); contact != "" {
		if id := extractSIPURIUser(contact); id != "" {
			return id
		}
	}
	return ""
}

// extractSIPURIUser извлекает user-часть из SIP URI
// Пример: <sip:34020000001310000001@192.168.1.100:5060> → "34020000001310000001"
func extractSIPURIUser(val string) string {
	start := strings.Index(val, "sip:")
	if start == -1 {
		return ""
	}
	rest := val[start+4:] // после "sip:"

	// Ищем конец user-части: @, >, ;, пробел
	end := strings.IndexAny(rest, "@>; \t")
	if end == -1 {
		return rest
	}
	return rest[:end]
}

// ═══════════════════════════════════════════════════════════════════════
// Extract Contact Address (с NAT-traversal)
// ═══════════════════════════════════════════════════════════════════════

func (h *SIPHandler) extractContactAddr(msg *SIPMessage, remote *net.UDPAddr) *net.UDPAddr {
	contactStr := msg.getHeader("Contact")
	if contactStr == "" {
		return remote
	}

	// Ищем host:port в Contact URI
	start := strings.Index(contactStr, "sip:")
	if start == -1 {
		return remote
	}
	rest := contactStr[start+4:]

	// Убираем user@ если есть
	if at := strings.Index(rest, "@"); at != -1 {
		rest = rest[at+1:]
	}

	// Обрезаем на >, ; или пробел
	if end := strings.IndexAny(rest, ">; \t"); end != -1 {
		rest = rest[:end]
	}

	host, portStr, err := net.SplitHostPort(rest)
	if err != nil {
		// Может быть IP без порта
		ip := net.ParseIP(rest)
		if ip != nil {
			return &net.UDPAddr{IP: ip, Port: remote.Port}
		}
		return remote
	}

	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = 5060
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return remote
	}

	// NAT traversal: если IP частный, используем remote IP
	if isPrivateIP(ip) {
		return &net.UDPAddr{IP: remote.IP, Port: port}
	}

	return &net.UDPAddr{IP: ip, Port: port}
}

// ═══════════════════════════════════════════════════════════════════════
// Vendor Detection
// ═══════════════════════════════════════════════════════════════════════

func (h *SIPHandler) detectVendor(info *GB28181DeviceID) string {
	if !info.IsValid {
		return "sip_unknown"
	}
	switch {
	case info.IsCamera():
		return "gb28181_camera"
	case info.IsNVR():
		return "gb28181_nvr"
	case info.IsPlatform():
		return "gb28181_platform"
	default:
		return "gb28181_device"
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Legacy structures (обратная совместимость с не-GB28181 устройствами)
// ═══════════════════════════════════════════════════════════════════════

type KeepaliveXML struct {
	XMLName  xml.Name `xml:"Keepalive"`
	DeviceID string   `xml:"DeviceID"`
	Status   string   `xml:"Status"`
}

type AlarmXML struct {
	XMLName          xml.Name `xml:"Alarm"`
	DeviceID         string   `xml:"DeviceID"`
	AlarmPriority    int      `xml:"AlarmPriority"`
	AlarmMethod      int      `xml:"AlarmMethod"`
	AlarmTime        string   `xml:"AlarmTime"`
	AlarmDescription string   `xml:"AlarmDescription,omitempty"`
}
