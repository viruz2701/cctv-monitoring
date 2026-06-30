package discovery

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// SNMPScanner performs basic SNMP discovery via public community strings.
// Uses raw UDP without gosnmp dependency to keep the binary small for OpenWrt.
//
// Compliance: IEC 62443-3-3 — SNMP доступ только в пределах Zone 5 (Edge LAN)
type SNMPScanner struct{}

const (
	snmpPort     = 161
	snmpTimeout  = 2 * time.Second
	oidSysDescr  = "1.3.6.1.2.1.1.1.0"
	oidSysObject = "1.3.6.1.2.1.1.2.0"
	oidSysName   = "1.3.6.1.2.1.1.5.0"
)

var snmpCommunities = []string{"public", "private", "read"}

// Name returns the scanner name.
func (s *SNMPScanner) Name() string {
	return "snmp"
}

// Scan probes devices for SNMP service and retrieves system information.
func (s *SNMPScanner) Scan(ctx context.Context, subnet string) ([]Device, error) {
	_, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, fmt.Errorf("parse subnet %s: %w", subnet, err)
	}

	arpDevices, err := scanSNMPHosts(ctx, ipnet)
	if err != nil {
		return nil, err
	}

	var devices []Device
	for _, d := range arpDevices {
		select {
		case <-ctx.Done():
			return devices, ctx.Err()
		default:
		}

		sysInfo := s.queryDevice(ctx, d.IP)
		if sysInfo != nil {
			devices = append(devices, *sysInfo)
		}
	}

	return devices, nil
}

func (s *SNMPScanner) queryDevice(ctx context.Context, ip net.IP) *Device {
	device := &Device{
		IP:         ip,
		LastSeen:   time.Now(),
		DeviceType: DeviceTypeUnknown,
	}

	for _, community := range snmpCommunities {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		descr, err := s.snmpGet(ip, community, oidSysDescr)
		if err != nil {
			continue
		}

		device.Ports = append(device.Ports, snmpPort)

		if descr != "" {
			device.Hostname = extractHostname(descr)
			device.Vendor = extractVendorSNMP(descr)
			device.DeviceType = classifyDeviceTypeSNMP(descr)
		}

		name, _ := s.snmpGet(ip, community, oidSysName)
		if name != "" && device.Hostname == "" {
			device.Hostname = name
		}

		break
	}

	if len(device.Ports) == 0 {
		return nil
	}

	return device
}

