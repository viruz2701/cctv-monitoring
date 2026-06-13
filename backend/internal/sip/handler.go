package sip

import (
    "context"
    "encoding/xml"
    "fmt"
    "gb-telemetry-collector/internal/models"
    "gb-telemetry-collector/internal/state"
    "log/slog"
    "net"
    "strconv"
    "strings"
    "sync"
    "time"
)

// Структуры для парсинга XML
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

type SIPHandler struct {
    stateManager state.DeviceStateManager
    logger       *slog.Logger
    conn         *net.UDPConn
    host         string
    port         int
    wg           sync.WaitGroup
    ctx          context.Context
    cancel       context.CancelFunc
}

func NewSIPHandler(stateMgr state.DeviceStateManager, logger *slog.Logger, host string, port int) *SIPHandler {
    return &SIPHandler{
        stateManager: stateMgr,
        logger:       logger,
        host:         host,
        port:         port,
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

    h.logger.Info("SIP UDP server started", "addr", addr)
    return nil
}

func (h *SIPHandler) Stop(ctx context.Context) error {
    if h.cancel != nil {
        h.cancel()
    }
    if h.conn != nil {
        h.conn.Close()
    }
    h.wg.Wait()
    return nil
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
        n, remoteAddr, err := h.conn.ReadFromUDP(buf)
        if err != nil {
            select {
            case <-h.ctx.Done():
                return
            default:
                h.logger.Error("UDP read error", "error", err)
                continue
            }
        }
        data := buf[:n]
        go h.processMessage(data, remoteAddr)
    }
}

func (h *SIPHandler) processMessage(data []byte, remote *net.UDPAddr) {
    msg := string(data)
    lines := strings.Split(msg, "\r\n")
    if len(lines) == 0 {
        return
    }
    requestLine := lines[0]
    parts := strings.Fields(requestLine)
    if len(parts) < 2 {
        return
    }
    method := parts[0]

    headers := make(map[string]string)
    var bodyStart int
    for i, line := range lines[1:] {
        if line == "" {
            bodyStart = i + 2 // после пустой строки
            break
        }
        if colon := strings.Index(line, ":"); colon != -1 {
            key := strings.TrimSpace(line[:colon])
            val := strings.TrimSpace(line[colon+1:])
            headers[key] = val
        }
    }
    body := ""
    if bodyStart < len(lines) {
        body = strings.Join(lines[bodyStart:], "\r\n")
    }

    switch method {
    case "REGISTER":
        h.handleRegister(headers, remote)
        response := "SIP/2.0 200 OK\r\n" +
            "Via: " + headers["Via"] + "\r\n" +
            "From: " + headers["From"] + "\r\n" +
            "To: " + headers["To"] + "\r\n" +
            "Call-ID: " + headers["Call-ID"] + "\r\n" +
            "CSeq: " + headers["CSeq"] + "\r\n" +
            "Contact: " + headers["Contact"] + "\r\n" +
            "Content-Length: 0\r\n\r\n"
        h.conn.WriteToUDP([]byte(response), remote)

    case "MESSAGE":
        deviceID := extractDeviceIDFromHeaders(headers)
        if deviceID == "" {
            h.logger.Warn("MESSAGE without device ID")
        } else {
            h.stateManager.UpdateLastSeen(deviceID)
            h.stateManager.SetOnline(deviceID)
            if strings.Contains(body, "Keepalive") {
                var k KeepaliveXML
                if err := xml.Unmarshal([]byte(body), &k); err == nil {
                    h.logger.Debug("Keepalive", "device_id", deviceID, "status", k.Status)
                    if k.Status != "OK" {
                        if dev, ok := h.stateManager.Get(deviceID); ok {
                            dev.LastError = fmt.Sprintf("keepalive status: %s", k.Status)
                            h.stateManager.Set(dev)
                        }
                    }
                }
            } else if strings.Contains(body, "Alarm") {
                var a AlarmXML
                if err := xml.Unmarshal([]byte(body), &a); err == nil {
                    alarm := &models.Alarm{
                        DeviceID:    deviceID,
                        Priority:    models.AlarmPriority(a.AlarmPriority),
                        Method:      models.AlarmMethod(a.AlarmMethod),
                        Timestamp:   time.Now(),
                        Description: a.AlarmDescription,
                    }
                    h.stateManager.AddAlarm(deviceID, alarm)
                    h.logger.Info("Alarm", "device_id", deviceID, "method", a.AlarmMethod)
                }
            }
        }
        response := "SIP/2.0 200 OK\r\n" +
            "Via: " + headers["Via"] + "\r\n" +
            "From: " + headers["From"] + "\r\n" +
            "To: " + headers["To"] + "\r\n" +
            "Call-ID: " + headers["Call-ID"] + "\r\n" +
            "CSeq: " + headers["CSeq"] + "\r\n" +
            "Content-Length: 0\r\n\r\n"
        h.conn.WriteToUDP([]byte(response), remote)
    }
}

func extractDeviceIDFromHeaders(headers map[string]string) string {
    for _, h := range []string{"From", "To"} {
        if val, ok := headers[h]; ok {
            start := strings.Index(val, "<sip:")
            if start == -1 {
                start = strings.Index(val, "sip:")
            }
            if start != -1 {
                rest := val[start+4:]
                end := strings.IndexAny(rest, "@>")
                if end != -1 {
                    return rest[:end]
                }
            }
        }
    }
    return ""
}

func (h *SIPHandler) handleRegister(headers map[string]string, remote *net.UDPAddr) {
    deviceID := extractDeviceIDFromHeaders(headers)
    if deviceID == "" {
        h.logger.Warn("REGISTER without device ID")
        return
    }
    expires := 3600
    if expStr, ok := headers["Expires"]; ok {
        if e, err := strconv.Atoi(expStr); err == nil {
            expires = e
        }
    }
    userAgent := headers["User-Agent"]

    contactAddr := remote
    if contactStr, ok := headers["Contact"]; ok {
        start := strings.Index(contactStr, "<sip:")
        if start != -1 {
            rest := contactStr[start+5:]
            end := strings.IndexAny(rest, ">")
            if end != -1 {
                hostPort := rest[:end]
                if h, p, err := net.SplitHostPort(hostPort); err == nil {
                    port, _ := strconv.Atoi(p)
                    contactAddr = &net.UDPAddr{IP: net.ParseIP(h), Port: port}
                } else {
                    contactAddr = &net.UDPAddr{IP: net.ParseIP(hostPort), Port: 5060}
                }
            }
        }
    }

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
        }
        h.logger.Info("New device registered", "device_id", deviceID, "contact", contactAddr)
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
}
