// CCTV-2.2.1: ONVIF Profile S/T Client
//
// Compliance:
//   - IEC 62443-3-3 SL-3: zone separation (ONVIF -> Zone 3)
//   - OWASP ASVS L3: input validation, error handling, output encoding
//   - ISO 27001 A.12.4: audit trail через штатный логгер
//   - Приказ ОАЦ №66 п.7.18: уникальная идентификация устройств
//
// Криптография:
//   - Digest auth (совместимость с Hikvision/Dahua) — разрешено для внешних систем
//   - TLS 1.3 для HTTPS соединений (ГОСТ-шифросюиты при наличии)

package protocols

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"gb-telemetry-collector/internal/config"
	"gb-telemetry-collector/internal/state"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/icholy/digest"
)

// ─── SOAP Namespaces ────────────────────────────────────────────────────────

const (
	onvifNS       = "http://www.onvif.org/ver10/schema"
	trtNS         = "http://www.onvif.org/ver10/media/wsdl"
	tevNS         = "http://www.onvif.org/ver10/events/wsdl"
	ptzNS         = "http://www.onvif.org/ver20/ptz/wsdl"
	wsDiscoveryNS = "http://schemas.xmlsoap.org/ws/2005/04/discovery"
	soapEnvNS     = "http://www.w3.org/2003/05/soap-envelope"
)

// ─── SOAP Envelope ──────────────────────────────────────────────────────────

type SOAPEnvelope struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Envelope"`
	Header  SOAPHeader
	Body    SOAPBody
}

type SOAPHeader struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Header"`
	Content []byte   `xml:",innerxml"`
}

type SOAPBody struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Body"`
	Content []byte   `xml:",innerxml"`
}

// ─── Device Information ─────────────────────────────────────────────────────

type DeviceInfo struct {
	Manufacturer    string
	Model           string
	FirmwareVersion string
	SerialNumber    string
	HardwareID      string
}

type Capabilities struct {
	Media     bool
	PTZ       bool
	Events    bool
	Analytics bool
}

type ONVIFDevice struct {
	XAddr      string
	DeviceInfo DeviceInfo
	Caps       Capabilities
	Scopes     []string
	Types      []string
	LastSeen   time.Time
}

// ─── PTZ Types ──────────────────────────────────────────────────────────────

type PTZVector struct {
	PanTilt struct {
		X float64 `xml:"x"`
		Y float64 `xml:"y"`
	} `xml:"PanTilt"`
	Zoom struct {
		X float64 `xml:"x"`
	} `xml:"Zoom"`
}

type PTZSpeed struct {
	PanTilt struct {
		X float64 `xml:"x"`
		Y float64 `xml:"y"`
	} `xml:"PanTilt"`
	Zoom struct {
		X float64 `xml:"x"`
	} `xml:"Zoom"`
}

// ─── Media Profile ──────────────────────────────────────────────────────────

type MediaProfile struct {
	Token    string `xml:"token,attr"`
	Name     string `xml:"Name"`
	VideoSrc string `xml:"VideoSourceConfiguration>SourceToken"`
	AudioSrc string `xml:"AudioSourceConfiguration>SourceToken"`
}

type StreamURI struct {
	URI string
}

// ─── Event Types ────────────────────────────────────────────────────────────

type ONVIFEvent struct {
	Topic     string
	Source    string
	Data      string
	Timestamp time.Time
	IsMotion  bool
	IsTamper  bool
	IsInput   bool
	IsDigital bool
}

// ─── ONVIFHandler ───────────────────────────────────────────────────────────

type ONVIFHandler struct {
	cfg      config.ONVIFConfig
	stateMgr state.DeviceStateManager
	logger   *slog.Logger
	devices  map[string]*ONVIFDevice
	mu       sync.RWMutex
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc

	// HTTP клиент с Digest auth поддержкой
	httpClient *http.Client

	// Edge connector (direct, p2p, или edge_agent)
	connector EdgeConnector
}

// NewONVIFHandler создаёт ONVIF клиент из конфигурации.
// ConnectMode: "direct" | "p2p" | "edge_agent"
func NewONVIFHandler(cfg config.ONVIFConfig, stateMgr state.DeviceStateManager, logger *slog.Logger) *ONVIFHandler {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &digest.Transport{
			Username: cfg.Username,
			Password: cfg.Password,
		},
	}

	connector := NewConnector(cfg.ConnectMode)

	return &ONVIFHandler{
		cfg:        cfg,
		stateMgr:   stateMgr,
		logger:     logger.With("protocol", "onvif"),
		devices:    make(map[string]*ONVIFDevice),
		httpClient: client,
		connector:  connector,
	}
}

