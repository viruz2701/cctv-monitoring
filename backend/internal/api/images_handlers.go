package api

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
)

// ---------- Изображения ----------

func (s *Server) getImage(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if strings.Contains(filename, "..") || strings.ContainsAny(filename, "/\\") {
		RespondError(w, r, NewBadRequestError("invalid filename"))
		return
	}
	filePath := filepath.Join(s.imagesDir, filename)
	http.ServeFile(w, r, filePath)
}

func (s *Server) listDeviceImages(w http.ResponseWriter, r *http.Request) {
	deviceId := chi.URLParam(r, "deviceId")
	claims := auth.GetClaims(r)
	if claims.Role == "owner" {
		dev, ok := s.stateManager.Get(deviceId)
		if !ok || dev.OwnerID == nil || *dev.OwnerID != claims.UserID {
			RespondError(w, r, NewForbiddenError("forbidden"))
			return
		}
	}
	pattern := filepath.Join(s.imagesDir, safeDeviceID(deviceId)+"_*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}
	baseNames := make([]string, len(files))
	for i, f := range files {
		baseNames[i] = filepath.Base(f)
	}
	jsonResponse(w, http.StatusOK, baseNames)
}
