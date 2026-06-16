package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"p2p-gateway/pkg/adapters"
)

func NewRouter(dm *DeviceManager) *chi.Mux {
	r := chi.NewRouter()

	// Регистрация устройства
	r.Post("/p2p/register", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Brand        string `json:"brand"`
			Serial       string `json:"serial"`
			Username     string `json:"username"`
			Password     string `json:"password"`
			SecurityCode string `json:"security_code"`
			IPAddress    string `json:"ip_address"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		dev, err := dm.AddDevice(req.Brand, req.Serial, req.Username, req.Password, req.SecurityCode, req.IPAddress)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         dev.ID,
			"rtsp_url":   dev.RTSPURL,
			"status":     dev.Status,
			"proxy_port": dev.ProxyPort,
		})
	})

	// Получить статус устройства
	r.Get("/p2p/status/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		dev, ok := dm.GetDevice(id)
		if !ok {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		adapter, ok := dm.GetAdapter(dev.Brand)
		if !ok {
			http.Error(w, "adapter not found", http.StatusNotFound)
			return
		}
		status, err := adapter.GetStatus(dev)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"device_id": dev.ID,
			"status":    status,
			"rtsp_url":  dev.RTSPURL,
		})
	})

	// Получить RTSP-поток
	r.Get("/p2p/stream/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		dev, ok := dm.GetDevice(id)
		if !ok {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"device_id": dev.ID,
			"rtsp_url":  dev.RTSPURL,
		})
	})

	// PTZ-команда (только для Xiongmai)
	r.Post("/p2p/command/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req struct {
			Command string `json:"command"`
			Speed   int    `json:"speed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		dev, ok := dm.GetDevice(id)
		if !ok {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		adapter, ok := dm.GetAdapter(dev.Brand)
		if !ok {
			http.Error(w, "adapter not found", http.StatusNotFound)
			return
		}
		xAdapter, ok := adapter.(*adapters.XiongmaiAdapter)
		if !ok {
			http.Error(w, "PTZ not supported for this device", http.StatusBadRequest)
			return
		}
		if req.Speed == 0 {
			req.Speed = 5
		}
		if err := xAdapter.SendPTZCommand(dev, req.Command, req.Speed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Снимок (только для Xiongmai)
	r.Get("/p2p/snapshot/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		dev, ok := dm.GetDevice(id)
		if !ok {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		adapter, ok := dm.GetAdapter(dev.Brand)
		if !ok {
			http.Error(w, "adapter not found", http.StatusNotFound)
			return
		}
		xAdapter, ok := adapter.(*adapters.XiongmaiAdapter)
		if !ok {
			http.Error(w, "snapshot not supported for this device", http.StatusBadRequest)
			return
		}
		imageData, err := xAdapter.GetSnapshot(dev)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(imageData)
	})

	// Логи (только для Xiongmai)
	r.Get("/p2p/logs/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		startTime := r.URL.Query().Get("start_time")
		endTime := r.URL.Query().Get("end_time")
		if startTime == "" || endTime == "" {
			now := time.Now()
			endTime = now.Format("2006-01-02 15:04:05")
			startTime = now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
		}
		dev, ok := dm.GetDevice(id)
		if !ok {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		adapter, ok := dm.GetAdapter(dev.Brand)
		if !ok {
			http.Error(w, "adapter not found", http.StatusNotFound)
			return
		}
		xAdapter, ok := adapter.(*adapters.XiongmaiAdapter)
		if !ok {
			http.Error(w, "logs not supported for this device", http.StatusBadRequest)
			return
		}
		logs, err := xAdapter.GetLogs(dev, startTime, endTime)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"device_id":  id,
			"start_time": startTime,
			"end_time":   endTime,
			"logs":       logs,
			"count":      len(logs),
		})
	})

	return r
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