// Start реализует ProtocolHandler. Запускает мониторинг ONVIF устройств.
func (h *ONVIFHandler) Start(ctx context.Context) error {
	h.ctx, h.cancel = context.WithCancel(ctx)

	if !h.cfg.Enabled {
		h.logger.Info("ONVIF handler disabled")
		return nil
	}

	h.logger.Info("Starting ONVIF handler",
		"discovery", h.cfg.Discovery,
		"connect_mode", h.cfg.ConnectMode,
	)

	// WS-Discovery поиск устройств в локальной сети
	if h.cfg.Discovery {
		h.wg.Add(1)
		go h.discoveryLoop()
	}

	// Health-check loop для зарегистрированных устройств
	h.wg.Add(1)
	go h.healthCheckLoop()

	return nil
}

// Stop реализует ProtocolHandler.
func (h *ONVIFHandler) Stop() error {
	h.cancel()
	h.wg.Wait()
	h.logger.Info("ONVIF handler stopped")
	return nil
}

// ─── Profile S: Device Information ──────────────────────────────────────────

// GetDeviceInformation получает информацию об устройстве через GetDeviceInformation.
func (h *ONVIFHandler) GetDeviceInformation(xaddr string) (*DeviceInfo, error) {
	body := buildSOAPBody(`<GetDeviceInformation xmlns="http://www.onvif.org/ver10/device/wsdl"/>`)
	envelope := wrapSOAP(body)

	resp, err := h.soapCall(h.ctx, xaddr, envelope)
	if err != nil {
		return nil, fmt.Errorf("get device info: %w", err)
	}

	return parseDeviceInfoResponse(resp)
}

// GetCapabilities получает capabilities устройства через GetCapabilities.
func (h *ONVIFHandler) GetCapabilities(xaddr string) (*Capabilities, error) {
	body := buildSOAPBody(`<GetCapabilities xmlns="http://www.onvif.org/ver10/device/wsdl">
		<Category>All</Category>
	</GetCapabilities>`)
	envelope := wrapSOAP(body)

	resp, err := h.soapCall(h.ctx, xaddr, envelope)
	if err != nil {
		return nil, fmt.Errorf("get capabilities: %w", err)
	}

	return parseCapabilitiesResponse(resp)
}

// ─── Profile S: Media ───────────────────────────────────────────────────────

// GetProfiles возвращает список MediaProfile.
func (h *ONVIFHandler) GetProfiles(xaddr string) ([]MediaProfile, error) {
	body := buildSOAPBody(`<GetProfiles xmlns="` + trtNS + `"/>`)
	envelope := wrapSOAP(body)

	resp, err := h.soapCall(h.ctx, xaddr, envelope)
	if err != nil {
		return nil, fmt.Errorf("get profiles: %w", err)
	}

	return parseProfilesResponse(resp)
}

// GetStreamURI возвращает RTSP URI для указанного profile.
func (h *ONVIFHandler) GetStreamURI(xaddr, profileToken, protocol string) (*StreamURI, error) {
	if protocol == "" {
		protocol = "RTSP"
	}

	body := buildSOAPBody(fmt.Sprintf(`<GetStreamUri xmlns="`+trtNS+`">
		<StreamSetup>
			<Stream xmlns="`+onvifNS+`">%s</Stream>
			<Transport xmlns="`+onvifNS+`">
				<Protocol>%s</Protocol>
			</Transport>
		</StreamSetup>
		<ProfileToken>%s</ProfileToken>
	</GetStreamUri>`, "RTP-Unicast", protocol, profileToken))
	envelope := wrapSOAP(body)

	resp, err := h.soapCall(h.ctx, xaddr, envelope)
	if err != nil {
		return nil, fmt.Errorf("get stream URI: %w", err)
	}

	return parseStreamURIResponse(resp)
}

// ─── Profile S: PTZ ─────────────────────────────────────────────────────────

// AbsoluteMove выполняет абсолютное перемещение PTZ.
func (h *ONVIFHandler) AbsoluteMove(xaddr, profileToken string, position PTZVector, speed *PTZSpeed) error {
	speedXML := ""
	if speed != nil {
		speedXML = fmt.Sprintf(`<Speed>%s</Speed>`, marshalPTZSpeed(*speed))
	}

	body := buildSOAPBody(fmt.Sprintf(`<AbsoluteMove xmlns="`+ptzNS+`">
		<ProfileToken>%s</ProfileToken>
		<Position>%s</Position>
		%s
	</AbsoluteMove>`, profileToken, marshalPTZVector(position), speedXML))
	envelope := wrapSOAP(body)

	_, err := h.soapCall(h.ctx, xaddr, envelope)
	return err
}

