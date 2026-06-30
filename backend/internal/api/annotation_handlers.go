// Package api — P1-PHOTO: Work Order Photo Annotation Handlers.
//
// Предоставляет CRUD endpoints для управления элементами аннотации
// на фото в рамках work order.
//
// Endpoints:
//
//	POST /api/v1/work-orders/{id}/photos/{photoId}/annotations
//	GET  /api/v1/work-orders/{id}/photos/{photoId}/annotations
//	PUT  /api/v1/work-orders/{id}/photos/{photoId}/annotations
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — annotation JSON schema)
//   - ISO 27001 A.12.4 (Audit trail — logAudit для каждой мутации)
//   - IEC 62443-3-3 SL-3 (Zone 3 — Application security)
//   - СТБ 34.101.27 п. 6.2 (Контроль целостности данных)
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════
// AnnotationStore — интерфейс для хранения аннотаций.
// Позволяет подменять реализацию для тестирования.
// ═══════════════════════════════════════════════════════════════════════

// AnnotationStore defines storage operations for work order annotations.
type AnnotationStore interface {
	GetAnnotation(ctx context.Context, workOrderID, photoURL string) (*models.WorkOrderAnnotation, error)
	UpsertAnnotation(ctx context.Context, annotation *models.WorkOrderAnnotation) error
}

// ═══════════════════════════════════════════════════════════════════════
// PostgreSQL implementation
// ═══════════════════════════════════════════════════════════════════════

// pgAnnotationStore implements AnnotationStore using PostgreSQL.
type pgAnnotationStore struct {
	database *db.DB
}

func newPGAnnotationStore(database *db.DB) *pgAnnotationStore {
	return &pgAnnotationStore{database: database}
}

func (s *pgAnnotationStore) GetAnnotation(ctx context.Context, workOrderID, photoURL string) (*models.WorkOrderAnnotation, error) {
	query := `
		SELECT id, work_order_id, photo_url, elements, created_by, created_at, updated_at
		FROM work_order_annotations
		WHERE work_order_id = $1 AND photo_url = $2
	`

	var ann models.WorkOrderAnnotation
	var elementsJSON []byte

	err := s.database.Pool.QueryRow(ctx, query, workOrderID, photoURL).Scan(
		&ann.ID, &ann.WorkOrderID, &ann.PhotoURL,
		&elementsJSON, &ann.CreatedBy, &ann.CreatedAt, &ann.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(elementsJSON, &ann.Elements); err != nil {
		return nil, err
	}

	return &ann, nil
}

func (s *pgAnnotationStore) UpsertAnnotation(ctx context.Context, ann *models.WorkOrderAnnotation) error {
	elementsJSON, err := json.Marshal(ann.Elements)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO work_order_annotations (work_order_id, photo_url, elements, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (work_order_id, photo_url) DO UPDATE SET
			elements = EXCLUDED.elements,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now().UTC()
	if ann.CreatedAt.IsZero() {
		ann.CreatedAt = now
	}

	_, err = s.database.Pool.Exec(ctx, query,
		ann.WorkOrderID, ann.PhotoURL, elementsJSON,
		ann.CreatedBy, ann.CreatedAt, now,
	)
	return err
}

// ═══════════════════════════════════════════════════════════════════════
// Handlers
// ═══════════════════════════════════════════════════════════════════════

// getAnnotationStore возвращает AnnotationStore (создаёт при первом вызове).
func (s *Server) getAnnotationStore() AnnotationStore {
	if s.annotationStore == nil {
		s.annotationStore = newPGAnnotationStore(s.db)
	}
	return s.annotationStore
}

// handleGetAnnotations — GET /api/v1/work-orders/{id}/photos/{photoId}/annotations
//
// Возвращает существующую аннотацию для указанного фото.
// Если аннотации нет — возвращает пустой массив elements.
func (s *Server) handleGetAnnotations(w http.ResponseWriter, r *http.Request) {
	workOrderID := chi.URLParam(r, "id")
	photoID := chi.URLParam(r, "photoId")

	if workOrderID == "" || photoID == "" {
		RespondError(w, r, NewBadRequestError("work_order_id and photo_id are required"))
		return
	}

	photoURL, err := url.QueryUnescape(photoID)
	if err != nil {
		RespondError(w, r, NewBadRequestError("invalid photo_id encoding"))
		return
	}

	store := s.getAnnotationStore()
	ann, err := store.GetAnnotation(r.Context(), workOrderID, photoURL)
	if err != nil {
		s.logger.Error("failed to get annotation", "error", err, "work_order_id", workOrderID)
		RespondError(w, r, NewInternalError("operation failed", err))
		return
	}

	if ann == nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"work_order_id": workOrderID,
			"photo_url":     photoURL,
			"elements":      []models.AnnotationElement{},
		})
		return
	}

	jsonResponse(w, http.StatusOK, ann)
}

