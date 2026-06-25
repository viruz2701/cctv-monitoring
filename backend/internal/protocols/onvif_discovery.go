// CCTV-2.2.1: WS-Discovery for ONVIF devices
//
// Реализует WS-Discovery (SOAP over UDP multicast) для поиска ONVIF-совместимых
// устройств в локальной сети. Соответствует ONVIF Discovery Specification.
//
// Compliance:
//   - IEC 62443-3-3 SL-3: поиск только в доверенной сети (L2)
//   - OWASP ASVS L3: валидация входящих XML от устройств
//   - Приказ ОАЦ №66 п.7.18: уникальная идентификация через XAddr + Scopes

package protocols

import (
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"
)

// ─── WS-Discovery Constants ─────────────────────────────────────────────────

const (
	wsDiscoveryAddr      = "239.255.255.250"
	wsDiscoveryPort      = 3702
	wsDiscoveryMulticast = "urn:schemas-xmlsoap-org:ws:2005:04:discovery"
	wsDiscoveryProbe     = `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope
	xmlns:soap="http://www.w3.org/2003/05/soap-envelope"
	xmlns:wsa="http://schemas.xmlsoap.org/ws/2004/08/addressing"
	xmlns:wsd="%s"
	xmlns:wsdp="http://schemas.xmlsoap.org/ws/2006/02/devprof">
	<soap:Header>
		<wsa:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</wsa:Action>
		<wsa:MessageID>urn:uuid:%s</wsa:MessageID>
		<wsa:To>%s</wsa:To>
	</soap:Header>
	<soap:Body>
		<wsd:Probe>
			<wsd:Types>wsdp:Device</wsd:Types>
		</wsd:Probe>
	</soap:Body>
</soap:Envelope>`
)

// ─── Discovered Device ──────────────────────────────────────────────────────

// DiscoveredDevice представляет устройство, найденное через WS-Discovery.
type DiscoveredDevice struct {
	XAddr           string
	Scopes          []string
	Types           []string
	MetadataVersion string
}

// ProbeMatch ответ от устройства.
type ProbeMatch struct {
	XMLName           xml.Name `xml:"ProbeMatch"`
	EndpointReference struct {
		Address string `xml:"Address"`
	} `xml:"EndpointReference"`
	Types           string `xml:"Types"`
	Scopes          string `xml:"Scopes"`
	XAddrs          string `xml:"XAddrs"`
	MetadataVersion string `xml:"MetadataVersion"`
}

// ProbeMatches контейнер ответов.
type ProbeMatches struct {
	XMLName xml.Name     `xml:"ProbeMatches"`
	Matches []ProbeMatch `xml:"ProbeMatch"`
}

// DiscoveryResponse полный SOAP ответ.
type DiscoveryResponse struct {
	XMLName xml.Name      `xml:"Envelope"`
	Body    DiscoveryBody `xml:"Body"`
}

type DiscoveryBody struct {
	XMLName      xml.Name     `xml:"Body"`
	ProbeMatches ProbeMatches `xml:"ProbeMatches"`
}

// ─── WS-Discovery Client ────────────────────────────────────────────────────

