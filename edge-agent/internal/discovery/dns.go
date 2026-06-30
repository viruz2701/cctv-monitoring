package discovery

import (
	"fmt"
	"net"
	"strings"
)

// dnsRecord represents a parsed DNS resource record.
type dnsRecord struct {
	Name   string
	Type   uint16
	Class  uint16
	TTL    uint32
	Data   []byte
	Parsed bool
}

// parseMDNSResponse parses a DNS response packet into discovered services.
func parseMDNSResponse(data []byte) []DiscoveredService {
	if len(data) < 12 {
		return nil
	}

	// Parse header
	flags := uint16(data[2])<<8 | uint16(data[3])
	// Check QR bit (response flag)
	if flags&0x8000 == 0 {
		return nil // Not a response
	}

	ancount := int(data[6])<<8 | int(data[7])
	nscount := int(data[8])<<8 | int(data[9])
	arcount := int(data[10])<<8 | int(data[11])

	if ancount == 0 && arcount == 0 {
		return nil
	}

	// Skip questions
	offset := 12
	qdcount := int(data[4])<<8 | int(data[5])
	for i := 0; i < qdcount; i++ {
		_, next, err := parseDNSName(data, offset)
		if err != nil {
			return nil
		}
		offset = next + 4 // Skip QTYPE + QCLASS
	}

	// Parse answer + additional records
	records := parseAllRecords(data, offset, ancount, nscount, arcount)
	return buildServicesFromRecords(records)
}

// parseAllRecords parses answer, authority, and additional sections.
func parseAllRecords(data []byte, offset int, ancount, nscount, arcount int) []dnsRecord {
	total := ancount + nscount + arcount
	if total == 0 {
		return nil
	}

	records := make([]dnsRecord, 0, total)
	offsetEnd := len(data)

	for i := 0; i < total; i++ {
		if offset >= offsetEnd {
			break
		}

		name, next, err := parseDNSName(data, offset)
		if err != nil {
			break
		}
		offset = next

		// Type, Class, TTL, RDLENGTH
		if offset+10 > offsetEnd {
			break
		}
		rtype := uint16(data[offset])<<8 | uint16(data[offset+1])
		rclass := uint16(data[offset+2])<<8 | uint16(data[offset+3])
		rttl := uint32(data[offset+4])<<24 | uint32(data[offset+5])<<16 |
			uint32(data[offset+6])<<8 | uint32(data[offset+7])
		rdlength := int(data[offset+8])<<8 | int(data[offset+9])
		offset += 10

		if offset+rdlength > offsetEnd {
			break
		}

		records = append(records, dnsRecord{
			Name:   name,
			Type:   rtype,
			Class:  rclass,
			TTL:    rttl,
			Data:   data[offset : offset+rdlength],
			Parsed: false,
		})
		offset += rdlength
	}

	return records
}

