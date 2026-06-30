package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"
)

// DeviceType represents the type of discovered CCTV device.
type DeviceType string

const (
	DeviceTypeCamera  DeviceType = "camera"
	DeviceTypeDVR     DeviceType = "dvr"
	DeviceTypeNVR     DeviceType = "nvr"
	DeviceTypeSwitch  DeviceType = "switch"
	DeviceTypeGateway DeviceType = "gateway"
	DeviceTypeUnknown DeviceType = "unknown"
)

// ServiceType represents the protocol type of a discovered service.
type ServiceType string

const (
	ServiceTypeONVIF  ServiceType = "onvif"
	ServiceTypeHTTP   ServiceType = "http"
	ServiceTypeRTSP   ServiceType = "rtsp"
	ServiceTypeSSDP   ServiceType = "ssdp"
	ServiceTypeMDNS   ServiceType = "mdns"
	ServiceTypeCustom ServiceType = "custom"
)

// DiscoveredService represents a service discovered via mDNS or SSDP.
//
// Compliance: Приказ ОАЦ №66 п. 7.18.1 — уникальная идентификация устройств
type DiscoveredService struct {
	// ID is the unique service identifier (mDNS instance name or USN).
	ID string
	// Type is the service type (e.g., "_onvif._tcp.local" or "urn:...:MediaServer:1").
	Type string
	// ServiceType is the protocol classification.
	ServiceType ServiceType
	// IP address of the device.
	IP net.IP
	// Port of the service.
	Port int
	// Hostname from SRV record or reverse DNS.
	Hostname string
	// Manufacturer extracted from TXT records or SERVER header.
	Manufacturer string
	// Model extracted from TXT records.
	Model string
	// FirmwareVersion extracted from TXT records.
	FirmwareVersion string
	// Location URL (SSDP LOCATION header).
	Location string
	// Server header string (SSDP).
	Server string
	// TXTRecords is a map of key-value pairs from mDNS TXT records.
	TXTRecords map[string]string
}

// Device represents a discovered network device.
//
// Compliance: Приказ ОАЦ №66 п. 7.18.1 — уникальная идентификация устройств
type Device struct {
	// IP is the device's IP address
	IP net.IP
	// MAC is the device's MAC address
	MAC net.HardwareAddr
	// Hostname from reverse DNS or DHCP
	Hostname string
	// Vendor from MAC OUI lookup or ONVIF
	Vendor string
	// DeviceType (camera, dvr, nvr, etc.)
	DeviceType DeviceType
	// Ports are open TCP/UDP ports discovered
	Ports []int
	// ONVIFScopes from WS-Discovery (if available)
	ONVIFScopes []string
	// InterfaceName is the network interface this device was found on
	InterfaceName string
	// LastSeen timestamp
	LastSeen time.Time
}

// Scanner defines the interface for device discovery methods.
//
// Compliance: IEC 62443-3-3 — сетевые сканеры работают только в пределах
// назначенной зоны (Zone 5 — Edge LAN).
type Scanner interface {
	// Scan performs discovery and returns discovered devices.
	// Must respect context cancellation for timeout control.
	Scan(ctx context.Context, subnet string) ([]Device, error)
	// Name returns the scanner name for logging.
	Name() string
}

// Orchestrator coordinates multiple discovery scanners.
type Orchestrator struct {
	subnet    string
	ifaceName string
	scanners  []Scanner
	logger    *slog.Logger
}

// NewOrchestrator creates a new discovery orchestrator.
// It initializes all available scanners (ARP, ONVIF, SNMP, mDNS, SSDP).
func NewOrchestrator(subnet, ifaceName string, logger *slog.Logger) *Orchestrator {
	return &Orchestrator{
		subnet:    subnet,
		ifaceName: ifaceName,
		scanners: []Scanner{
			&ARPScanner{},
			&ONVIFScanner{},
			&SNMPScanner{},
			NewMDNSDiscovery(logger),
			NewSSDPDiscovery(logger),
		},
		logger: logger,
	}
}

// Scan runs all scanners and merges results, deduplicating by MAC/IP.
func (o *Orchestrator) Scan(ctx context.Context) ([]Device, error) {
	deviceMap := make(map[string]*Device)

	// Resolve network interface
	iface, err := o.resolveInterface()
	if err != nil {
		o.logger.Warn("interface resolution failed, using default", "error", err)
	}

	for _, scanner := range o.scanners {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		o.logger.Debug("running scanner", "scanner", scanner.Name())

		devices, err := scanner.Scan(ctx, o.subnet)
		if err != nil {
			o.logger.Warn("scanner failed",
				"scanner", scanner.Name(),
				"error", err,
			)
			continue
		}

		o.logger.Debug("scanner results",
			"scanner", scanner.Name(),
			"count", len(devices),
		)

		for _, d := range devices {
			key := deviceKey(d.IP, d.MAC)
			if existing, ok := deviceMap[key]; ok {
				mergeDevice(existing, d)
			} else {
				deviceMap[key] = &d
			}
		}
	}

	// Convert map to slice
	result := make([]Device, 0, len(deviceMap))
	for _, d := range deviceMap {
		d.LastSeen = time.Now()
		// Enrich with vendor from MAC OUI if available
		if d.MAC != nil && d.Vendor == "" {
			d.Vendor = lookupOUI(d.MAC)
		}
		result = append(result, *d)
	}

	// Mark interface on devices
	if iface != nil {
		for i := range result {
			result[i].InterfaceName = iface.Name
		}
	}

	return result, nil
}

