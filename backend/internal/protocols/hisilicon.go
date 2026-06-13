package protocols

import (
    "context"
    "encoding/json"
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

type HisiliconHandler struct {
    port      int
    stateMgr  state.DeviceStateManager
    logger    *slog.Logger
    listener  *net.TCPListener
    wg        sync.WaitGroup
    ctx       context.Context
    cancel    context.CancelFunc
}

func NewHisiliconHandler(port int, stateMgr state.DeviceStateManager, logger *slog.Logger) *HisiliconHandler {
    return &HisiliconHandler{
        port:     port,
        stateMgr: stateMgr,
        logger:   logger,
    }
}

func (h *HisiliconHandler) Start(ctx context.Context) error {
    h.ctx, h.cancel = context.WithCancel(ctx)
    addr := &net.TCPAddr{Port: h.port}
    listener, err := net.ListenTCP("tcp", addr)
    if err != nil {
        return err
    }
    h.listener = listener
    h.wg.Add(1)
    go h.acceptLoop()
    h.logger.Info("Hisilicon protocol server started", "port", h.port)
    return nil
}

func (h *HisiliconHandler) acceptLoop() {
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
                h.logger.Error("Hisilicon accept error", "error", err)
                continue
            }
        }
        h.logger.Debug("Hisilicon connection from", "addr", conn.RemoteAddr().String())
        go h.handleConnection(conn)
    }
}

// extractJSONFromBinary находит первый корректный JSON в бинарных данных
func (h *HisiliconHandler) extractJSONFromBinary(data []byte) (map[string]interface{}, error) {
    start := -1
    // Ищем начало JSON - символ '{'
    for i, b := range data {
        if b == '{' {
            start = i
            break
        }
    }
    if start == -1 {
        return nil, fmt.Errorf("no JSON object found")
    }

    braceCount := 0
    end := start
    for i := start; i < len(data); i++ {
        if data[i] == '{' {
            braceCount++
        } else if data[i] == '}' {
            braceCount--
            if braceCount == 0 {
                end = i + 1
                break
            }
        }
    }
    if end <= start {
        return nil, fmt.Errorf("incomplete JSON object")
    }

    jsonStr := data[start:end]
    var result map[string]interface{}
    if err := json.Unmarshal(jsonStr, &result); err != nil {
        return nil, err
    }
    return result, nil
}

// hexToIP конвертирует hex-строку вида "0x1704A8C0" в IP "192.168.4.23"
func (h *HisiliconHandler) hexToIP(hexAddr string) string {
    if hexAddr == "" || hexAddr == "0" {
        return "unknown"
    }
    // Удаляем префикс "0x"
    hexAddr = strings.TrimPrefix(hexAddr, "0x")
    if len(hexAddr) != 8 {
        return "unknown"
    }
    // Разбиваем по 2 символа и переворачиваем порядок (little-endian в IP)
    parts := make([]string, 4)
    for i := 0; i < 4; i++ {
        byteHex := hexAddr[i*2 : i*2+2]
        dec, err := strconv.ParseInt(byteHex, 16, 64)
        if err != nil {
            return "unknown"
        }
        parts[3-i] = strconv.Itoa(int(dec))
    }
    return strings.Join(parts, ".")
}

// eventMapping преобразует сырые типы событий в понятные названия
func (h *HisiliconHandler) eventMapping(rawType string) string {
    mapping := map[string]string{
        "StorageLowSpace":  "Low Storage Space",
        "StorageFailure":   "Storage Failure",
        "MotionDetect":     "Motion Detection",
        "VideoLoss":        "Video Loss",
        "Alarm":            "Alarm",
        "EventStart":       "Event Started",
        "EventStop":        "Event Stopped",
    }
    if mapped, ok := mapping[rawType]; ok {
        return mapped
    }
    return rawType
}