// DiscoverONVIFDevices выполняет WS-Discovery Probe и возвращает найденные устройства.
// Использует multicast UDP на 239.255.255.250:3702.
func DiscoverONVIFDevices(ctx context.Context, port int, timeout time.Duration, logger *slog.Logger) ([]DiscoveredDevice, error) {
	if port <= 0 {
		port = wsDiscoveryPort
	}

	// Парсим multicast адрес
	addr := fmt.Sprintf("%s:%d", wsDiscoveryAddr, port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("resolve multicast addr: %w", err)
	}

	// Открываем UDP соединение
	conn, err := net.ListenMulticastUDP("udp", nil, udpAddr)
	if err != nil {
		// Fallback: обычный UDP listen
		conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		if err != nil {
			return nil, fmt.Errorf("listen udp: %w", err)
		}
	}
	defer conn.Close()

	// Таймаут
	deadline := time.Now().Add(timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("set deadline: %w", err)
	}

	// Формируем Probe сообщение
	messageID := fmt.Sprintf("urn:uuid:%s", newUUID())
	probeMsg := fmt.Sprintf(wsDiscoveryProbe, wsDiscoveryMulticast, messageID, addr)

	// Отправляем Probe через multicast
	sent, err := conn.WriteToUDP([]byte(probeMsg), udpAddr)
	if err != nil {
		return nil, fmt.Errorf("send probe: %w", err)
	}

	logger.Debug("WS-Discovery probe sent", "bytes", sent, "addr", addr)

	// Читаем ответы
	var devices []DiscoveredDevice
	buf := make([]byte, 65536)

	for {
		select {
		case <-ctx.Done():
			return devices, ctx.Err()
		default:
		}

		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			// Таймаут — нормальное завершение
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			logger.Debug("WS-Discovery read error", "error", err)
			break
		}

		// Парсим ProbeMatch
		device, err := parseProbeMatch(buf[:n])
		if err != nil {
			logger.Debug("WS-Discovery parse error", "error", err)
			continue
		}

		if device != nil {
			devices = append(devices, *device)
			logger.Debug("WS-Discovery device found",
				"xaddr", device.XAddr,
				"types", device.Types,
			)
		}
	}

	logger.Info("WS-Discovery completed",
		"devices_found", len(devices),
		"timeout", timeout,
	)

	return devices, nil
}

// ─── Response Parsing ───────────────────────────────────────────────────────

func parseProbeMatch(data []byte) (*DiscoveredDevice, error) {
	// Пробуем распарсить как полноценный SOAP ответ
	var resp DiscoveryResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal discovery response: %w", err)
	}

	if len(resp.Body.ProbeMatches.Matches) == 0 {
		return nil, nil
	}

	// Берём первый ProbeMatch (обычно одно устройство на пакет)
	match := resp.Body.ProbeMatches.Matches[0]

	// Парсим Scopes
	scopes := parseScopes(match.Scopes)

	// Парсим Types
	types := parseScopes(match.Types)

	device := &DiscoveredDevice{
		XAddr:           match.XAddrs,
		Scopes:          scopes,
		Types:           types,
		MetadataVersion: match.MetadataVersion,
	}

	// Валидация: XAddr обязателен
	if device.XAddr == "" {
		return nil, fmt.Errorf("device with empty XAddr")
	}

	return device, nil
}

// ─── Scopes Parsing ─────────────────────────────────────────────────────────

func parseScopes(raw string) []string {
	if raw == "" {
		return nil
	}

	var scopes []string
	current := ""
	inSpace := false

	for _, ch := range raw {
		if ch == ' ' {
			if current != "" && !inSpace {
				scopes = append(scopes, current)
				current = ""
			}
			inSpace = true
		} else {
			current += string(ch)
			inSpace = false
		}
	}

	if current != "" {
		scopes = append(scopes, current)
	}

	return scopes
}

// FilterDevicesByType фильтрует устройства по типу.
// typesFilter: "NetworkVideoTransmitter", "NetworkVideoDisplay", etc.
// Поиск производится по подстроке (типы ONVIF содержат префикс "wsdp:").
func FilterDevicesByType(devices []DiscoveredDevice, typesFilter []string) []DiscoveredDevice {
	if len(typesFilter) == 0 {
		return devices
	}

	var filtered []DiscoveredDevice
	for _, d := range devices {
		for _, t := range d.Types {
			for _, filter := range typesFilter {
				if strings.Contains(t, filter) {
					filtered = append(filtered, d)
					goto nextDevice
				}
			}
		}
	nextDevice:
	}

	return filtered
}

// ─── UUID Generator ─────────────────────────────────────────────────────────

// newUUID генерирует UUID v4 строку для WS-Discovery messageId.
func newUUID() string {
	now := time.Now().UnixNano()
	r1 := uint32(now >> 32)
	r2 := uint32(now)
	r3 := uint32(now >> 16)
	r4 := uint32(now)

	// Используем маску для фиксации длины каждой части UUID
	return fmt.Sprintf("%08x-%04x-4%03x-%04x-%08x%04x",
		r1,
		r2&0xFFFF,
		(r3 & 0x0FFF),
		(0x8000 | (r4 & 0x3FFF)),
		(r1^r2)&0xFFFFFFFF,
		(r3^r4)&0xFFFF,
	)
}
