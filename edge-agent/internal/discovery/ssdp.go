package discovery

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"time"
)

// SSDPDiscovery discovers UPnP/CCTV devices via SSDP (Simple Service Discovery Protocol).
//
// SSDP uses HTTP-like messages over UDP multicast (239.255.255.250:1900).
// Devices respond to M-SEARCH requests with device/service descriptions.
//
// Compliance: IEC 62443-3-3 SL-3 — обнаружение только в Zone 5 (Edge LAN)
// Compliance: Приказ ОАЦ №66 п. 7.18.1 — уникальная идентификация устройств
type SSDPDiscovery struct {
	logger *slog.Logger
}

// SSDP multicast address and default timeout.
const (
	ssdpAddr    = "239.255.255.250:1900"
	ssdpTimeout = 3 * time.Second
	ssdpMX      = 2 // Maximum wait time in seconds for responses
)

// ssdpSearchTargets lists the service types to discover.
var ssdpSearchTargets = []string{
	"urn:schemas-upnp-org:device:MediaServer:1",
	"urn:schemas-upnp-org:service:ConnectionManager:1",
	"urn:schemas-upnp-org:device:InternetGatewayDevice:1",
	"ssdp:all", // Fallback: discover everything
}

// NewSSDPDiscovery creates a new SSDP discovery instance.
func NewSSDPDiscovery(logger *slog.Logger) *SSDPDiscovery {
	return &SSDPDiscovery{logger: logger}
}

// Name returns the discovery method name for logging.
func (d *SSDPDiscovery) Name() string {
	return "ssdp"
}

// Discover sends SSDP M-SEARCH requests and collects device responses.
func (d *SSDPDiscovery) Discover(ctx context.Context, timeout time.Duration) ([]DiscoveredService, error) {
	mcastAddr, err := net.ResolveUDPAddr("udp", ssdpAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve SSDP addr: %w", err)
	}

	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, fmt.Errorf("listen SSDP: %w", err)
	}
	defer conn.Close()

	// Send M-SEARCH for each target service type
	for _, st := range ssdpSearchTargets {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		search := buildMSearch(st)
		if _, err := conn.WriteTo(search, mcastAddr); err != nil {
			d.logger.Warn("SSDP send failed", "st", st, "error", err)
			continue
		}
	}

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, fmt.Errorf("set SSDP deadline: %w", err)
	}

	// Collect responses
	services := make(map[string]*DiscoveredService)
	buf := make([]byte, 8192)

	for {
		select {
		case <-ctx.Done():
			return flattenServices(services), ctx.Err()
		default:
		}

		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			d.logger.Debug("SSDP read error", "error", err)
			break
		}

		svc := parseSSDPResponse(buf[:n], addr)
		if svc != nil {
			key := svc.ID
			if key == "" {
				key = fmt.Sprintf("%s:%s", svc.Type, svc.IP.String())
			}
			if existing, ok := services[key]; ok {
				mergeService(existing, *svc)
			} else {
				services[key] = svc
			}
		}
	}

	return flattenServices(services), nil
}

// Scan implements the Scanner interface for Orchestrator integration.
func (d *SSDPDiscovery) Scan(ctx context.Context, subnet string) ([]Device, error) {
	services, err := d.Discover(ctx, ssdpTimeout)
	if err != nil {
		return nil, err
	}
	devices := make([]Device, 0, len(services))
	for _, s := range services {
		devices = append(devices, serviceToDevice(s))
	}
	return devices, nil
}

// --- SSDP M-SEARCH Construction ---

// buildMSearch constructs an SSDP M-SEARCH request for the given service type.
func buildMSearch(st string) []byte {
	return []byte(fmt.Sprintf(
		"M-SEARCH * HTTP/1.1\r\n"+
			"HOST: %s\r\n"+
			"MAN: \"ssdp:discover\"\r\n"+
			"MX: %d\r\n"+
			"ST: %s\r\n"+
			"USER-AGENT: POSIX/1.0 edge-agent/1.0\r\n"+
			"\r\n",
		ssdpAddr, ssdpMX, st,
	))
}

// --- SSDP Response Parsing ---

