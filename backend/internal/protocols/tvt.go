package protocols

import (
    "context"
    "encoding/json"
    "encoding/xml"
    "gb-telemetry-collector/internal/models"
    "gb-telemetry-collector/internal/state"
    "log/slog"
    "net"
    "regexp"
    "sync"
    "time"
    "fmt" 
)

type TVTHandler struct {
    port      int
    stateMgr  state.DeviceStateManager
    logger    *slog.Logger
    listener  *net.TCPListener
    wg        sync.WaitGroup
    ctx       context.Context
    cancel    context.CancelFunc
}

func NewTVTHandler(port int, stateMgr state.DeviceStateManager, logger *slog.Logger) *TVTHandler {
    return &TVTHandler{
        port:     port,
        stateMgr: stateMgr,
        logger:   logger,
    }
}

func (h *TVTHandler) Start(ctx context.Context) error {
    h.ctx, h.cancel = context.WithCancel(ctx)
    addr := &net.TCPAddr{Port: h.port}
    listener, err := net.ListenTCP("tcp", addr)
    if err != nil {
        return err
    }
    h.listener = listener
    h.wg.Add(1)
    go h.acceptLoop()
    h.logger.Info("TVT protocol server started", "port", h.port)
    return nil
}

func (h *TVTHandler) acceptLoop() {
    defer h.wg.Done()
    for {
        select {
        case <-h.ctx.Done():
            return
        default:
        }
        conn, err := h.listener.AcceptTCP()
        if err != nil {
            select {
            case <-h.ctx.Done():
                return
            default:
                h.logger.Error("TVT accept error", "error", err)
                continue
            }
        }
        go h.handleConnection(conn)
    }
}

func (h *TVTHandler) handleConnection(conn *net.TCPConn) {
    defer conn.Close()
    conn.SetReadDeadline(time.Now().Add(10 * time.Second))
    buf := make([]byte, 4096)
    n, err := conn.Read(buf)
    if err != nil {
        return
    }
    data := buf[:n]
    // Пытаемся распарсить XML
    var eventType, serial string
    if bytesHasPrefix(data, []byte("<")) {
        var xmlRoot struct {
            EventType string `xml:"EventType"`
            DeviceID  string `xml:"DeviceID"`
            SerialNo  string `xml:"SerialNo"`
        }
        if err := xml.Unmarshal(data, &xmlRoot); err == nil && xmlRoot.EventType != "" {
            eventType = xmlRoot.EventType
            serial = xmlRoot.DeviceID
            if serial == "" {
                serial = xmlRoot.SerialNo
            }
        }
    }
    if eventType == "" {
        // пробуем JSON
        var jsonData map[string]interface{}
        if err := json.Unmarshal(data, &jsonData); err == nil {
            if et, ok := jsonData["Event"].(string); ok {
                eventType = et
            } else if et, ok := jsonData["event"].(string); ok {
                eventType = et
            }
            if s, ok := jsonData["SerialID"].(string); ok {
                serial = s
            } else if s, ok := jsonData["serial"].(string); ok {
                serial = s
            }
        }
    }
    if eventType == "" {
        // поиск ASCII строк
        ascii := regexp.MustCompile(`[A-Za-z0-9_\-]{8,}`).FindAll(data, -1)
        if len(ascii) > 0 {
            eventType = string(ascii[0])
        }
    }
    if eventType == "" {
        h.logger.Debug("TVT unrecognized", "hex", string(data))
        return
    }
    deviceID := fmt.Sprintf("tvt_%s_%s", serial, conn.RemoteAddr().(*net.TCPAddr).IP.String())
    if dev, ok := h.stateMgr.Get(deviceID); !ok {
        dev = &models.Device{
            DeviceID:     deviceID,
            Status:       models.StatusOnline,
            LastSeen:     time.Now(),
            RegisteredAt: time.Now(),
            VendorType:   "tvt",
            Name:         fmt.Sprintf("TVT %s", serial),
            Location:     conn.RemoteAddr().(*net.TCPAddr).IP.String(),
        }
        h.stateMgr.Set(dev)
    } else {
        h.stateMgr.UpdateLastSeen(deviceID)
    }
    alarm := &models.Alarm{
        DeviceID:    deviceID,
        Priority:    models.AlarmPriorityMedium,
        Method:      models.AlarmMethodMotionDetection,
        Timestamp:   time.Now(),
        Description: eventType,
    }
    h.stateMgr.AddAlarm(deviceID, alarm)
    h.logger.Info("TVT alarm", "device_id", deviceID, "event", eventType)
}

func bytesHasPrefix(b []byte, prefix []byte) bool {
    if len(b) < len(prefix) {
        return false
    }
    for i := 0; i < len(prefix); i++ {
        if b[i] != prefix[i] {
            return false
        }
    }
    return true
}

func (h *TVTHandler) Stop() error {
    h.cancel()
    if h.listener != nil {
        h.listener.Close()
    }
    h.wg.Wait()
    return nil
}