func (o *Orchestrator) resolveInterface() (*net.Interface, error) {
	if o.ifaceName != "" {
		return net.InterfaceByName(o.ifaceName)
	}

	// Try to find default route interface
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if ipnet.IP.IsPrivate() && ipnet.IP.To4() != nil {
				return &iface, nil
			}
		}
	}

	return nil, nil
}

// deviceKey generates a unique key for deduplication.
func deviceKey(ip net.IP, mac net.HardwareAddr) string {
	if mac != nil {
		return mac.String()
	}
	if ip != nil {
		return ip.String()
	}
	return ""
}

// mergeDevice merges new device data into existing record.
func mergeDevice(existing *Device, new Device) {
	if existing.IP == nil && new.IP != nil {
		existing.IP = new.IP
	}
	if existing.Hostname == "" && new.Hostname != "" {
		existing.Hostname = new.Hostname
	}
	if existing.Vendor == "" && new.Vendor != "" {
		existing.Vendor = new.Vendor
	}
	if existing.DeviceType == DeviceTypeUnknown && new.DeviceType != DeviceTypeUnknown {
		existing.DeviceType = new.DeviceType
	}
	// Merge ports
	portSet := make(map[int]bool)
	for _, p := range existing.Ports {
		portSet[p] = true
	}
	for _, p := range new.Ports {
		if !portSet[p] {
			existing.Ports = append(existing.Ports, p)
			portSet[p] = true
		}
	}
	// Merge ONVIF scopes
	scopeSet := make(map[string]bool)
	for _, s := range existing.ONVIFScopes {
		scopeSet[s] = true
	}
	for _, s := range new.ONVIFScopes {
		if !scopeSet[s] {
			existing.ONVIFScopes = append(existing.ONVIFScopes, s)
			scopeSet[s] = true
		}
	}
}

// lookupOUI returns vendor name from MAC address OUI.
// Basic implementation — can be extended with OUI database file.
func lookupOUI(mac net.HardwareAddr) string {
	if len(mac) < 3 {
		return ""
	}

	// First 3 bytes (24-bit OUI) as string key
	oui := fmt.Sprintf("%02x:%02x:%02x", mac[0], mac[1], mac[2])

	vendors := map[string]string{
		"00:12:0e": "Hikvision",
		"00:1b:a1": "Dahua",
		"00:0c:43": "Axis",
		"00:04:0e": "Bosch",
		"00:1c:b4": "Samsung",
		"00:23:54": "Tiandy",
		"00:1a:4b": "Uniview",
		"00:0f:5e": "Tantos",
		"00:09:0f": "Honeywell",
		"00:12:4b": "Panasonic",
		"00:04:f2": "Sony",
		"00:1e:8c": "Geovision",
		"00:1b:74": "Arecont Vision",
		"00:15:c5": "Pelco",
		"00:0e:3a": "Avigilon",
		"c0:56:27": "Raspberry Pi",
	}

	return vendors[oui]
}

// --- Shared Helpers for mDNS/SSDP Discovery ---

// flattenServices converts a map of services to a slice.
func flattenServices(m map[string]*DiscoveredService) []DiscoveredService {
	result := make([]DiscoveredService, 0, len(m))
	for _, svc := range m {
		result = append(result, *svc)
	}
	return result
}

// mergeService merges fields from src into dst (non-empty fields only).
func mergeService(dst *DiscoveredService, src DiscoveredService) {
	if dst.IP == nil && src.IP != nil {
		dst.IP = src.IP
	}
	if dst.Port == 0 && src.Port != 0 {
		dst.Port = src.Port
	}
	if dst.Hostname == "" && src.Hostname != "" {
		dst.Hostname = src.Hostname
	}
	if dst.Manufacturer == "" && src.Manufacturer != "" {
		dst.Manufacturer = src.Manufacturer
	}
	if dst.Model == "" && src.Model != "" {
		dst.Model = src.Model
	}
	if dst.FirmwareVersion == "" && src.FirmwareVersion != "" {
		dst.FirmwareVersion = src.FirmwareVersion
	}
	if dst.Location == "" && src.Location != "" {
		dst.Location = src.Location
	}
	if dst.Server == "" && src.Server != "" {
		dst.Server = src.Server
	}
	if dst.TXTRecords == nil && src.TXTRecords != nil {
		dst.TXTRecords = src.TXTRecords
	}
}

// serviceToDevice converts a DiscoveredService to a Device for
// integration with the existing Orchestrator.
func serviceToDevice(svc DiscoveredService) Device {
	device := Device{
		IP:         svc.IP,
		LastSeen:   time.Now(),
		DeviceType: DeviceTypeUnknown,
	}
	if svc.Port > 0 {
		device.Ports = append(device.Ports, svc.Port)
	}
	if svc.Hostname != "" {
		device.Hostname = svc.Hostname
	}
	if svc.Manufacturer != "" {
		device.Vendor = svc.Manufacturer
	}
	// Classify device type based on service type
	switch {
	case strings.Contains(svc.Type, "_onvif."):
		device.DeviceType = DeviceTypeCamera
	case strings.Contains(svc.Type, "_rtsp."):
		device.DeviceType = DeviceTypeCamera
	case strings.Contains(svc.Type, "_http."):
		device.DeviceType = DeviceTypeUnknown
	}
	// Add ONVIF scopes if TXT records are available
	if len(svc.TXTRecords) > 0 {
		for k, v := range svc.TXTRecords {
			device.ONVIFScopes = append(device.ONVIFScopes, k+"="+v)
		}
	}
	// Classify via SSDP classifier as fallback
	if dt := classifySSDPDevice(svc); dt != DeviceTypeUnknown {
		device.DeviceType = dt
	}
	return device
}