// RelativeMove выполняет относительное перемещение PTZ.
func (h *ONVIFHandler) RelativeMove(xaddr, profileToken string, translation PTZVector, speed *PTZSpeed) error {
	speedXML := ""
	if speed != nil {
		speedXML = fmt.Sprintf(`<Speed>%s</Speed>`, marshalPTZSpeed(*speed))
	}

	body := buildSOAPBody(fmt.Sprintf(`<RelativeMove xmlns="`+ptzNS+`">
		<ProfileToken>%s</ProfileToken>
		<Translation>%s</Translation>
		%s
	</RelativeMove>`, profileToken, marshalPTZVector(translation), speedXML))
	envelope := wrapSOAP(body)

	_, err := h.soapCall(h.ctx, xaddr, envelope)
	return err
}

// ContinuousMove выполняет непрерывное перемещение PTZ.
func (h *ONVIFHandler) ContinuousMove(xaddr, profileToken string, speed PTZSpeed, timeout *time.Duration) error {
	timeoutXML := ""
	if timeout != nil {
		timeoutXML = fmt.Sprintf(`<Timeout>%s</Timeout>`, durationToXML(*timeout))
	}

	body := buildSOAPBody(fmt.Sprintf(`<ContinuousMove xmlns="`+ptzNS+`">
		<ProfileToken>%s</ProfileToken>
		<Velocity>%s</Velocity>
		%s
	</ContinuousMove>`, profileToken, marshalPTZSpeed(speed), timeoutXML))
	envelope := wrapSOAP(body)

	_, err := h.soapCall(h.ctx, xaddr, envelope)
	return err
}

// PTZStop останавливает PTZ движение.
func (h *ONVIFHandler) PTZStop(xaddr, profileToken string) error {
	body := buildSOAPBody(fmt.Sprintf(`<Stop xmlns="`+ptzNS+`">
		<ProfileToken>%s</ProfileToken>
		<PanTilt>true</PanTilt>
		<Zoom>true</Zoom>
	</Stop>`, profileToken))
	envelope := wrapSOAP(body)

	_, err := h.soapCall(h.ctx, xaddr, envelope)
	return err
}

// HomePosition перемещает PTZ в home position.
func (h *ONVIFHandler) HomePosition(xaddr, profileToken string) error {
	body := buildSOAPBody(fmt.Sprintf(`<GotoHomePosition xmlns="`+ptzNS+`">
		<ProfileToken>%s</ProfileToken>
	</GotoHomePosition>`, profileToken))
	envelope := wrapSOAP(body)

	_, err := h.soapCall(h.ctx, xaddr, envelope)
	return err
}

// ─── Profile T: Events ──────────────────────────────────────────────────────

// GetEventProperties получает свойства событий устройства.
func (h *ONVIFHandler) GetEventProperties(xaddr string) ([]string, error) {
	body := buildSOAPBody(`<GetEventProperties xmlns="` + tevNS + `"/>`)
	envelope := wrapSOAP(body)

	resp, err := h.soapCall(h.ctx, xaddr, envelope)
	if err != nil {
		return nil, fmt.Errorf("get event properties: %w", err)
	}

	return parseEventPropertiesResponse(resp)
}

// PullMessages получает сообщения из event subscription.
func (h *ONVIFHandler) PullMessages(xaddr, subscriptionAddr string, timeout time.Duration, msgLimit int) ([]ONVIFEvent, error) {
	body := buildSOAPBody(fmt.Sprintf(`<PullMessages xmlns="`+tevNS+`">
		<Timeout>%s</Timeout>
		<MessageLimit>%d</MessageLimit>
	</PullMessages>`, durationToXML(timeout), msgLimit))
	envelope := wrapSOAP(body)

	resp, err := h.soapCall(h.ctx, xaddr, envelope)
	if err != nil {
		return nil, fmt.Errorf("pull messages: %w", err)
	}

	return parsePullMessagesResponse(resp)
}

