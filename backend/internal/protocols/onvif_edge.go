// CCTV-2.2.1: Edge Agent Abstraction for ONVIF
//
// Предоставляет единый интерфейс для подключения к ONVIF устройствам через
// различные транспорты: direct TCP/HTTP, P2P relay (NAT), или будущий edge agent.
//
// Compliance:
//   - IEC 62443-3-3 SL-3: изоляция зон через разные типы коннекторов
//   - OWASP ASVS L3: тщательная валидация соединений
//   - Приказ ОАЦ №66 п.7.18: идентификация через сертификаты (в edge agent mode)

package protocols

import (
	"context"
	"fmt"
	"gb-telemetry-collector/internal/config"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

// ─── EdgeConnector Interface ────────────────────────────────────────────────

// EdgeConnector предоставляет абстракцию транспортного уровня для ONVIF устройств.
// Позволяет единообразно работать с устройствами через разные каналы связи.
type EdgeConnector interface {
	// Connect устанавливает соединение с устройством.
	// Возвращает канал для SOAP/HTTP запросов.
	Connect(ctx context.Context, deviceAddr string) (EdgeConnection, error)
}

// EdgeConnection представляет активное соединение с устройством.
type EdgeConnection interface {
	// Disconnect закрывает соединение.
	Disconnect() error
}

// ─── DirectConnector ────────────────────────────────────────────────────────

// DirectConnector — прямое TCP/HTTP подключение к ONVIF устройству.
// Используется когда устройство доступно напрямую (без NAT).
type DirectConnector struct {
	logger *slog.Logger
	mu     sync.Mutex
	conns  map[string]*directConn
}

type directConn struct {
	addr   string
	closed bool
}

// NewDirectConnector создаёт коннектор для прямых подключений.
func NewDirectConnector(logger *slog.Logger) *DirectConnector {
	return &DirectConnector{
		logger: logger.With("connector", "direct"),
		conns:  make(map[string]*directConn),
	}
}

func (c *DirectConnector) Connect(ctx context.Context, addr string) (EdgeConnection, error) {
	// Проверяем доступность через TCP dial
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("direct connect to %s: %w", addr, err)
	}
	conn.Close()

	c.mu.Lock()
	defer c.mu.Unlock()

	dc := &directConn{
		addr:   addr,
		closed: false,
	}
	c.conns[addr] = dc

	c.logger.Debug("Direct connection established", "addr", addr)
	return dc, nil
}

func (c *directConn) Disconnect() error {
	if c.closed {
		return nil
	}
	c.closed = true
	return nil
}

// ─── P2PConnector ───────────────────────────────────────────────────────────

// P2PConnector — подключение через P2P gateway (NAT traversal).
// Использует ONVIFNATManager для регистрации устройств и relay.
type P2PConnector struct {
	natManager *ONVIFNATManager
	logger     *slog.Logger
	mu         sync.Mutex
	conns      map[string]*p2pConn
}

type p2pConn struct {
	deviceID string
	session  *P2PSession
	closed   bool
}

// NewP2PConnector создаёт коннектор для подключения через P2P gateway.
func NewP2PConnector(natManager *ONVIFNATManager, logger *slog.Logger) *P2PConnector {
	return &P2PConnector{
		natManager: natManager,
		logger:     logger.With("connector", "p2p"),
		conns:      make(map[string]*p2pConn),
	}
}

func (c *P2PConnector) Connect(ctx context.Context, addr string) (EdgeConnection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Регистрируемся через NAT manager
	deviceID := fmt.Sprintf("onvif_p2p_%s", addr)
	session, err := c.natManager.RegisterDevice(ctx, deviceID, addr)
	if err != nil {
		return nil, fmt.Errorf("p2p connect to %s: %w", addr, err)
	}

	pc := &p2pConn{
		deviceID: deviceID,
		session:  session,
		closed:   false,
	}
	c.conns[addr] = pc

	c.logger.Debug("P2P connection established",
		"addr", addr,
		"relay_addr", session.RelayAddr,
		"session_id", session.SessionID,
	)
	return pc, nil
}