func (s *SNMPScanner) snmpGet(ip net.IP, community, oid string) (string, error) {
	addr := net.JoinHostPort(ip.String(), fmt.Sprintf("%d", snmpPort))
	conn, err := net.DialTimeout("udp", addr, snmpTimeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	req := buildSNMPGetRequest(community, oid)
	if req == nil {
		return "", fmt.Errorf("failed to build request")
	}

	conn.SetDeadline(time.Now().Add(snmpTimeout))

	if _, err := conn.Write(req); err != nil {
		return "", err
	}

	resp := make([]byte, 4096)
	n, err := conn.Read(resp)
	if err != nil {
		return "", err
	}

	return parseSNMPResponse(resp[:n])
}

// --- SNMP Protocol Encoding (BER/DER) ---

func buildSNMPGetRequest(community, oid string) []byte {
	oidBytes := parseOID(oid)
	if oidBytes == nil {
		return nil
	}

	// Build VarBind: SEQUENCE { OID, NULL }
	varBindContent := make([]byte, 0, len(oidBytes)+4)
	varBindContent = append(varBindContent, buildOID(oidBytes)...)
	varBindContent = append(varBindContent, 0x05, 0x00) // NULL
	varBind := wrapSequence(varBindContent)

	// Build VarBindList: SEQUENCE of VarBind
	varBindList := wrapSequence(varBind)

	// Build GetRequest PDU (tag 0xA0)
	reqID := []byte{0x02, 0x04, 0x00, 0x00, 0x00, 0x01} // INTEGER 1
	errorVal := []byte{0x02, 0x01, 0x00}                // INTEGER 0
	errorIdx := []byte{0x02, 0x01, 0x00}                // INTEGER 0

	pduContent := make([]byte, 0, len(reqID)+len(errorVal)+len(errorIdx)+len(varBindList))
	pduContent = append(pduContent, reqID...)
	pduContent = append(pduContent, errorVal...)
	pduContent = append(pduContent, errorIdx...)
	pduContent = append(pduContent, varBindList...)
	pdu := wrapTagged(0xA0, pduContent)

	// Version (INTEGER 1 = SNMP v2c) + Community (OCTET STRING)
	version := []byte{0x02, 0x01, 0x01}
	communityBytes := wrapOctetString([]byte(community))

	// Message body: SEQUENCE { version, community, PDU }
	msgBody := make([]byte, 0, len(version)+len(communityBytes)+len(pdu))
	msgBody = append(msgBody, version...)
	msgBody = append(msgBody, communityBytes...)
	msgBody = append(msgBody, pdu...)

	return wrapSequence(msgBody)
}

func parseSNMPResponse(data []byte) (string, error) {
	if len(data) < 2 {
		return "", fmt.Errorf("response too short")
	}

	_, offset, err := parseTLV(data, 0)
	if err != nil {
		return "", err
	}

	for i := 0; i < 2; i++ {
		_, offset, err = parseTLV(data, offset)
		if err != nil {
			return "", err
		}
	}

	var tag byte
	tag, offset, err = parseTLV(data, offset)
	if err != nil {
		return "", err
	}
	if tag != 0xA2 {
		return "", fmt.Errorf("expected Response PDU (0xA2), got 0x%02x", tag)
	}

	for i := 0; i < 3; i++ {
		_, offset, err = parseTLV(data, offset)
		if err != nil {
			return "", err
		}
	}

	_, offset, err = parseTLV(data, offset)
	if err != nil {
		return "", err
	}

	_, offset, err = parseTLV(data, offset)
	if err != nil {
		return "", err
	}

	_, offset, err = parseTLV(data, offset)
	if err != nil {
		return "", err
	}

	value, _, err := parseSNMPValue(data, offset)
	if err != nil {
		return "", err
	}

	return value, nil
}

// --- ASN.1 BER Tag-Length-Value parsing ---

func parseTLV(data []byte, offset int) (byte, int, error) {
	if offset >= len(data) {
		return 0, 0, fmt.Errorf("offset out of bounds")
	}

	tag := data[offset]
	offset++

	if offset >= len(data) {
		return 0, 0, fmt.Errorf("length missing")
	}

	length := int(data[offset])
	offset++

	if length&0x80 != 0 {
		numBytes := length & 0x7f
		length = 0
		for i := 0; i < numBytes; i++ {
			if offset >= len(data) {
				return 0, 0, fmt.Errorf("long length truncated")
			}
			length = (length << 8) | int(data[offset])
			offset++
		}
	}

	if offset+length > len(data) {
		return 0, 0, fmt.Errorf("value truncated: need %d bytes at offset %d, have %d",
			length, offset, len(data))
	}

	return tag, offset + length, nil
}

func parseSNMPValue(data []byte, offset int) (string, int, error) {
	if offset >= len(data) {
		return "", 0, fmt.Errorf("offset out of bounds")
	}

	tag := data[offset]
	offset++

	if offset >= len(data) {
		return "", 0, fmt.Errorf("length missing")
	}

	length := int(data[offset])
	offset++

	if offset+length > len(data) {
		return "", 0, fmt.Errorf("value truncated")
	}

	value := data[offset : offset+length]
	offset += length

	switch tag {
	case 0x02:
		return fmt.Sprintf("%d", bytesToInt(value)), offset, nil
	case 0x04, 0x12:
		return string(value), offset, nil
	case 0x05:
		return "", offset, nil
	case 0x06:
		return formatOID(value), offset, nil
	default:
		return string(value), offset, nil
	}
}

// --- ASN.1 BER Encoding Helpers ---

func wrapSequence(content []byte) []byte {
	return append([]byte{0x30, byte(len(content))}, content...)
}

func wrapTagged(tag byte, content []byte) []byte {
	return append([]byte{tag, byte(len(content))}, content...)
}

func wrapOctetString(data []byte) []byte {
	return append([]byte{0x04, byte(len(data))}, data...)
}

func buildOID(oidBytes []byte) []byte {
	return append([]byte{0x06, byte(len(oidBytes))}, oidBytes...)
}

func parseOID(oid string) []byte {
	parts := strings.Split(oid, ".")
	if len(parts) < 2 {
		return nil
	}

	var result []byte
	first, err := parseInt(parts[0])
	if err != nil {
		return nil
	}
	second, err := parseInt(parts[1])
	if err != nil {
		return nil
	}

	result = append(result, byte(first*40+second))

	for _, part := range parts[2:] {
		val, err := parseInt(part)
		if err != nil {
			return nil
		}
		result = append(result, encodeOIDComponent(val)...)
	}

	return result
}

func encodeOIDComponent(val int) []byte {
	if val < 128 {
		return []byte{byte(val)}
	}

	var bytes []byte
	for val > 0 {
		b := byte(val & 0x7f)
		val >>= 7
		if len(bytes) > 0 {
			b |= 0x80
		}
		bytes = append([]byte{b}, bytes...)
	}
	return bytes
}

func scanSNMPHosts(ctx context.Context, ipnet *net.IPNet) ([]Device, error) {
	ones, bits := ipnet.Mask.Size()
	if bits != 32 {
		return nil, nil
	}

	numHosts := (1 << uint(bits-ones)) - 2
	if numHosts < 1 || numHosts > 254 {
		return nil, nil
	}

	ip := ipnet.IP.To4()
	if ip == nil {
		return nil, nil
	}

	var devices []Device
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

		addr := net.JoinHostPort(targetIP.String(), fmt.Sprintf("%d", snmpPort))
		conn, err := net.DialTimeout("udp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			devices = append(devices, Device{
				IP:         targetIP,
				Ports:      []int{snmpPort},
				LastSeen:   time.Now(),
				DeviceType: DeviceTypeUnknown,
			})
		}
	}

	return devices, nil
}

