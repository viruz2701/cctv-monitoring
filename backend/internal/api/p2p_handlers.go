package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
)

// ---------- P2P Management ----------

func (s *Server) listP2PDevices(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "manager" {
		respondError(w, r, NewForbiddenError("forbidden"))
		return
	}
	resp, err := s.proxyP2PGateway("GET", "/api/v1/devices", nil)
	if err != nil {
		s.logger.Error("p2p list devices failed", "error", err)
		respondError(w, r, NewExternalServiceError("p2p gateway error"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (s *Server) registerP2PDevice(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "manager" {
		respondError(w, r, NewForbiddenError("forbidden"))
		return
	}
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	resp, err := s.proxyP2PGateway("POST", "/api/v1/devices", body)
	if err != nil {
		s.logger.Error("p2p register device failed", "error", err)
		respondError(w, r, NewExternalServiceError("p2p gateway error"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (s *Server) getP2PDeviceStatus(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "manager" {
		respondError(w, r, NewForbiddenError("forbidden"))
		return
	}
	id := chi.URLParam(r, "id")
	resp, err := s.proxyP2PGateway("GET", "/api/v1/devices/"+id+"/status", nil)
	if err != nil {
		s.logger.Error("p2p get status failed", "error", err)
		respondError(w, r, NewExternalServiceError("p2p gateway error"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (s *Server) sendP2PCommand(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "manager" {
		respondError(w, r, NewForbiddenError("forbidden"))
		return
	}
	id := chi.URLParam(r, "id")
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	resp, err := s.proxyP2PGateway("POST", "/api/v1/devices/"+id+"/command", body)
	if err != nil {
		s.logger.Error("p2p send command failed", "error", err)
		respondError(w, r, NewExternalServiceError("p2p gateway error"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (s *Server) getP2PSnapshot(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "manager" {
		respondError(w, r, NewForbiddenError("forbidden"))
		return
	}
	id := chi.URLParam(r, "id")
	resp, err := s.proxyP2PGateway("GET", "/api/v1/devices/"+id+"/snapshot", nil)
	if err != nil {
		s.logger.Error("p2p get snapshot failed", "error", err)
		respondError(w, r, NewExternalServiceError("p2p gateway error"))
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(resp)
}

// proxyP2PGateway проксирует запросы к P2P Gateway.
func (s *Server) proxyP2PGateway(method, path string, body []byte) ([]byte, error) {
	url := strings.TrimRight(s.p2pGatewayURL, "/") + path
	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-API-Key", s.p2pAPIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
