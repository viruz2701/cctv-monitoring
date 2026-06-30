package discovery

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// ARPScanner discovers devices by scanning the local ARP cache.
// Uses native Go net.Interface — no external dependencies.
//
// Compliance: IEC 62443-3-3 — сканирование только в пределах Zone 5 (Edge LAN)
type ARPScanner struct{}

// Name returns the scanner name.
func (s *ARPScanner) Name() string {
	return "arp"
}

// Scan reads the local ARP table to discover active devices.
// On Linux, it parses /proc/net/arp. On other platforms, it falls back
// to sending TCP connection probes.
func (s *ARPScanner) Scan(ctx context.Context, subnet string) ([]Device, error) {
	// Parse subnet CIDR
	_, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, fmt.Errorf("parse subnet %s: %w", subnet, err)
	}

	devices, err := s.readARPTable(ipnet)
	if err != nil {
		// Fall back to TCP probe scan if ARP table unavailable
		return s.tcpProbeScan(ctx, ipnet)
	}

	return devices, nil
}

// readARPTable parses /proc/net/arp on Linux (OpenWrt target).
func (s *ARPScanner) readARPTable(ipnet *net.IPNet) ([]Device, error) {
	data, err := os.ReadFile("/proc/net/arp")
	if err != nil {
		return nil, err
	}

	return s.parseARPLines(string(data), ipnet)
}

// parseARPLines parses ARP table lines into Device structs.
func (s *ARPScanner) parseARPLines(data string, ipnet *net.IPNet) ([]Device, error) {
	lines := strings.Split(data, "\n")
	var devices []Device

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		ipStr := fields[0]
		hwStr := fields[3]

		// Skip header and incomplete entries
		if ipStr == "IP" || hwStr == "(incomplete)" || hwStr == "" {
			continue
		}

		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		// Filter by subnet
		if !ipnet.Contains(ip) {
			continue
		}

		mac, err := net.ParseMAC(hwStr)
		if err != nil {
			continue
		}

		devices = append(devices, Device{
			IP:         ip,
			MAC:        mac,
			LastSeen:   time.Now(),
			DeviceType: DeviceTypeUnknown,
		})
	}

	return devices, nil
}

// tcpProbeScan performs a TCP port probe sweep to discover active hosts.
// Used as fallback when /proc/net/arp is not available.
func (s *ARPScanner) tcpProbeScan(ctx context.Context, ipnet *net.IPNet) ([]Device, error) {
	ones, bits := ipnet.Mask.Size()
	if bits != 32 {
		return nil, nil // Only support IPv4
	}

	// Calculate number of hosts
	numHosts := (1 << uint(bits-ones)) - 2
	if numHosts < 1 || numHosts > 254 {
		return nil, nil // Skip large subnets
	}

	ip := ipnet.IP.To4()
	if ip == nil {
		return nil, nil
	}

	var devices []Device
	// Common CCTV ports to probe: 80 (HTTP), 554 (RTSP), 8000 (ONVIF)
	probePorts := []string{"80", "554", "8000"}

	baseIP := make(net.IP, 4)
	copy(baseIP, ip)
	baseIP[3] = 1

	for i := 0; i < int(numHosts) && i < 254; i++ {
		select {
		case <-ctx.Done():
			return devices, ctx.Err()
		default:
		}

		targetIP := net.IP(make([]byte, 4))
		copy(targetIP, baseIP)
		targetIP[3] = baseIP[3] + byte(i)

		for _, port := range probePorts {
			addr := net.JoinHostPort(targetIP.String(), port)
			conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
			if err == nil {
				conn.Close()
				devices = append(devices, Device{
					IP:         targetIP,
					Ports:      []int{parsePort(port)},
					LastSeen:   time.Now(),
					DeviceType: classifyPort(port),
				})
				break // Found at least one open port
			}
		}
	}

	return devices, nil
}

func parsePort(p string) int {
	port := 0
	fmt.Sscanf(p, "%d", &port)
	return port
}

func classifyPort(port string) DeviceType {
	switch port {
	case "80":
		return DeviceTypeUnknown // Could be any web-enabled device
	case "554":
		return DeviceTypeCamera // RTSP typically indicates camera
	case "8000":
		return DeviceTypeCamera // ONVIF typically indicates camera
	default:
		return DeviceTypeUnknown
	}
}
