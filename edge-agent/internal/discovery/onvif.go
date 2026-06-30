package discovery

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// ONVIFScanner discovers ONVIF-compatible devices via WS-Discovery.
// Sends SOAP multicast probe messages over UDP.
//
// Compliance: IEC 62443-3-3 SL-3 — обнаружение только внутри зоны Edge LAN
type ONVIFScanner struct{}

const (
	onvifMulticastAddr = "239.255.255.250:3702"
	onvifProbeTimeout  = 3 * time.Second
)

// Name returns the scanner name.
func (s *ONVIFScanner) Name() string {
	return "onvif"
}

// Scan sends WS-Discovery Probe message to ONVIF multicast address
// and collects responses from compatible devices.
func (s *ONVIFScanner) Scan(ctx context.Context, subnet string) ([]Device, error) {
	// Resolve multicast address
	mcastAddr, err := net.ResolveUDPAddr("udp", onvifMulticastAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve multicast addr: %w", err)
	}

	// Use PacketConn for multicast (works on all platforms)
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, fmt.Errorf("listen packet: %w", err)
	}
	defer conn.Close()

	// Send WS-Discovery Probe message
	probe := buildONVIFProbe()
	if _, err := conn.WriteTo(probe, mcastAddr); err != nil {
		return nil, fmt.Errorf("send probe: %w", err)
	}

	// Read responses with timeout
	if err := conn.SetReadDeadline(time.Now().Add(onvifProbeTimeout)); err != nil {
		return nil, fmt.Errorf("set read deadline: %w", err)
	}

	var devices []Device
	buf := make([]byte, 8192)

	for {
		select {
		case <-ctx.Done():
			return devices, ctx.Err()
		default:
		}

		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break // Timeout — no more responses
			}
			break
		}

		device, ok := parseONVIFResponse(buf[:n])
		if ok {
			devices = append(devices, device)
		}
	}

	return devices, nil
}

// buildONVIFProbe constructs a WS-Discovery Probe SOAP message.
func buildONVIFProbe() []byte {
	template := `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"
               xmlns:wsa="http://schemas.xmlsoap.org/ws/2004/08/addressing"
               xmlns:wsd="http://schemas.xmlsoap.org/ws/2005/04/discovery"
               xmlns:dn="http://www.onvif.org/ver10/network/wsdl">
  <soap:Header>
    <wsa:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</wsa:Action>
    <wsa:MessageID>uuid:%s</wsa:MessageID>
    <wsa:To>urn:schemas-xmlsoap-org:ws:2005:04:discovery</wsa:To>
  </soap:Header>
  <soap:Body>
    <wsd:Probe>
      <wsd:Types>dn:NetworkVideoTransmitter</wsd:Types>
    </wsd:Probe>
  </soap:Body>
</soap:Envelope>`

	uuid := fmt.Sprintf("edge-agent-%d", time.Now().UnixNano())
	return []byte(fmt.Sprintf(template, uuid))
}

// parseONVIFResponse parses WS-Discovery ProbeMatch response.
func parseONVIFResponse(data []byte) (Device, bool) {
	body := string(data)

	if !strings.Contains(body, "ProbeMatches") {
		return Device{}, false
	}

	device := Device{
		DeviceType: DeviceTypeCamera,
		LastSeen:   time.Now(),
	}

	// Extract XAddrs (device addresses)
	if xaddrs := extractXMLTag(body, "XAddrs"); xaddrs != "" {
		urls := strings.Split(xaddrs, " ")
		for _, url := range urls {
			url = strings.TrimSpace(url)
			if url == "" {
				continue
			}
			if host := extractHostFromURL(url); host != "" {
				ip := net.ParseIP(host)
				if ip != nil {
					device.IP = ip
					break
				}
				// Try DNS resolution
				ips, err := net.LookupIP(host)
				if err == nil && len(ips) > 0 {
					device.IP = ips[0]
					break
				}
			}
		}
	}

	// Extract Types (device type info)
	if types := extractXMLTag(body, "Types"); types != "" {
		device.ONVIFScopes = append(device.ONVIFScopes, types)
	}

	// Extract Scopes (capabilities)
	if scopes := extractXMLTag(body, "Scopes"); scopes != "" {
		scopeList := strings.Split(scopes, " ")
		for _, scope := range scopeList {
			scope = strings.TrimSpace(scope)
			if scope != "" {
				device.ONVIFScopes = append(device.ONVIFScopes, scope)
				// Try to extract vendor from scope
				if vendor := extractVendorFromScope(scope); vendor != "" {
					device.Vendor = vendor
				}
			}
		}
	}

	// Extract device MAC or hardware ID
	if hwID := extractHardwareID(body); hwID != "" {
		if mac, err := net.ParseMAC(hwID); err == nil {
			device.MAC = mac
		}
	}

	if device.IP == nil {
		return Device{}, false
	}

	return device, true
}

// extractXMLTag extracts content between XML tags.
func extractXMLTag(body, tag string) string {
	openTag := "<" + tag + ">"
	closeTag := "</" + tag + ">"

	start := strings.Index(body, openTag)
	if start == -1 {
		// Try namespace-qualified tag
		openTag = ":" + tag + ">"
		start = strings.Index(body, openTag)
		if start == -1 {
			return ""
		}
		start += len(openTag)
	} else {
		start += len(openTag)
	}

	end := strings.Index(body[start:], closeTag)
	if end == -1 {
		// Try namespace-qualified close tag
		closeTag = ":" + tag + ">"
		end = strings.Index(body[start:], closeTag)
		if end == -1 {
			return ""
		}
	}

	return strings.TrimSpace(body[start : start+end])
}

// extractHostFromURL extracts hostname or IP from a URL-like string.
func extractHostFromURL(url string) string {
	// Strip scheme
	if idx := strings.Index(url, "://"); idx != -1 {
		url = url[idx+3:]
	}
	// Strip port and path
	if idx := strings.Index(url, ":"); idx != -1 {
		url = url[:idx]
	}
	if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}
	return url
}

// extractVendorFromScope attempts to extract vendor name from ONVIF scope.
// Example: onvif://www.onvif.org/type/axis -> "Axis"
func extractVendorFromScope(scope string) string {
	knownVendors := []string{
		"axis", "hikvision", "dahua", "bosch", "panasonic",
		"samsung", "sony", "pelco", "tiandy", "uniview",
	}

	scopeLower := strings.ToLower(scope)
	for _, vendor := range knownVendors {
		if strings.Contains(scopeLower, vendor) {
			return strings.Title(vendor)
		}
	}
	return ""
}

// extractHardwareID attempts to find a MAC address or hardware ID
// in the ONVIF response body.
func extractHardwareID(body string) string {
	bodyBytes := bytes.TrimSpace([]byte(body))
	words := bytes.Fields(bodyBytes)
	for _, word := range words {
		w := string(word)
		if strings.Count(w, ":") == 5 && len(w) == 17 {
			_, err := net.ParseMAC(w)
			if err == nil {
				return w
			}
		}
	}
	return ""
}

// findInterfaceForSubnet finds network interface for given subnet.
func findInterfaceForSubnet(subnet string) (*net.Interface, error) {
	_, targetNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, err
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
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
			if targetNet.Contains(ipnet.IP) {
				return &iface, nil
			}
		}
	}

	return nil, fmt.Errorf("no interface found for subnet %s", subnet)
}