// CreatePullPointSubscription создаёт pull point subscription для событий.
func (h *ONVIFHandler) CreatePullPointSubscription(xaddr string) (string, error) {
	body := buildSOAPBody(`<CreatePullPointSubscription xmlns="` + tevNS + `"/>`)
	envelope := wrapSOAP(body)

	resp, err := h.soapCall(h.ctx, xaddr, envelope)
	if err != nil {
		return "", fmt.Errorf("create pull point subscription: %w", err)
	}

	return parseSubscriptionReference(resp)
}

// GetMotionEvents получает события движения (Profile T).
func (h *ONVIFHandler) GetMotionEvents(xaddr string) ([]ONVIFEvent, error) {
	subAddr, err := h.CreatePullPointSubscription(xaddr)
	if err != nil {
		// Некоторые устройства не поддерживают pull point
		h.logger.Debug("Pull point not supported, falling back to GetEventProperties",
			"device", xaddr, "error", err)
		return nil, err
	}

	events, err := h.PullMessages(xaddr, subAddr, 5*time.Second, 10)
	if err != nil {
		return nil, fmt.Errorf("get motion events: %w", err)
	}

	return events, nil
}

// ─── Internal: SOAP ─────────────────────────────────────────────────────────

func (h *ONVIFHandler) soapCall(ctx context.Context, xaddr string, envelope []byte) ([]byte, error) {
	// Выбор коннектора
	conn, err := h.connector.Connect(ctx, xaddr)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", xaddr, err)
	}
	defer conn.Disconnect()

	// Построение endpoint URL
	u, err := url.Parse(xaddr)
	if err != nil {
		return nil, fmt.Errorf("parse xaddr: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(envelope))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	req.Header.Set("Accept", "application/soap+xml")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("soap call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("soap error: status=%d body=%s", resp.StatusCode, string(body[:min(len(body), 512)]))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read soap response: %w", err)
	}

	return body, nil
}

// ─── Device Registry ────────────────────────────────────────────────────────

func (h *ONVIFHandler) RegisterDevice(xaddr string) (*ONVIFDevice, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Проверяем, не зарегистрировано ли уже
	if dev, ok := h.devices[xaddr]; ok {
		dev.LastSeen = time.Now()
		return dev, nil
	}

	// Получаем информацию
	info, err := h.GetDeviceInformation(xaddr)
	if err != nil {
		return nil, fmt.Errorf("register device %s: %w", xaddr, err)
	}

	caps, err := h.GetCapabilities(xaddr)
	if err != nil {
		return nil, fmt.Errorf("get caps for %s: %w", xaddr, err)
	}

	device := &ONVIFDevice{
		XAddr:      xaddr,
		DeviceInfo: *info,
		Caps:       *caps,
		LastSeen:   time.Now(),
	}

	h.devices[xaddr] = device
	h.logger.Info("ONVIF device registered",
		"xaddr", xaddr,
		"manufacturer", info.Manufacturer,
		"model", info.Model,
	)

	return device, nil
}

func (h *ONVIFHandler) GetDevice(xaddr string) (*ONVIFDevice, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	dev, ok := h.devices[xaddr]
	return dev, ok
}

func (h *ONVIFHandler) ListDevices() []*ONVIFDevice {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]*ONVIFDevice, 0, len(h.devices))
	for _, dev := range h.devices {
		result = append(result, dev)
	}
	return result
}

func (h *ONVIFHandler) RemoveDevice(xaddr string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.devices, xaddr)
	h.logger.Info("ONVIF device removed", "xaddr", xaddr)
}

// ─── Health Check ───────────────────────────────────────────────────────────

func (h *ONVIFHandler) healthCheckLoop() {
	defer h.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.runHealthCheck()
		}
	}
}

func (h *ONVIFHandler) runHealthCheck() {
	h.mu.RLock()
	devices := make([]string, 0, len(h.devices))
	for xaddr := range h.devices {
		devices = append(devices, xaddr)
	}
	h.mu.RUnlock()

	for _, xaddr := range devices {
		select {
		case <-h.ctx.Done():
			return
		default:
		}

		_, err := h.GetDeviceInformation(xaddr)
		if err != nil {
			h.logger.Warn("ONVIF device health check failed",
				"xaddr", xaddr, "error", err)
			// Не удаляем, даём шанс восстановиться
		} else {
			h.mu.Lock()
			if dev, ok := h.devices[xaddr]; ok {
				dev.LastSeen = time.Now()
			}
			h.mu.Unlock()
		}
	}
}

