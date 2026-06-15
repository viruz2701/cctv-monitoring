package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(dm *DeviceManager) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/p2p/register", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Brand        string `json:"brand"`
			Serial       string `json:"serial"`
			Username     string `json:"username"`
			Password     string `json:"password"`
			SecurityCode string `json:"security_code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		dev, err := dm.AddDevice(req.Brand, req.Serial, req.Username, req.Password, req.SecurityCode)
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
	return r
}