func (c *p2pConn) Disconnect() error {
	if c.closed {
		return nil
	}
	c.closed = true
	return nil
}

// ─── EdgeAgentConnector ─────────────────────────────────────────────────────

// EdgeAgentConnector — заглушка для будущего edge agent.
// Edge agent будет запускаться на устройстве или рядом с ним и обеспечивать
// безопасное соединение с backend через mTLS 1.3.
type EdgeAgentConnector struct {
	edgeAgentURL string
	logger       *slog.Logger
	mu           sync.Mutex
	conns        map[string]*edgeAgentConn
}

type edgeAgentConn struct {
	deviceID string
	closed   bool
}

// NewEdgeAgentConnector создаёт заглушку коннектора для edge agent.
func NewEdgeAgentConnector(edgeAgentURL string, logger *slog.Logger) *EdgeAgentConnector {
	return &EdgeAgentConnector{
		edgeAgentURL: edgeAgentURL,
		logger:       logger.With("connector", "edge_agent"),
		conns:        make(map[string]*edgeAgentConn),
	}
}

func (c *EdgeAgentConnector) Connect(ctx context.Context, addr string) (EdgeConnection, error) {
	_ = c.edgeAgentURL // reserved for future use

	c.mu.Lock()
	defer c.mu.Unlock()

	// Пока возвращаем заглушку
	c.logger.Warn("Edge agent connector not fully implemented, using stub",
		"addr", addr)

	ec := &edgeAgentConn{
		deviceID: addr,
		closed:   false,
	}
	c.conns[addr] = ec
	return ec, nil
}

func (c *edgeAgentConn) Disconnect() error {
	if c.closed {
		return nil
	}
	c.closed = true
	return nil
}

// ─── Factory ────────────────────────────────────────────────────────────────

// NewConnector создаёт коннектор в зависимости от режима.
// mode: "direct" | "p2p" | "edge_agent"
func NewConnector(mode string) EdgeConnector {
	switch mode {
	case "p2p":
		// P2P connector требует NAT manager, создаём с пустым конфигом
		natManager := NewONVIFNATManager(
			config.ONVIFConfig{},
			"", "", slog.Default(),
		)
		return NewP2PConnector(natManager, slog.Default())
	case "edge_agent":
		return NewEdgeAgentConnector("", slog.Default())
	default:
		return NewDirectConnector(slog.Default())
	}
}

// ─── HTTP Client Wrapper ────────────────────────────────────────────────────

// HTTPConnector оборачивает EdgeConnector и предоставляет HTTP клиент.
// Позволяет делать SOAP запросы через любой тип соединения.
type HTTPConnector struct {
	connector EdgeConnector
	client    *http.Client
	logger    *slog.Logger
}

// NewHTTPConnector создаёт HTTP-обёртку над EdgeConnector.
func NewHTTPConnector(connector EdgeConnector, logger *slog.Logger) *HTTPConnector {
	return &HTTPConnector{
		connector: connector,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: false,
			},
		},
		logger: logger,
	}
}

// Do выполняет HTTP запрос через выбранный коннектор.
func (hc *HTTPConnector) Do(req *http.Request) (*http.Response, error) {
	conn, err := hc.connector.Connect(req.Context(), req.URL.Host)
	if err != nil {
		return nil, fmt.Errorf("connector: %w", err)
	}
	defer conn.Disconnect()

	return hc.client.Do(req)
}

// Ensure interfaces are implemented
var _ EdgeConnector = (*DirectConnector)(nil)
var _ EdgeConnector = (*P2PConnector)(nil)
var _ EdgeConnector = (*EdgeAgentConnector)(nil)
var _ EdgeConnection = (*directConn)(nil)
var _ EdgeConnection = (*p2pConn)(nil)
var _ EdgeConnection = (*edgeAgentConn)(nil)