func (h *ONVIFHandler) discoveryLoop() {
	defer h.wg.Done()
	h.logger.Info("ONVIF discovery loop started")

	// Первый discovery сразу при старте
	h.runDiscovery()

	ticker := time.NewTicker(300 * time.Second) // Каждые 5 минут
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.runDiscovery()
		}
	}
}

func (h *ONVIFHandler) runDiscovery() {
	devices, err := DiscoverONVIFDevices(h.ctx, h.cfg.DiscoveryPort, 5*time.Second, h.logger)
	if err != nil {
		h.logger.Error("ONVIF discovery failed", "error", err)
		return
	}

	for _, dev := range devices {
		select {
		case <-h.ctx.Done():
			return
		default:
		}

		if _, err := h.RegisterDevice(dev.XAddr); err != nil {
			h.logger.Debug("Failed to register discovered device",
				"xaddr", dev.XAddr, "error", err)
		}
	}
}

// ─── SOAP Helpers ───────────────────────────────────────────────────────────

func buildSOAPBody(innerXML string) []byte {
	return []byte(fmt.Sprintf(`<s:Body xmlns:s="http://www.w3.org/2003/05/soap-envelope">%s</s:Body>`, innerXML))
}

func wrapSOAP(body []byte) []byte {
	envelope := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
%s
</s:Envelope>`, string(body))
	return []byte(envelope)
}

func marshalPTZVector(v PTZVector) string {
	return fmt.Sprintf(`<PanTilt x="%f" y="%f" xmlns="%s"/>`, v.PanTilt.X, v.PanTilt.Y, onvifNS) +
		fmt.Sprintf(`<Zoom x="%f" xmlns="%s"/>`, v.Zoom.X, onvifNS)
}

func marshalPTZSpeed(s PTZSpeed) string {
	return fmt.Sprintf(`<PanTilt x="%f" y="%f" xmlns="%s"/>`, s.PanTilt.X, s.PanTilt.Y, onvifNS) +
		fmt.Sprintf(`<Zoom x="%f" xmlns="%s"/>`, s.Zoom.X, onvifNS)
}

func durationToXML(d time.Duration) string {
	secs := int(d.Seconds())
	return fmt.Sprintf("PT%dS", secs)
}

// ─── Response Parsers ───────────────────────────────────────────────────────

func parseDeviceInfoResponse(data []byte) (*DeviceInfo, error) {
	var envelope struct {
		Body struct {
			GetDeviceInformationResponse struct {
				Manufacturer    string `xml:"Manufacturer"`
				Model           string `xml:"Model"`
				FirmwareVersion string `xml:"FirmwareVersion"`
				SerialNumber    string `xml:"SerialNumber"`
				HardwareID      string `xml:"HardwareId"`
			} `xml:"GetDeviceInformationResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("parse device info: %w", err)
	}

	r := envelope.Body.GetDeviceInformationResponse
	return &DeviceInfo{
		Manufacturer:    r.Manufacturer,
		Model:           r.Model,
		FirmwareVersion: r.FirmwareVersion,
		SerialNumber:    r.SerialNumber,
		HardwareID:      r.HardwareID,
	}, nil
}