// parseSSDPResponse parses an SSDP response packet.
// SSDP uses HTTP-like headers over UDP.
func parseSSDPResponse(data []byte, addr net.Addr) *DiscoveredService {
	body := string(data)

	// Determine response type
	isNotify := strings.HasPrefix(body, "NOTIFY")
	isResponse := strings.HasPrefix(body, "HTTP/1.1 200 OK") ||
		strings.HasPrefix(body, "HTTP/1.1 200 OK")

	if !isNotify && !isResponse {
		return nil
	}

	svc := &DiscoveredService{}

	// Parse headers
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break // End of headers
		}

		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			svc = applySSDPHeader(svc, key, value)
		}
	}

	// Set IP from source address
	if udpAddr, ok := addr.(*net.UDPAddr); ok {
		svc.IP = udpAddr.IP
		if svc.Port == 0 {
			svc.Port = udpAddr.Port
		}
	}

	// Extract port from LOCATION URL if not set
	if svc.Port == 0 && svc.Location != "" {
		if u, err := url.Parse(svc.Location); err == nil {
			if u.Port() != "" {
				fmt.Sscanf(u.Port(), "%d", &svc.Port)
			} else if u.Scheme == "https" {
				svc.Port = 443
			} else {
				svc.Port = 80
			}
			// Try to set IP from host if not already set
			if svc.IP == nil {
				if hostIP := net.ParseIP(u.Hostname()); hostIP != nil {
					svc.IP = hostIP
				}
			}
		}
	}

	// Extract manufacturer from SERVER header
	if svc.Server != "" && svc.Manufacturer == "" {
		svc.Manufacturer = extractSSDPVendor(svc.Server)
	}

	if svc.ID == "" && svc.Type == "" && svc.Location == "" {
		return nil // Not enough info
	}

	return svc
}

// applySSDPHeader processes an SSDP HTTP header and updates the service.
func applySSDPHeader(svc *DiscoveredService, key, value string) *DiscoveredService {
	switch strings.ToUpper(key) {
	case "ST":
		svc.Type = value
	case "USN":
		svc.ID = value
	case "LOCATION":
		svc.Location = value
	case "SERVER":
		svc.Server = value
	case "CACHE-CONTROL":
		// Cache control: max-age=seconds
		// Could use for timing, but not critical for discovery
	case "NTS":
		// Notification type: ssdp:alive, ssdp:byebye
		// Not stored currently
	case "NT":
		// Notification type (NOTIFY messages)
		if svc.Type == "" {
			svc.Type = value
		}
	case "AL":
		// Application location
		if svc.Location == "" {
			svc.Location = value
		}
	case "X-USER-AGENT":
		// Custom header from some CCTV devices
		if svc.Server == "" {
			svc.Server = value
		}
	case "X-AV-API-VERSION":
		// AVTech / CCTV vendor specific
		svc.FirmwareVersion = value
	}
	return svc
}

// extractSSDPVendor attempts to extract vendor name from SSDP SERVER header.
// Server format: OS/version UPnP/1.0 Product/version
// Example: "Linux/4.1.15 UPnP/1.0 Hikvision/1.0"
func extractSSDPVendor(server string) string {
	knownVendors := []string{
		"hikvision", "dahua", "axis", "bosch", "panasonic",
		"sony", "samsung", "pelco", "tiandy", "uniview",
		"geovision", "arecont", "avigilon", "honeywell",
		"vivotek", "mobotix", "acti", "d-link", "tp-link",
		"trendnet", "wanscam", "foscam", "amcrest", "reolink",
		"annke", "swann", "lorex", "zmodo", "hiseeu",
	}

	lower := strings.ToLower(server)
	for _, vendor := range knownVendors {
		if strings.Contains(lower, vendor) {
			return vendor
		}
	}
	return ""
}

// --- DeviceClassifier ---

// classifySSDPDevice determines DeviceType from SSDP service type or server header.
func classifySSDPDevice(svc DiscoveredService) DeviceType {
	st := strings.ToLower(svc.Type)
	server := strings.ToLower(svc.Server)

	switch {
	case strings.Contains(st, "camera"),
		strings.Contains(st, "video"),
		strings.Contains(st, "ipcamera"):
		return DeviceTypeCamera
	case strings.Contains(st, "dvr"),
		strings.Contains(st, "digitalvideorecorder"):
		return DeviceTypeDVR
	case strings.Contains(st, "nvr"),
		strings.Contains(st, "networkvideorecorder"):
		return DeviceTypeNVR
	case strings.Contains(st, "gateway"),
		strings.Contains(st, "router"),
		strings.Contains(st, "internetgateway"):
		return DeviceTypeGateway
	case strings.Contains(server, "camera"),
		strings.Contains(server, "ipc"):
		return DeviceTypeCamera
	case strings.Contains(server, "dvr"):
		return DeviceTypeDVR
	case strings.Contains(server, "nvr"):
		return DeviceTypeNVR
	default:
		return DeviceTypeUnknown
	}
}