// --- Misc Helpers ---

func bytesToInt(b []byte) int {
	val := 0
	for _, v := range b {
		val = (val << 8) | int(v)
	}
	return val
}

func parseInt(s string) (int, error) {
	val := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}
		val = val*10 + int(c-'0')
	}
	return val, nil
}

func formatOID(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	parts := []string{
		fmt.Sprintf("%d.%d", data[0]/40, data[0]%40),
	}

	val := 0
	for _, b := range data[1:] {
		val = (val << 7) | int(b&0x7f)
		if b&0x80 == 0 {
			parts = append(parts, fmt.Sprintf("%d", val))
			val = 0
		}
	}

	return strings.Join(parts, ".")
}

func extractHostname(descr string) string {
	lines := strings.SplitN(descr, "\n", 2)
	return strings.TrimSpace(lines[0])
}

func extractVendorSNMP(descr string) string {
	descrLower := strings.ToLower(descr)
	vendors := []string{
		"hikvision", "dahua", "axis", "bosch", "panasonic",
		"sony", "samsung", "pelco", "tiandy", "uniview",
		"geovision", "arecont", "avigilon", "honeywell",
	}
	for _, v := range vendors {
		if strings.Contains(descrLower, v) {
			return strings.Title(v)
		}
	}
	return ""
}

func classifyDeviceTypeSNMP(descr string) DeviceType {
	lower := strings.ToLower(descr)
	switch {
	case strings.Contains(lower, "camera"), strings.Contains(lower, "ipc"):
		return DeviceTypeCamera
	case strings.Contains(lower, "dvr"):
		return DeviceTypeDVR
	case strings.Contains(lower, "nvr"):
		return DeviceTypeNVR
	case strings.Contains(lower, "switch"), strings.Contains(lower, "vlan"):
		return DeviceTypeSwitch
	default:
		return DeviceTypeUnknown
	}
}