// handleSaveAnnotations — POST /api/v1/work-orders/{id}/photos/{photoId}/annotations
//
// Создаёт новую аннотацию для указанного фото.
// Если аннотация уже существует — возвращает 409 Conflict.
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — Validate() на модели)
//   - ISO 27001 A.12.4 (Audit trail — logAudit)
func (s *Server) handleSaveAnnotations(w http.ResponseWriter, r *http.Request) {
	workOrderID := chi.URLParam(r, "id")
	photoID := chi.URLParam(r, "photoId")

	if workOrderID == "" || photoID == "" {
		RespondError(w, r, NewBadRequestError("work_order_id and photo_id are required"))
		return
	}

	photoURL, err := url.QueryUnescape(photoID)
	if err != nil {
		RespondError(w, r, NewBadRequestError("invalid photo_id encoding"))
		return
	}

	var req models.AnnotationSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}

	// OWASP ASVS V5.1: Input validation
	if errs := req.Validate(); len(errs) > 0 {
		RespondError(w, r, NewBadRequestError("validation failed: "+strings.Join(errs, "; ")))
		return
	}

	store := s.getAnnotationStore()

	existing, err := store.GetAnnotation(r.Context(), workOrderID, photoURL)
	if err != nil {
		s.logger.Error("failed to check existing annotation", "error", err)
		RespondError(w, r, NewInternalError("operation failed", err))
		return
	}
	if existing != nil {
		RespondError(w, r, NewConflictError("annotation already exists for this photo, use PUT to update"))
		return
	}

	userID := getUserIDFromContext(r.Context())

	ann := &models.WorkOrderAnnotation{
		WorkOrderID: workOrderID,
		PhotoURL:    photoURL,
		Elements:    req.Elements,
		CreatedBy:   userID,
		CreatedAt:   time.Now().UTC(),
	}

	if err := store.UpsertAnnotation(r.Context(), ann); err != nil {
		s.logger.Error("failed to save annotation", "error", err)
		RespondError(w, r, NewInternalError("operation failed", err))
		return
	}

	// ISO 27001 A.12.4: Audit trail
	s.logAudit(userID, "create_annotation", "work_order_annotation", workOrderID,
		nil, map[string]interface{}{
			"photo_url": photoURL,
			"elements":  len(req.Elements),
		})

	jsonResponse(w, http.StatusCreated, ann)
}

// handleUpdateAnnotations — PUT /api/v1/work-orders/{id}/photos/{photoId}/annotations
//
// Обновляет существующую аннотацию для указанного фото.
// Если аннотации нет — создаёт новую (upsert).
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — Validate() на модели)
//   - ISO 27001 A.12.4 (Audit trail — logAudit с diff)
func (s *Server) handleUpdateAnnotations(w http.ResponseWriter, r *http.Request) {
	workOrderID := chi.URLParam(r, "id")
	photoID := chi.URLParam(r, "photoId")

	if workOrderID == "" || photoID == "" {
		RespondError(w, r, NewBadRequestError("work_order_id and photo_id are required"))
		return
	}

	photoURL, err := url.QueryUnescape(photoID)
	if err != nil {
		RespondError(w, r, NewBadRequestError("invalid photo_id encoding"))
		return
	}

	var req models.AnnotationSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}

	// OWASP ASVS V5.1: Input validation
	if errs := req.Validate(); len(errs) > 0 {
		RespondError(w, r, NewBadRequestError("validation failed: "+strings.Join(errs, "; ")))
		return
	}

	store := s.getAnnotationStore()
	userID := getUserIDFromContext(r.Context())

	existing, _ := store.GetAnnotation(r.Context(), workOrderID, photoURL)

	ann := &models.WorkOrderAnnotation{
		WorkOrderID: workOrderID,
		PhotoURL:    photoURL,
		Elements:    req.Elements,
		CreatedBy:   userID,
	}

	if existing != nil {
		ann.CreatedAt = existing.CreatedAt
		ann.ID = existing.ID
	}

	if err := store.UpsertAnnotation(r.Context(), ann); err != nil {
		s.logger.Error("failed to update annotation", "error", err)
		RespondError(w, r, NewInternalError("operation failed", err))
		return
	}

	// ISO 27001 A.12.4: Audit trail
	action := "update_annotation"
	oldCount := 0
	if existing != nil {
		oldCount = len(existing.Elements)
	} else {
		action = "create_annotation"
	}
	s.logAudit(userID, action, "work_order_annotation", workOrderID,
		map[string]interface{}{"elements_count": oldCount},
		map[string]interface{}{
			"photo_url":      photoURL,
			"elements_count": len(req.Elements),
		},
	)

	jsonResponse(w, http.StatusOK, ann)
}
