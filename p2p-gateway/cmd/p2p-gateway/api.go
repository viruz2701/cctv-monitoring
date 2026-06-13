// api.go (исправленная версия, удалено неиспользуемое объявление dev)

package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type RegisterRequest struct {
	Brand        string `json:"brand"`
	Serial       string `json:"serial"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	SecurityCode string `json:"security_code,omitempty"`
}

type CommandRequest struct {
	Command string            `json:"command"`
	Params  map[string]string `json:"params"`
}

func NewRouter(dm *DeviceManager, cfg *Config) *chi.Mux {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Аутентификация через API-ключ
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-API-Key") != cfg.BackendAPIKey {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Эндпоинты
	r.Post("/p2p/register", func(w http.ResponseWriter, r *http.Request) {
		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		dev := &Device{
			ID:           fmt.Sprintf("p2p_%s_%s", req.Brand, req.Serial),
			Brand:        req.Brand,
			Serial:       req.Serial,
			Username:     req.Username,
			Password:     req.Password,
			SecurityCode: req.SecurityCode,
			Status:       StatusUnknown,
		}
		if err := dm.AddDevice(dev); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(dev)
	})

	r.Get("/p2p/devices", func(w http.ResponseWriter, r *http.Request) {
		devs := dm.GetAllDevices()
		json.NewEncoder(w).Encode(devs)
	})

	r.Get("/p2p/status/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		dev, ok := dm.GetDevice(id)
		if !ok {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    dev.Status,
			"rtsp_url":  dev.RTSPURL,
			"last_seen": dev.LastSeen,
		})
	})

	r.Get("/p2p/snapshot/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		if _, ok := dm.GetDevice(id); !ok {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		// Заглушка: возвращаем фиктивное изображение (серый квадрат)
		// В реальности адаптер должен запросить snapshot у устройства
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte{}) // здесь должен быть JPEG
	})

	r.Post("/p2p/command/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req CommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		dev, ok := dm.GetDevice(id)
		if !ok {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		adapter, ok := dm.adapters[dev.Brand]
		if !ok {
			http.Error(w, "unsupported brand", http.StatusBadRequest)
			return
		}
		if err := adapter.Command(dev, req.Command, req.Params); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	return r
}
