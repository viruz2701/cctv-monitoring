// ═══════════════════════════════════════════════════════════════════════════
// Package api — Self-Service VPN Handlers (SELFSERV-02)
//
// Self-Service endpoint для скачивания WireGuard конфигурации инженером.
// В отличие от admin handleGetVPNSessionConfig, этот endpoint:
//   - Проверяет, что инженер — владелец сессии
//   - Возвращает wg-quick .conf файл (Content-Disposition: attachment)
//   - Использует WGConfigGenerator для форматирования
//
// Endpoint (заменяет старый GET /api/v1/vpn/sessions/{id}/config):
//   GET /api/v1/vpn/sessions/{id}/config — скачать WG конфиг (self-service)
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - IEC 62443-3-3 SR 2.1: Authorisation enforcement (ownership check)
//   - Приказ ОАЦ №66 п. 7.18.2: Управление удалённым доступом
//   - ISO 27001 A.12.4: Audit trail
//   - OWASP ASVS L3 V3.3: Privilege escalation prevention
// ═══════════════════════════════════════════════════════════════════════════

package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"gb-telemetry-collector/internal/edge"
)

// handleSelfServiceGetVPNSessionConfig возвращает WireGuard .conf файл
// для self-service скачивания инженером.
//
// Отличия от admin handleGetVPNSessionConfig:
// 1. Проверка ownership (инженер должен быть владельцем сессии)
// 2. Возвращает wg-quick .conf (Content-Disposition: attachment)
// 3. Использует WGConfigGenerator для форматирования
//
// Compliance:
//   - OWASP ASVS V3.3: Проверка прав доступа к сессии
//   - ISO 27001 A.9.2: Ownership verification
//   - IEC 62443-3-3 SR 2.1: Authorisation enforcement
func (s *Server) handleSelfServiceGetVPNSessionConfig(w http.ResponseWriter, r *http.Request) {
	if s.vpnSessionManager == nil {
		RespondError(w, r, NewBadRequestError("VPN sessions are not configured"))
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, NewValidationError("invalid session id"))
		return
	}

	// Получаем сессию через менеджер
	session, err := s.vpnSessionManager.GetSession(r.Context(), sessionID)
	if err != nil {
		RespondError(w, r, NewNotFoundError("session not found"))
		return
	}

	// SELFSERV-02: Проверка ownership — инженер должен быть владельцем сессии
	// OWASP ASVS V3.3: Privilege escalation prevention
	userIDStr := getUserIDFromContext(r.Context())
	if userIDStr == "" || userIDStr == "system" {
		RespondError(w, r, NewUnauthorizedError("user not authenticated"))
		return
	}
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		RespondError(w, r, NewUnauthorizedError("invalid user identity"))
		return
	}
	if session.EngineerID != userUUID {
		RespondError(w, r, NewForbiddenError("you are not the owner of this session"))
		return
	}

	// Проверка статуса сессии
	if session.Status != "active" {
		RespondError(w, r, NewBadRequestError(
			fmt.Sprintf("session is not active (status: %s)", session.Status),
		))
		return
	}

	// SELFSERV-02: Генерируем конфиг через WGConfigGenerator
	generator := edge.NewWGConfigGenerator()

	// Берём настройки из конфига WireGuard сервера
	serverPubKey := s.vpnSessionManager.GetServerPublicKey()
	serverEndpoint := s.vpnSessionManager.GetServerEndpoint()

	// Определяем адрес клиента (из allowed_ips, первый /32)
	clientAddress := getClientAddressFromSession(session)
	if clientAddress == "" {
		RespondError(w, r, NewInternalError("failed to determine client address", nil))
		return
	}

	// DNS сервера из конфига
	dns := s.vpnSessionManager.GetDNS()
	if dns == nil {
		dns = []string{}
	}

	configFile, err := generator.GenerateConfigFile(
		session,
		serverPubKey,
		serverEndpoint,
		clientAddress,
		dns,
	)
	if err != nil {
		RespondError(w, r, NewInternalError("failed to generate wireguard config", err))
		return
	}

	// Подписываем конфиг HMAC (если ключ настроен)
	hmacKey := s.vpnSessionManager.GetHMACKey()
	if hmacKey != nil {
		configFile = generator.SignConfig(configFile, hmacKey)
	}

	// SELFSERV-02: Возвращаем как .conf файл (Content-Disposition: attachment)
	filename := fmt.Sprintf("cctv-vpn-%s.conf", session.ID.String()[:8])
	w.Header().Set("Content-Type", "application/x-wireguard-config")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(configFile))
}

// getClientAddressFromSession извлекает первый /32 адрес из allowed_ips сессии.
// Если подходящего адреса нет, возвращает первый доступный адрес.
func getClientAddressFromSession(session *edge.VPNSession) string {
	for _, ipNet := range session.AllowedIPs {
		ones, bits := ipNet.Mask.Size()
		// Ищем адрес с маской /32 (IPv4) или /128 (IPv6)
		if ones == bits {
			return ipNet.IP.String() + "/" + fmt.Sprintf("%d", ones)
		}
	}
	// Если нет /32, берём первый адрес из allowed_ips
	if len(session.AllowedIPs) > 0 {
		ipNet := session.AllowedIPs[0]
		ip := ipNet.IP.Mask(ipNet.Mask)
		// Добавляем 1 к последнему октету для адреса клиента
		clientIP := make([]byte, len(ip))
		copy(clientIP, ip)
		if len(clientIP) == 4 {
			clientIP[3]++ // первый адрес в подсети
		}
		ones, _ := ipNet.Mask.Size()
		return fmt.Sprintf("%d.%d.%d.%d/%d", clientIP[0], clientIP[1], clientIP[2], clientIP[3], ones)
	}
	return ""
}
