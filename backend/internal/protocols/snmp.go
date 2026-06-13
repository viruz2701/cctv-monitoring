package protocols

import (
    "context"
    "fmt"
    "log/slog"
    "net"
    "strings"
    "sync"
    "time"

    "gb-telemetry-collector/internal/config"
    "gb-telemetry-collector/internal/models"
    "gb-telemetry-collector/internal/state"

    "github.com/gosnmp/gosnmp"
)

type SNMPHandler struct {
    config   config.SNMPConfig
    stateMgr state.DeviceStateManager
    logger   *slog.Logger
    conn     *net.UDPConn
    wg       sync.WaitGroup
    ctx      context.Context
    cancel   context.CancelFunc
}

func NewSNMPHandler(cfg config.SNMPConfig, stateMgr state.DeviceStateManager, logger *slog.Logger) *SNMPHandler {
    return &SNMPHandler{
        config:   cfg,
        stateMgr: stateMgr,
        logger:   logger,
    }
}

func (h *SNMPHandler) Start(ctx context.Context) error {
    h.ctx, h.cancel = context.WithCancel(ctx)

    addr := &net.UDPAddr{Port: h.config.Port}
    conn, err := net.ListenUDP("udp", addr)
    if err != nil {
        return fmt.Errorf("failed to listen on UDP port %d: %w", h.config.Port, err)
    }
    h.conn = conn

    h.wg.Add(1)
    go h.receiveLoop()
    h.logger.Info("SNMP trap receiver started", "port", h.config.Port, "version", h.config.Version)
    return nil
}

func (h *SNMPHandler) receiveLoop() {
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
            h.logger.Error("SNMP read error", "error", err)
            continue
        }
        go h.handleTrap(buf[:n], remoteAddr)
    }
}

func (h *SNMPHandler) handleTrap(data []byte, addr *net.UDPAddr) {
    // Для версии 1.43.2 используем метод UnmarshalTrap у GoSNMP с двумя аргументами
    g := &gosnmp.GoSNMP{}
    packet, err := g.UnmarshalTrap(data, false) // false — не строгий режим
    if err != nil {
        h.logger.Error("Failed to decode SNMP trap", "error", err, "from", addr.IP)
        return
    }
    if packet == nil {
        h.logger.Warn("Received nil packet after unmarshalling", "from", addr.IP)
        return
    }

    ip := addr.IP.String()
    var serial, hostname, vendor, eventType string

    for _, varbind := range packet.Variables {
        oid := varbind.Name
        strVal := h.valueToString(varbind)

        if serial == "" {
            serial = h.extractSerial(oid, strVal)
        }
        if hostname == "" {
            hostname = h.extractHostname(oid, strVal)
        }
        if vendor == "" {
            vendor = h.detectVendor(oid, strVal)
        }
        if eventType == "" {
            eventType = h.detectEvent(oid, strVal)
        }
    }

    if serial == "" && hostname == "" {
        serial = fmt.Sprintf("snmp_%s", strings.ReplaceAll(ip, ".", "_"))
    }

    deviceID := fmt.Sprintf("snmp_%s", serial)
    if serial == "" && hostname != "" {
        deviceID = fmt.Sprintf("snmp_host_%s", hostname)
    }

    // Обновление или создание устройства
    if dev, ok := h.stateMgr.Get(deviceID); !ok {
        dev = &models.Device{
            DeviceID:     deviceID,
            Status:       models.StatusOnline,
            LastSeen:     time.Now(),
            RegisteredAt: time.Now(),
            VendorType:   vendor,
            Name:         hostname,
            Location:     ip,
        }
        h.stateMgr.Set(dev)
        h.logger.Info("New SNMP device registered", "device_id", deviceID, "vendor", vendor)
    } else {
        h.stateMgr.UpdateLastSeen(deviceID)
        if dev.Status == models.StatusOffline {
            h.stateMgr.SetOnline(deviceID)
        }
    }

    // Создание аларма
    alarm := &models.Alarm{
        DeviceID:    deviceID,
        Priority:    h.mapPriority(eventType),
        Method:      models.AlarmMethodMotionDetection,
        Timestamp:   time.Now(),
        Description: fmt.Sprintf("SNMP trap: %s from %s (%s)", eventType, hostname, ip),
    }
    h.stateMgr.AddAlarm(deviceID, alarm)
    h.logger.Info("SNMP alarm", "device_id", deviceID, "event", eventType)
}

func (h *SNMPHandler) valueToString(varbind gosnmp.SnmpPDU) string {
    switch varbind.Type {
    case gosnmp.OctetString:
        if b, ok := varbind.Value.([]byte); ok {
            return string(b)
        }
    case gosnmp.Integer:
        if i, ok := varbind.Value.(int); ok {
            return fmt.Sprintf("%d", i)
        }
    default:
        if varbind.Value != nil {
            return fmt.Sprintf("%v", varbind.Value)
        }
    }
    return ""
}

func (h *SNMPHandler) extractSerial(oid, value string) string {
    serialOIDs := []string{
        "1.3.6.1.4.1.39165.1.4.0",    // Hikvision
        "1.3.6.1.4.1.1004849.2.1.2.4.0", // Dahua/Intelbras
        "1.3.6.1.4.1.1981.1.1.1.0",   // Dahua
    }
    for _, target := range serialOIDs {
        if strings.Contains(oid, target) && value != "" {
            return strings.Trim(value, "\"")
        }
    }
    return ""
}

func (h *SNMPHandler) extractHostname(oid, value string) string {
    if strings.HasSuffix(oid, ".1.3.6.1.2.1.1.5.0") && value != "" {
        return strings.Trim(value, "\"")
    }
    return ""
}

func (h *SNMPHandler) detectVendor(oid, value string) string {
    if strings.Contains(oid, "39165") {
        return "Hikvision"
    }
    if strings.Contains(oid, "1004849") || strings.Contains(oid, "1981") {
        return "Dahua/Intelbras"
    }
    if strings.HasPrefix(value, "DS-") {
        return "Hikvision"
    }
    if strings.HasPrefix(value, "DH-") {
        return "Dahua"
    }
    return "Unknown"
}

func (h *SNMPHandler) detectEvent(oid, value string) string {
    eventKeywords := []string{"motion", "alarm", "tamper", "video loss", "hdd", "storage"}
    lower := strings.ToLower(value)
    for _, kw := range eventKeywords {
        if strings.Contains(lower, kw) {
            return kw
        }
    }
    return "SNMP Event"
}

func (h *SNMPHandler) mapPriority(event string) models.AlarmPriority {
    switch {
    case strings.Contains(event, "motion"), strings.Contains(event, "tamper"):
        return models.AlarmPriorityHigh
    case strings.Contains(event, "storage"), strings.Contains(event, "hdd"):
        return models.AlarmPriorityMedium
    default:
        return models.AlarmPriorityLow
    }
}

func (h *SNMPHandler) Stop() error {
    h.cancel()
    if h.conn != nil {
        h.conn.Close()
    }
    h.wg.Wait()
    return nil
}