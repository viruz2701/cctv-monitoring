package protocols

import (
    "context"
    "encoding/base64"
    "encoding/xml"
    "fmt"
    "gb-telemetry-collector/internal/models"
    "gb-telemetry-collector/internal/state"
    "io"
    "log/slog"
    "mime"
    "mime/multipart"
    
    "net/http"
    "net/textproto"
    
    "strings"
    "sync"
    "time"

    "github.com/icholy/digest"
)

type HikvisionHandler struct {
    cameras   []HikCameraConfig
    stateMgr  state.DeviceStateManager
    logger    *slog.Logger
    wg        sync.WaitGroup
    ctx       context.Context
    cancel    context.CancelFunc
}

type HikCameraConfig struct {
    Name      string
    Address   string
    HTTPS     bool
    Username  string
    Password  string
    RawTCP    bool // для камер с битым HTTP
    Channel   string // опционально
}

type HikEvent struct {
    CameraName  string
    EventType   string
    Description string
}

// XML-структура события
type HikXmlEvent struct {
    XMLName     xml.Name `xml:"EventNotificationAlert"`
    EventType   string   `xml:"eventType"`
    EventState  string   `xml:"eventState"`
    Description string   `xml:"eventDescription"`
    ChannelID   int      `xml:"channelID"`
    Active      bool     // внутренний флаг
}

func NewHikvisionHandler(cameras []HikCameraConfig, stateMgr state.DeviceStateManager, logger *slog.Logger) *HikvisionHandler {
    return &HikvisionHandler{
        cameras:  cameras,
        stateMgr: stateMgr,
        logger:   logger,
    }
}

func (h *HikvisionHandler) Start(ctx context.Context) error {
    h.ctx, h.cancel = context.WithCancel(ctx)
    for _, cam := range h.cameras {
        h.wg.Add(1)
        go h.runCamera(cam)
    }
    h.logger.Info("Hikvision handler started", "cameras", len(h.cameras))
    return nil
}

func (h *HikvisionHandler) Stop() error {
    h.cancel()
    h.wg.Wait()
    return nil
}

func (h *HikvisionHandler) runCamera(cfg HikCameraConfig) {
    defer h.wg.Done()
    deviceID := fmt.Sprintf("hikvision_%s", cfg.Name)

    // Регистрируем устройство в stateManager (если ещё нет)
    if _, ok := h.stateMgr.Get(deviceID); !ok {
        dev := &models.Device{
            DeviceID:     deviceID,
            Status:       models.StatusOnline,
            LastSeen:     time.Now(),
            RegisteredAt: time.Now(),
            VendorType:   "hikvision",
            Name:         cfg.Name,
            Location:     cfg.Address,
        }
        h.stateMgr.Set(dev)
    }

    for {
        select {
        case <-h.ctx.Done():
            h.logger.Debug("Hikvision camera loop stopped", "camera", cfg.Name)
            return
        default:
        }

        if cfg.RawTCP {
            h.readEventsRawTCP(cfg, deviceID)
        } else {
            h.readEventsHTTP(cfg, deviceID)
        }

        // При разрыве соединения ждём перед переподключением
        time.Sleep(5 * time.Second)
        h.logger.Info("Reconnecting to Hikvision camera", "camera", cfg.Name)
    }
}