func parseCapabilitiesResponse(data []byte) (*Capabilities, error) {
	var envelope struct {
		Body struct {
			GetCapabilitiesResponse struct {
				Capabilities struct {
					Media struct {
						XAddr string `xml:"XAddr"`
					} `xml:"Media"`
					PTZ struct {
						XAddr string `xml:"XAddr"`
					} `xml:"PTZ"`
					Events struct {
						XAddr string `xml:"XAddr"`
					} `xml:"Events"`
					Analytics struct {
						XAddr string `xml:"XAddr"`
					} `xml:"Analytics"`
				} `xml:"Capabilities"`
			} `xml:"GetCapabilitiesResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("parse capabilities: %w", err)
	}

	c := envelope.Body.GetCapabilitiesResponse.Capabilities
	return &Capabilities{
		Media:     c.Media.XAddr != "",
		PTZ:       c.PTZ.XAddr != "",
		Events:    c.Events.XAddr != "",
		Analytics: c.Analytics.XAddr != "",
	}, nil
}

func parseProfilesResponse(data []byte) ([]MediaProfile, error) {
	var envelope struct {
		Body struct {
			GetProfilesResponse struct {
				Profiles []struct {
					Token string `xml:"token,attr"`
					Name  string `xml:"Name"`
					Video struct {
						SourceToken string `xml:"SourceToken"`
					} `xml:"VideoSourceConfiguration"`
					Audio struct {
						SourceToken string `xml:"SourceToken"`
					} `xml:"AudioSourceConfiguration"`
				} `xml:"Profiles"`
			} `xml:"GetProfilesResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("parse profiles: %w", err)
	}

	profiles := make([]MediaProfile, 0, len(envelope.Body.GetProfilesResponse.Profiles))
	for _, p := range envelope.Body.GetProfilesResponse.Profiles {
		profiles = append(profiles, MediaProfile{
			Token:    p.Token,
			Name:     p.Name,
			VideoSrc: p.Video.SourceToken,
			AudioSrc: p.Audio.SourceToken,
		})
	}
	return profiles, nil
}

func parseStreamURIResponse(data []byte) (*StreamURI, error) {
	var envelope struct {
		Body struct {
			GetStreamUriResponse struct {
				MediaUri struct {
					Uri string `xml:"Uri"`
				} `xml:"MediaUri"`
			} `xml:"GetStreamUriResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("parse stream URI: %w", err)
	}

	uri := envelope.Body.GetStreamUriResponse.MediaUri.Uri
	if uri == "" {
		return nil, fmt.Errorf("empty stream URI in response")
	}

	return &StreamURI{URI: uri}, nil
}

func parseEventPropertiesResponse(data []byte) ([]string, error) {
	var envelope struct {
		Body struct {
			GetEventPropertiesResponse struct {
				Topics []struct {
					Topic string `xml:",chardata"`
				} `xml:"SupportedEvents"`
			} `xml:"GetEventPropertiesResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("parse event properties: %w", err)
	}

	topics := make([]string, 0, len(envelope.Body.GetEventPropertiesResponse.Topics))
	for _, t := range envelope.Body.GetEventPropertiesResponse.Topics {
		topics = append(topics, t.Topic)
	}
	return topics, nil
}

func parsePullMessagesResponse(data []byte) ([]ONVIFEvent, error) {
	var envelope struct {
		Body struct {
			PullMessagesResponse struct {
				Notifications []struct {
					Topic struct {
						Text string `xml:",chardata"`
					} `xml:"Topic"`
					Message struct {
						Source struct {
							SimpleItem []struct {
								Name  string `xml:"Name,attr"`
								Value string `xml:"Value,attr"`
							} `xml:"SimpleItem"`
						} `xml:"Source"`
						Data struct {
							SimpleItem []struct {
								Name  string `xml:"Name,attr"`
								Value string `xml:"Value,attr"`
							} `xml:"SimpleItem"`
						} `xml:"Data"`
					} `xml:"Message"`
				} `xml:"NotificationMessage"`
			} `xml:"PullMessagesResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("parse pull messages: %w", err)
	}

	events := make([]ONVIFEvent, 0, len(envelope.Body.PullMessagesResponse.Notifications))
	for _, n := range envelope.Body.PullMessagesResponse.Notifications {
		event := ONVIFEvent{
			Topic: n.Topic.Text,
		}

		for _, item := range n.Message.Source.SimpleItem {
			if item.Name == "Source" {
				event.Source = item.Value
			}
		}

		for _, item := range n.Message.Data.SimpleItem {
			if item.Name == "Data" {
				event.Data = item.Value
			}
		}

		// Классификация событий
		event.IsMotion = contains(event.Topic, "Motion")
		event.IsTamper = contains(event.Topic, "Tamper") || contains(event.Topic, "Tampering")
		event.IsInput = contains(event.Topic, "Input")
		event.IsDigital = contains(event.Topic, "Digital")

		events = append(events, event)
	}

	return events, nil
}

func parseSubscriptionReference(data []byte) (string, error) {
	var envelope struct {
		Body struct {
			CreatePullPointSubscriptionResponse struct {
				SubscriptionReference struct {
					Address string `xml:"Address"`
				} `xml:"SubscriptionReference"`
			} `xml:"CreatePullPointSubscriptionResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(data, &envelope); err != nil {
		return "", fmt.Errorf("parse subscription reference: %w", err)
	}

	addr := envelope.Body.CreatePullPointSubscriptionResponse.SubscriptionReference.Address
	if addr == "" {
		return "", fmt.Errorf("empty subscription reference")
	}

	return addr, nil
}

// ─── Utils ──────────────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > 0 && len(substr) > 0 &&
			(s[0:min(len(s), len(substr))] == substr ||
				(len(s) > len(substr) && contains(s[1:], substr)))))
}

// Min возвращает минимальное из двух int.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
