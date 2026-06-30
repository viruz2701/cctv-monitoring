package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"
)

// MDNSDiscovery discovers CCTV/IoT devices via Multicast DNS (RFC 6762).
//
// Compliance: IEC 62443-3-3 SL-3 — обнаружение только в Zone 5 (Edge LAN)
// Compliance: Приказ ОАЦ №66 п. 7.18.1 — уникальная идентификация устройств
type MDNSDiscovery struct {
	logger *slog.Logger
}

// DNS record type constants for mDNS packet parsing.
const (
	dnsTypeA   = 1
	dnsTypePTR = 12
	dnsTypeTXT = 16
	dnsTypeSRV = 33
)

// mDNS multicast address and default timeout.
const (
	mdnsAddr    = "224.0.0.251:5353"
	mdnsTimeout = 3 * time.Second
)

// mdnsServices are the service types queried for CCTV/IoT device discovery.
var mdnsServices = []string{
	"_onvif._tcp.local",
	"_http._tcp.local",
	"_rtsp._tcp.local",
}

// NewMDNSDiscovery creates a new mDNS discovery instance.
func NewMDNSDiscovery(logger *slog.Logger) *MDNSDiscovery {
	return &MDNSDiscovery{logger: logger}
}

// Name returns the discovery method name for logging.
func (d *MDNSDiscovery) Name() string {
	return "mdns"
}

// Discover sends mDNS PTR queries for CCTV-related service types
// and collects responses from compatible devices.
func (d *MDNSDiscovery) Discover(ctx context.Context, timeout time.Duration) ([]DiscoveredService, error) {
	mcastAddr, err := net.ResolveUDPAddr("udp", mdnsAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve mDNS addr: %w", err)
	}

	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, fmt.Errorf("listen mDNS: %w", err)
	}
	defer conn.Close()

	// Build and send mDNS PTR query packet
	query := buildMDNSQuery()
	if _, err := conn.WriteTo(query, mcastAddr); err != nil {
		return nil, fmt.Errorf("send mDNS query: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, fmt.Errorf("set mDNS deadline: %w", err)
	}

	// Parse responses — mDNS responders may send multiple packets
	services := make(map[string]*DiscoveredService)
	buf := make([]byte, 65535)

	for {
		select {
		case <-ctx.Done():
			return flattenServices(services), ctx.Err()
		default:
		}

		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			d.logger.Debug("mDNS read error", "error", err)
			break
		}

		parsed := parseMDNSResponse(buf[:n])
		for _, svc := range parsed {
			key := svc.ID
			if key == "" {
				key = fmt.Sprintf("%s:%s", svc.Type, svc.IP.String())
			}
			if existing, ok := services[key]; ok {
				mergeService(existing, svc)
			} else {
				svcCopy := svc
				services[key] = &svcCopy
			}
		}
	}

	return flattenServices(services), nil
}

// Scan implements the Scanner interface for Orchestrator integration.
func (d *MDNSDiscovery) Scan(ctx context.Context, subnet string) ([]Device, error) {
	services, err := d.Discover(ctx, mdnsTimeout)
	if err != nil {
		return nil, err
	}
	devices := make([]Device, 0, len(services))
	for _, s := range services {
		devices = append(devices, serviceToDevice(s))
	}
	return devices, nil
}

// --- mDNS Query Construction ---

// buildMDNSQuery constructs a DNS query packet with PTR questions
// for each CCTV-relevant service type.
func buildMDNSQuery() []byte {
	var buf []byte

	// DNS Header (12 bytes): ID=0, Flags=0 (standard query)
	buf = append(buf, 0x00, 0x00, 0x00, 0x00)        // ID=0, Flags=0
	buf = append(buf, 0x00, byte(len(mdnsServices))) // QDCOUNT
	buf = append(buf, 0x00, 0x00)                    // ANCOUNT=0
	buf = append(buf, 0x00, 0x00)                    // NSCOUNT=0
	buf = append(buf, 0x00, 0x00)                    // ARCOUNT=0

	// Questions
	for _, svc := range mdnsServices {
		buf = append(buf, encodeDNSName(svc)...)
		buf = append(buf, 0x00, dnsTypePTR) // QTYPE=PTR
		buf = append(buf, 0x00, 0x01)       // QCLASS=IN
	}

	return buf
}

// encodeDNSName converts a domain name (e.g. "_onvif._tcp.local")
// into DNS wire format (sequence of length-prefixed labels).
func encodeDNSName(name string) []byte {
	if name == "" {
		return []byte{0x00}
	}
	labels := strings.Split(name, ".")
	var buf []byte
	for _, label := range labels {
		if len(label) > 63 {
			label = label[:63]
		}
		buf = append(buf, byte(len(label)))
		buf = append(buf, label...)
	}
	buf = append(buf, 0x00)
	return buf
}