// HTTP Multipart streaming (стандартный способ)
func (h *HikvisionHandler) readEventsHTTP(cfg HikCameraConfig, deviceID string) {
    scheme := "http"
    if cfg.HTTPS {
        scheme = "https"
    }
    baseURL := fmt.Sprintf("%s://%s/ISAPI/Event/notification/alertStream", scheme, cfg.Address)

    client := &http.Client{}
    // Пробуем Digest аутентификацию
    client.Transport = &digest.Transport{
        Username: cfg.Username,
        Password: cfg.Password,
    }

    req, err := http.NewRequest("GET", baseURL, nil)
    if err != nil {
        h.logger.Error("Hikvision request creation failed", "camera", cfg.Name, "error", err)
        return
    }
    req.SetBasicAuth(cfg.Username, cfg.Password) // сначала Basic, если не пройдёт – Digest сам подхватит

    resp, err := client.Do(req)
    if err != nil {
        h.logger.Error("Hikvision HTTP connection failed", "camera", cfg.Name, "error", err)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        h.logger.Warn("Hikvision bad status", "camera", cfg.Name, "status", resp.StatusCode)
        return
    }

    mediaType, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
    if err != nil || mediaType != "multipart/mixed" || params["boundary"] == "" {
        h.logger.Warn("Hikvision not multipart", "camera", cfg.Name, "content-type", resp.Header.Get("Content-Type"))
        return
    }

    multipartReader := multipart.NewReader(resp.Body, params["boundary"])
    var xmlEvent HikXmlEvent

    for {
        part, err := multipartReader.NextPart()
        if err == io.EOF {
            break
        }
        if err != nil {
            h.logger.Error("Hikvision multipart read error", "camera", cfg.Name, "error", err)
            break
        }

        body, err := io.ReadAll(part)
        if err != nil {
            continue
        }

        if err := xml.Unmarshal(body, &xmlEvent); err != nil {
            h.logger.Debug("Hikvision XML unmarshal error", "camera", cfg.Name, "error", err)
            continue
        }

        // Обновляем LastSeen
        h.stateMgr.UpdateLastSeen(deviceID)

        if xmlEvent.EventState == "active" && !xmlEvent.Active {
            h.logger.Info("Hikvision alarm", "camera", cfg.Name, "event", xmlEvent.EventType)
            // Формируем Alarm
            alarm := &models.Alarm{
                DeviceID:    deviceID,
                Priority:    h.mapPriority(xmlEvent.EventType),
                Method:      models.AlarmMethodMotionDetection,
                Timestamp:   time.Now(),
                Description: fmt.Sprintf("%s: %s", xmlEvent.EventType, xmlEvent.Description),
            }
            h.stateMgr.AddAlarm(deviceID, alarm)
            xmlEvent.Active = true
        } else if xmlEvent.EventState == "inactive" {
            xmlEvent.Active = false
        }
    }
}

// Raw TCP для камер с битым HTTP (например, дверные звонки)
func (h *HikvisionHandler) readEventsRawTCP(cfg HikCameraConfig, deviceID string) {
    host := cfg.Address
    if !strings.Contains(host, ":") {
        host += ":80"
    }
    basicAuth := base64.StdEncoding.EncodeToString([]byte(cfg.Username + ":" + cfg.Password))

    conn, err := textproto.Dial("tcp", host)
    if err != nil {
        h.logger.Error("Hikvision TCP dial failed", "camera", cfg.Name, "error", err)
        return
    }
    defer conn.Close()

    // Отправляем HTTP-запрос вручную
    req := fmt.Sprintf("GET /ISAPI/Event/notification/alertStream HTTP/1.1\r\n"+
        "Host: %s\r\n"+
        "Authorization: Basic %s\r\n"+
        "\r\n", cfg.Address, basicAuth)

    if _, err := conn.Cmd(req); err != nil {
        h.logger.Error("Hikvision TCP write failed", "camera", cfg.Name, "error", err)
        return
    }

    // Читаем HTTP-статус
    line, err := conn.ReadLine()
    if err != nil || !strings.Contains(line, "200 OK") {
        h.logger.Warn("Hikvision TCP bad response", "camera", cfg.Name, "line", line)
        return
    }

    // Пропускаем заголовки
    for {
        line, err := conn.ReadLine()
        if err != nil || line == "" {
            break
        }
    }

    // Читаем события (простейший парсинг: каждое событие заканчивается пустой строкой)
    var eventXML string
    for {
        line, err := conn.ReadLine()
        if err != nil {
            break
        }
        if line == "" && eventXML != "" {
            // Обрабатываем накопленный XML
            var xmlEvent HikXmlEvent
            if err := xml.Unmarshal([]byte(eventXML), &xmlEvent); err == nil {
                h.stateMgr.UpdateLastSeen(deviceID)
                if xmlEvent.EventState == "active" {
                    alarm := &models.Alarm{
                        DeviceID:    deviceID,
                        Priority:    h.mapPriority(xmlEvent.EventType),
                        Method:      models.AlarmMethodMotionDetection,
                        Timestamp:   time.Now(),
                        Description: fmt.Sprintf("%s: %s", xmlEvent.EventType, xmlEvent.Description),
                    }
                    h.stateMgr.AddAlarm(deviceID, alarm)
                    h.logger.Info("Hikvision raw TCP alarm", "camera", cfg.Name, "event", xmlEvent.EventType)
                }
            }
            eventXML = ""
        } else {
            eventXML += line
        }
    }
}

func (h *HikvisionHandler) mapPriority(eventType string) models.AlarmPriority {
    switch {
    case strings.Contains(eventType, "Motion"):
        return models.AlarmPriorityHigh
    case strings.Contains(eventType, "VideoLoss"), strings.Contains(eventType, "Tamper"):
        return models.AlarmPriorityHigh
    case strings.Contains(eventType, "Storage"), strings.Contains(eventType, "HDD"):
        return models.AlarmPriorityMedium
    default:
        return models.AlarmPriorityLow
    }
}