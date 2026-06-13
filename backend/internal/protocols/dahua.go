package protocols

import (
    "context"
    "encoding/binary"
    "fmt"
    "gb-telemetry-collector/internal/models"
    "gb-telemetry-collector/internal/state"
    "log/slog"
    "net"
    
    "strings"
    "sync"
    "time"
)

type DahuaHandler struct {
    ports    []int
    stateMgr state.DeviceStateManager
    logger   *slog.Logger
    servers  []*net.TCPListener
    wg       sync.WaitGroup
    ctx      context.Context
    cancel   context.CancelFunc
}

func NewDahuaHandler(ports []int, stateMgr state.DeviceStateManager, logger *slog.Logger) *DahuaHandler {
    return &DahuaHandler{
        ports:    ports,
        stateMgr: stateMgr,
        logger:   logger,
    }
}

func (h *DahuaHandler) Start(ctx context.Context) error {
    h.ctx, h.cancel = context.WithCancel(ctx)
    for _, port := range h.ports {
        addr := &net.TCPAddr{Port: port}
        listener, err := net.ListenTCP("tcp", addr)
        if err != nil {
            h.logger.Error("Failed to start Dahua listener", "port", port, "error", err)
            continue
        }
        h.servers = append(h.servers, listener)
        h.wg.Add(1)
        go h.acceptLoop(listener, port)
        h.logger.Info("Dahua private protocol server started", "port", port)
    }
    return nil
}

func (h *DahuaHandler) acceptLoop(listener *net.TCPListener, port int) {
    defer h.wg.Done()
    for {
        select {
        case <-h.ctx.Done():
            return
        default:
        }
        conn, err := listener.AcceptTCP()
        if err != nil {
            select {
            case <-h.ctx.Done():
                return
            default:
                h.logger.Error("Dahua accept error", "port", port, "error", err)
                continue
            }
        }
        go h.handleConnection(conn)
    }
}

// parsePacket разбирает пакет приватного протокола Dahua
func (h *DahuaHandler) parsePacket(data []byte) *DahuaEvent {
    if len(data) < 6 {
        return nil
    }
    // Проверяем заголовок 0x12, 0x34
    if data[0] != 0x12 || data[1] != 0x34 {
        return nil
    }
    packetLen := binary.BigEndian.Uint32(data[2:6])
    if len(data) < int(packetLen) {
        return nil
    }
    payload := data[6:packetLen]
    return h.parsePayload(payload)
}

// parsePayload разбирает payload формата key=value&key=value...
func (h *DahuaHandler) parsePayload(payload []byte) *DahuaEvent {
    payloadStr := string(payload)
    params := make(map[string]string)
    for _, item := range strings.Split(payloadStr, "&") {
        parts := strings.SplitN(item, "=", 2)
        if len(parts) == 2 {
            params[parts[0]] = parts[1]
        }
    }
    eventCode := params["Code"]
    if eventCode == "" {
        return nil
    }
    action := params["action"]
    index := params["index"]
    if index == "" {
        index = "0"
    }
    channel := params["channel"]
    if channel == "" {
        channel = index
    }
    data := params["data"]
    timestamp := params["timestamp"]
    if timestamp == "" {
        timestamp = time.Now().Format("20060102150405")
    }

    eventMap := map[string]string{
        "VideoMotion":        "Motion Detection",
        "VideoLoss":          "Video Loss",
        "VideoBlind":         "Video Tampering",
        "AlarmLocal":         "Local Alarm",
        "CrossLineDetection": "Tripwire",
        "RegionDetection":    "Intrusion",
        "FaceDetection":      "Face Detected",
        "HumanDetection":     "Human Detected",
        "VehicleDetection":   "Vehicle Detected",
        "HDDFailure":         "HDD Failure",
        "HDDFull":            "HDD Full",
        "NetworkDisconnect":  "Network Disconnected",
        "TemperatureAlarm":   "Temperature Alarm",
        "FanAlarm":           "Fan Error",
        "StorageFailure":     "Storage Failure",
    }
    eventType := eventMap[eventCode]
    if eventType == "" {
        eventType = eventCode
    }

    return &DahuaEvent{
        EventType: eventType,
        EventCode: eventCode,
        Action:    action,
        Index:     index,
        Channel:   channel,
        Data:      data,
        Timestamp: timestamp,
        Raw:       payloadStr,
    }
}

func (h *DahuaHandler) handleConnection(conn *net.TCPConn) {
    defer conn.Close()
    conn.SetReadDeadline(time.Now().Add(10 * time.Second))
    buf := make([]byte, 8192)
    n, err := conn.Read(buf)
    if err != nil {
        h.logger.Debug("Dahua read error", "error", err)
        return
    }
    data := buf[:n]
    event := h.parsePacket(data)
    if event == nil {
        h.logger.Debug("Failed to parse Dahua packet", "hex", fmt.Sprintf("%x", data))
        return
    }

    remoteAddr := conn.RemoteAddr().(*net.TCPAddr)
    ip := remoteAddr.IP.String()
    cameraName := fmt.Sprintf("Dahua_Private_%s_%s", ip, event.Channel)

    deviceID := fmt.Sprintf("dahua_%s_%s", strings.ReplaceAll(ip, ".", "_"), event.Channel)
    // Регистрируем устройство
    if dev, ok := h.stateMgr.Get(deviceID); !ok {
        dev = &models.Device{
            DeviceID:     deviceID,
            Status:       models.StatusOnline,
            LastSeen:     time.Now(),
            RegisteredAt: time.Now(),
            VendorType:   "dahua",
            Name:         cameraName,
            Location:     ip,
        }
        h.stateMgr.Set(dev)
    } else {
        h.stateMgr.UpdateLastSeen(deviceID)
        if dev.Status == models.StatusOffline {
            h.stateMgr.SetOnline(deviceID)
        }
    }

    // Определяем приоритет
    priority := models.AlarmPriorityLow
    if strings.Contains(event.EventCode, "Motion") || strings.Contains(event.EventCode, "VideoLoss") {
        priority = models.AlarmPriorityHigh
    } else if strings.Contains(event.EventCode, "HDD") || strings.Contains(event.EventCode, "Storage") || strings.Contains(event.EventCode, "Network") {
        priority = models.AlarmPriorityMedium
    }

    alarm := &models.Alarm{
        DeviceID:    deviceID,
        Priority:    priority,
        Method:      models.AlarmMethodMotionDetection, // уточнить
        Timestamp:   time.Now(),
        Description: fmt.Sprintf("%s (%s) on channel %s: %s", event.EventType, event.EventCode, event.Channel, event.Data),
    }
    h.stateMgr.AddAlarm(deviceID, alarm)

    h.logger.Info("Dahua alarm",
        "camera", cameraName,
        "event", event.EventType,
        "event_code", event.EventCode,
        "action", event.Action,
        "channel", event.Channel,
        "ip", ip,
    )
}

func (h *DahuaHandler) Stop() error {
    h.cancel()
    for _, l := range h.servers {
        l.Close()
    }
    h.wg.Wait()
    return nil
}

type DahuaEvent struct {
    EventType string
    EventCode string
    Action    string
    Index     string
    Channel   string
    Data      string
    Timestamp string
    Raw       string
}