// priorityByEvent определяет приоритет тревоги на основе типа события
func (h *HisiliconHandler) priorityByEvent(eventType string) models.AlarmPriority {
    switch {
    case strings.Contains(eventType, "Motion"), strings.Contains(eventType, "VideoLoss"):
        return models.AlarmPriorityHigh
    case strings.Contains(eventType, "Storage"), strings.Contains(eventType, "Failure"):
        return models.AlarmPriorityMedium
    default:
        return models.AlarmPriorityLow
    }
}

func (h *HisiliconHandler) handleConnection(conn *net.TCPConn) {
    defer conn.Close()
    conn.SetReadDeadline(time.Now().Add(10 * time.Second))

    buf := make([]byte, 4096)
    n, err := conn.Read(buf)
    if err != nil {
        h.logger.Debug("Hisilicon read error", "error", err)
        return
    }
    data := buf[:n]

    // Извлекаем JSON из бинарных данных
    eventData, err := h.extractJSONFromBinary(data)
    if err != nil {
        h.logger.Warn("Hisilicon: no valid JSON found", "addr", conn.RemoteAddr().String(), "error", err)
        // Всё равно отвечаем OK, чтобы камера не повторяла
        conn.Write([]byte("OK"))
        return
    }

    // Извлекаем поля
    serial, _ := eventData["SerialID"].(string)
    if serial == "" {
        serial, _ = eventData["SerialId"].(string)
    }
    if serial == "" {
        serial = "unknown"
    }

    rawEventType, _ := eventData["Event"].(string)
    if rawEventType == "" {
        rawEventType, _ = eventData["event"].(string)
    }

    eventStatus, _ := eventData["Status"].(string)
    description, _ := eventData["Descrip"].(string)
    if description == "" {
        description, _ = eventData["Description"].(string)
    }
    startTime, _ := eventData["StartTime"].(string)
    addressHex, _ := eventData["Address"].(string)

    ipAddr := h.hexToIP(addressHex)
    eventName := h.eventMapping(rawEventType)

    // Формируем deviceID (уникальный идентификатор камеры)
    deviceID := fmt.Sprintf("hisilicon_%s", serial)
    if ipAddr != "unknown" {
        deviceID = fmt.Sprintf("hisilicon_%s_%s", serial, strings.ReplaceAll(ipAddr, ".", "_"))
    }

    // Регистрируем/обновляем устройство в stateManager
    if dev, ok := h.stateMgr.Get(deviceID); !ok {
        dev = &models.Device{
            DeviceID:     deviceID,
            Status:       models.StatusOnline,
            LastSeen:     time.Now(),
            RegisteredAt: time.Now(),
            VendorType:   "hisilicon",
            Name:         fmt.Sprintf("Hisilicon %s", serial),
            Location:     ipAddr,
        }
        h.stateMgr.Set(dev)
    } else {
        h.stateMgr.UpdateLastSeen(deviceID)
        if dev.Status == models.StatusOffline {
            h.stateMgr.SetOnline(deviceID)
        }
    }

    // Формируем и сохраняем Alarm
    alarm := &models.Alarm{
        DeviceID:    deviceID,
        Priority:    h.priorityByEvent(rawEventType),
        Method:      models.AlarmMethodMotionDetection, // можно уточнить по событию
        Timestamp:   time.Now(),
        Description: fmt.Sprintf("%s | Status: %s | Desc: %s | StartTime: %s", eventName, eventStatus, description, startTime),
    }
    h.stateMgr.AddAlarm(deviceID, alarm)

    h.logger.Info("Hisilicon alarm",
        "device_id", deviceID,
        "event", eventName,
        "raw_event", rawEventType,
        "status", eventStatus,
        "ip", ipAddr,
    )

    // Отправляем подтверждение камере
    conn.Write([]byte("OK"))
}

func (h *HisiliconHandler) Stop() error {
    h.cancel()
    if h.listener != nil {
        h.listener.Close()
    }
    h.wg.Wait()
    return nil
}