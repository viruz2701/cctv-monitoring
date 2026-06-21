package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/sip"
)

// ---------- Устройства ----------

func (s *Server) listDevices(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	devicesMap := s.stateManager.GetAll()
	allDevices := make([]*models.Device, 0, len(devicesMap))
	for _, dev := range devicesMap {
		allDevices = append(allDevices, dev)
	}

	var filtered []*models.Device
	switch claims.Role {
	case "admin", "support":
		filtered = allDevices
	case "owner":
		for _, dev := range allDevices {
			if dev.OwnerID != nil && *dev.OwnerID == claims.UserID {
				filtered = append(filtered, dev)
			}
		}
	default:
		filtered = []*models.Device{}
	}

	resp := make([]map[string]interface{}, len(filtered))
	for i, dev := range filtered {
		resp[i] = map[string]interface{}{
			"device_id":     dev.DeviceID,
			"owner_id":      dev.OwnerID,
			"name":          dev.Name,
			"location":      dev.Location,
			"vendor_type":   dev.VendorType,
			"status":        dev.Status,
			"last_seen":     dev.LastSeen,
			"registered_at": dev.RegisteredAt,
			"user_agent":    dev.UserAgent,
			// P2P fields if present
			"p2p_brand":    dev.P2PBrand,
			"p2p_serial":   dev.P2PSerial,
			"cloud_status": dev.CloudStatus,
		}
	}
	jsonResponse(w, http.StatusOK, resp)
}

func (s *Server) getDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dev, ok := s.stateManager.Get(id)
	if !ok {
		respondError(w, r, NewNotFoundError("device not found"))
		return
	}
	claims := auth.GetClaims(r)
	if claims.Role == "owner" {
		if dev.OwnerID == nil || *dev.OwnerID != claims.UserID {
			respondError(w, r, NewForbiddenError("forbidden"))
			return
		}
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"device_id":     dev.DeviceID,
		"owner_id":      dev.OwnerID,
		"name":          dev.Name,
		"location":      dev.Location,
		"vendor_type":   dev.VendorType,
		"status":        dev.Status,
		"last_seen":     dev.LastSeen,
		"registered_at": dev.RegisteredAt,
		"user_agent":    dev.UserAgent,
		"p2p_brand":     dev.P2PBrand,
		"p2p_serial":    dev.P2PSerial,
		"cloud_status":  dev.CloudStatus,
	})
}

func (s *Server) getDeviceStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dev, ok := s.stateManager.Get(id)
	if !ok {
		respondError(w, r, NewNotFoundError("device not found"))
		return
	}
	claims := auth.GetClaims(r)
	if claims.Role == "owner" {
		if dev.OwnerID == nil || *dev.OwnerID != claims.UserID {
			respondError(w, r, NewForbiddenError("forbidden"))
			return
		}
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"device_id": dev.DeviceID,
		"status":    dev.Status,
		"last_seen": dev.LastSeen.Format(time.RFC3339),
	})
}

// ---------- GB28181 ----------

func (s *Server) requestCatalog(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")

	if err := s.sipHandler.RequestCatalog(deviceID); err != nil {
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":    "catalog_requested",
		"device_id": deviceID,
	})
}

func (s *Server) sendPTZCommand(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")

	var req struct {
		Command string `json:"command"`
		Speed   int    `json:"speed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	if req.Speed == 0 {
		req.Speed = 128
	}

	cmd := sip.PTZCommand{
		Action: req.Command,
		Speed:  req.Speed,
	}

	if err := s.sipHandler.SendPTZCommand(deviceID, cmd); err != nil {
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":    "command_sent",
		"device_id": deviceID,
		"command":   req.Command,
	})
}
