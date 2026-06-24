package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// ServiceStatusDetail — статус одного сервиса.
type ServiceStatusDetail struct {
	Status  string `json:"status"` // "running" | "stopped" | "error" | "disabled"
	Port    int    `json:"port"`
	Message string `json:"message,omitempty"`
}

// ServicesStatusResponse — ответ со статусами всех сервисов.
type ServicesStatusResponse struct {
	Services map[string]ServiceStatusDetail `json:"services"`
}

// mountServicesStatusRoute регистрирует маршрут проверки статуса сервисов.
func (s *Server) mountServicesStatusRoute(r chi.Router) {
	r.Get("/api/v1/settings/services/status", s.handleServicesStatus)
}

// parseDBObj парсит DB-настройку как map[string]interface{}.
func parseDBObj(raw json.RawMessage) map[string]interface{} {
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	return obj
}

// getFloat64 удобно достаёт float64 из map (числа из JSON всегда float64).
func getFloat64(m map[string]interface{}, key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	f, ok := v.(float64)
	return f, ok
}

// getBool удобно достаёт bool из map.
func getBool(m map[string]interface{}, key string) (bool, bool) {
	v, ok := m[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// getString удобно достаёт string из map.
func getString(m map[string]interface{}, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// handleServicesStatus возвращает статус для каждого сервиса на основе реально запущенных процессов.
// Приоритет: 1) config (где сервисы реально запущены), 2) DB (пользовательские настройки).
func (s *Server) handleServicesStatus(w http.ResponseWriter, r *http.Request) {
	cfg := s.config
	response := ServicesStatusResponse{
		Services: make(map[string]ServiceStatusDetail),
	}

	settingsFromDB, _ := s.db.GetSystemSettings()
	checkTimeout := 2 * time.Second

	// Вспомогательная функция: получает настройки сервиса из БД или nil
	getDBObj := func(serviceKey string) map[string]interface{} {
		raw, ok := settingsFromDB[serviceKey]
		if !ok {
			return nil
		}
		return parseDBObj(raw)
	}

	// ── Syslog (Log Server) ───────────────────────────────
	// Реальный порт из config.go: log_server_port (default 515)
	svcPort := cfg.LogServerPort
	if svcPort == 0 {
		svcPort = 515
	}
	svcEnabled := true // log server всегда запускается в main.go (line 227)
	status, msg := checkServiceStatus(svcEnabled, "127.0.0.1", svcPort, checkTimeout)
	response.Services["syslog"] = ServiceStatusDetail{Status: status, Port: svcPort, Message: msg}

	// ── FTP ───────────────────────────────────────────────
	svcPort = cfg.FTP.Port
	if svcPort == 0 {
		svcPort = 2121
	}
	svcEnabled = cfg.FTP.Enabled
	if db := getDBObj("services_ftp"); db != nil {
		if e, ok := getBool(db, "enabled"); ok {
			svcEnabled = e
		}
	}
	status, msg = checkServiceStatus(svcEnabled, "127.0.0.1", svcPort, checkTimeout)
	response.Services["ftp"] = ServiceStatusDetail{Status: status, Port: svcPort, Message: msg}

	// ── SNMP ──────────────────────────────────────────────
	svcPort = cfg.SNMP.Port
	if svcPort == 0 {
		svcPort = 162
	}
	svcEnabled = cfg.SNMP.Enabled
	status, msg = checkServiceStatus(svcEnabled, "127.0.0.1", svcPort, checkTimeout)
	response.Services["snmp"] = ServiceStatusDetail{Status: status, Port: svcPort, Message: msg}

	// ── HTTP Log Receiver ─────────────────────────────────
	// HTTP-логгер НЕ запускается по умолчанию (HTTPEnabled: false в main.go)
	// Проверяем через config или DB
	svcEnabled = false
	svcPort = 8083
	if db := getDBObj("services_http"); db != nil {
		if e, ok := getBool(db, "enabled"); ok {
			svcEnabled = e
		}
		if p, ok := getFloat64(db, "port"); ok {
			svcPort = int(p)
		}
	}
	// HTTP-логгер стартует, только если cfg.HTTPXMLEnabled || logCfg.HTTPEnabled
	if cfg.HTTPXMLEnabled {
		svcEnabled = true
	}
	status, msg = checkServiceStatus(svcEnabled, "127.0.0.1", svcPort, checkTimeout)
	response.Services["http"] = ServiceStatusDetail{Status: status, Port: svcPort, Message: msg}

	// ── Dahua Private Protocol ────────────────────────────
	svcEnabled = cfg.Dahua.Enabled
	svcPort = 37777
	if len(cfg.Dahua.Ports) > 0 {
		svcPort = cfg.Dahua.Ports[0]
	}
	status, msg = checkServiceStatus(svcEnabled, "127.0.0.1", svcPort, checkTimeout)
	response.Services["dahua"] = ServiceStatusDetail{Status: status, Port: svcPort, Message: msg}

	// ── Hisilicon ─────────────────────────────────────────
	svcEnabled = cfg.Hisilicon.Enabled
	svcPort = cfg.Hisilicon.Port
	if svcPort == 0 {
		svcPort = 15002
	}
	status, msg = checkServiceStatus(svcEnabled, "127.0.0.1", svcPort, checkTimeout)
	response.Services["hisilicon"] = ServiceStatusDetail{Status: status, Port: svcPort, Message: msg}

	// ── TVT ───────────────────────────────────────────────
	svcEnabled = cfg.TVT.Enabled
	svcPort = cfg.TVT.Port
	if svcPort == 0 {
		svcPort = 15003
	}
	status, msg = checkServiceStatus(svcEnabled, "127.0.0.1", svcPort, checkTimeout)
	response.Services["tvt"] = ServiceStatusDetail{Status: status, Port: svcPort, Message: msg}

	// ── GB28181 / SIP ─────────────────────────────────────
	svcEnabled = cfg.GB28181.Enabled
	svcPort = cfg.GB28181.Port
	if svcPort == 0 {
		svcPort = 5060
	}
	gbHost := cfg.GB28181.Host
	if gbHost == "" || gbHost == "0.0.0.0" {
		gbHost = "127.0.0.1"
	}
	status, msg = checkServiceStatus(svcEnabled, gbHost, svcPort, checkTimeout)
	response.Services["gb28181"] = ServiceStatusDetail{Status: status, Port: svcPort, Message: msg}

	// ── P2P Gateway (HTTP health check) ───────────────────
	// Читаем URL: 1) из Go config (env/файл), 2) из БД (фронтенд)
	p2pURL := s.p2pGatewayURL
	if p2pURL == "" {
		p2pURL = cfg.P2PGatewayURL
	}
	if db := getDBObj("services_p2p_gateway"); db != nil {
		if u, ok := getString(db, "url"); ok && u != "" {
			p2pURL = u
		}
	}

	p2pStatus := "disabled"
	p2pMsg := ""
	if p2pURL != "" {
		client := &http.Client{Timeout: 3 * time.Second}
		healthURL := p2pURL + "/health"
		resp, err := client.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				p2pStatus = "running"
			} else {
				p2pStatus = "error"
				p2pMsg = fmt.Sprintf("HTTP %d from %s", resp.StatusCode, healthURL)
			}
		} else {
			p2pStatus = "stopped"
			p2pMsg = fmt.Sprintf("Cannot reach %s: %v", healthURL, err)
		}
	}
	response.Services["p2p_gateway"] = ServiceStatusDetail{Status: p2pStatus, Port: 0, Message: p2pMsg}

	// ── CMMS Adapter (CMMS-3.3.2) ────────────────────────────
	cmmsAdapterName := s.config.CMMSAdapter
	if cmmsAdapterName == "" {
		cmmsAdapterName = "internal"
	}
	adapterStatus := "running"
	adapterMsg := fmt.Sprintf("%s adapter", cmmsAdapterName)

	// Для внешних адаптеров пробуем HealthCheck
	if cmmsAdapterName != "internal" {
		type healthChecker interface{ HealthCheck(context.Context) error }
		if hc, ok := s.cmmsRouter.Adapter().(healthChecker); ok {
			hcCtx, hcCancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer hcCancel()
			if err := hc.HealthCheck(hcCtx); err != nil {
				adapterStatus = "error"
				adapterMsg = fmt.Sprintf("%s: %v", cmmsAdapterName, err)
			}
		}
	}
	response.Services["cmms_adapter"] = ServiceStatusDetail{
		Status:  adapterStatus,
		Port:    0,
		Message: adapterMsg,
	}

	jsonResponse(w, http.StatusOK, response)
}

// checkServiceStatus проверяет, запущен ли сервис, по TCP (или UDP если TCP не отвечает).
// SIP/GB28181 и Syslog используют UDP, поэтому пробуем оба протокола.
func checkServiceStatus(enabled bool, host string, port int, timeout time.Duration) (string, string) {
	if !enabled {
		return "disabled", ""
	}
	target := net.JoinHostPort(host, strconv.Itoa(port))

	// Сначала пробуем TCP
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err == nil {
		conn.Close()
		return "running", ""
	}

	// Если TCP не отвечает, пробуем UDP (SIP, Syslog)
	udpConn, udpErr := net.DialTimeout("udp", target, timeout)
	if udpErr == nil {
		udpConn.Close()
		return "running", ""
	}

	return "stopped", fmt.Sprintf("Port %d not listening (TCP/UDP)", port)
}