// buildServicesFromRecords assembles DiscoveredService entries from
// PTR, SRV, TXT, and A records.
func buildServicesFromRecords(records []dnsRecord) []DiscoveredService {
	// Index records by type
	type ptrEntry struct {
		instanceName string
		serviceType  string
		ttl          uint32
	}
	var ptrs []ptrEntry

	srvMap := make(map[string]struct {
		target string
		port   uint16
		ttl    uint32
	})
	txtMap := make(map[string][]dnsRecord)
	addrMap := make(map[string]net.IP)

	for _, r := range records {
		if r.Parsed {
			continue
		}
		r.Parsed = true

		switch r.Type {
		case dnsTypePTR:
			name, _ := parseDNSNameRaw(r.Data)
			if name != "" {
				ptrs = append(ptrs, ptrEntry{
					instanceName: name,
					serviceType:  r.Name,
					ttl:          r.TTL,
				})
			}
		case dnsTypeSRV:
			if len(r.Data) >= 6 {
				port := uint16(r.Data[4])<<8 | uint16(r.Data[5])
				target, _ := parseDNSNameRaw(r.Data[6:])
				if target != "" {
					srvMap[r.Name] = struct {
						target string
						port   uint16
						ttl    uint32
					}{target: target, port: port, ttl: r.TTL}
				}
			}
		case dnsTypeTXT:
			txtMap[r.Name] = append(txtMap[r.Name], r)
		case dnsTypeA:
			if len(r.Data) == 4 {
				ip := net.IPv4(r.Data[0], r.Data[1], r.Data[2], r.Data[3])
				addrMap[r.Name] = ip
			}
		}
	}

	// Build services from PTR records
	var services []DiscoveredService
	for _, ptr := range ptrs {
		svc := DiscoveredService{
			ID:   ptr.instanceName,
			Type: ptr.serviceType,
		}

		// Look up SRV for this instance
		if srv, ok := srvMap[ptr.instanceName]; ok {
			svc.Port = int(srv.port)
			svc.Hostname = srv.target
			// Look up A record for SRV target
			if ip, ok := addrMap[srv.target]; ok {
				svc.IP = ip
			}
		}

		// Look up TXT records for this instance
		if txts, ok := txtMap[ptr.instanceName]; ok {
			kv := make(map[string]string)
			for _, t := range txts {
				parseTXTRecord(t.Data, kv)
			}
			if len(kv) > 0 {
				svc.TXTRecords = kv
				svc.Manufacturer = kv["manufacturer"]
				svc.Model = kv["model"]
				svc.FirmwareVersion = kv["firmware"]
				if svc.Hostname == "" {
					svc.Hostname = kv["hostname"]
				}
			}
		}

		// Fallback: if no SRV but hostname was set via TXT, try to find A record
		if svc.IP == nil && svc.Hostname != "" {
			if ip, ok := addrMap[svc.Hostname]; ok {
				svc.IP = ip
			}
		}

		services = append(services, svc)
	}

	return services
}

// parseDNSNameRaw extracts a domain name from raw RDATA bytes
// (without the parent record header context).
func parseDNSNameRaw(data []byte) (string, bool) {
	name, _, err := parseDNSName(data, 0)
	return name, err == nil
}

// parseDNSName parses a DNS-encoded domain name starting at offset,
// handling label compression pointers (RFC 1035 §4.1.4).
func parseDNSName(data []byte, offset int) (string, int, error) {
	var labels []string
	startOffset := offset
	jumped := false

	for {
		if offset >= len(data) {
			return "", offset, fmt.Errorf("dns: truncated name")
		}

		b := data[offset]
		if b == 0x00 {
			if !jumped {
				offset++
			}
			break
		}

		if b&0xC0 == 0xC0 {
			// Compression pointer (2 bytes)
			if offset+1 >= len(data) {
				return "", offset, fmt.Errorf("dns: truncated pointer")
			}
			ptr := int(b&0x3F)<<8 | int(data[offset+1])
			if ptr >= len(data) {
				return "", offset, fmt.Errorf("dns: pointer out of bounds")
			}
			if !jumped {
				offset += 2
				jumped = true
			}
			expanded, _, err := parseDNSName(data, ptr)
			if err != nil {
				return "", offset, err
			}
			labels = append(labels, expanded)
			break
		}

		// Normal label
		length := int(b)
		offset++
		if offset+length > len(data) {
			return "", offset, fmt.Errorf("dns: truncated label")
		}
		labels = append(labels, string(data[offset:offset+length]))
		offset += length
	}

	name := strings.Join(labels, ".")
	if !jumped {
		return name, offset, nil
	}
	return name, startOffset, nil
}

// parseTXTRecord parses DNS TXT record data into a key-value map.
// TXT format: sequence of length-prefixed strings ("key=value").
func parseTXTRecord(data []byte, kv map[string]string) {
	offset := 0
	for offset < len(data) {
		if offset >= len(data) {
			break
		}
		length := int(data[offset])
		offset++
		if offset+length > len(data) {
			break
		}
		entry := string(data[offset : offset+length])
		offset += length

		if idx := strings.IndexByte(entry, '='); idx > 0 {
			key := strings.ToLower(strings.TrimSpace(entry[:idx]))
			value := strings.TrimSpace(entry[idx+1:])
			if key != "" {
				kv[key] = value
			}
		}
	}
}
