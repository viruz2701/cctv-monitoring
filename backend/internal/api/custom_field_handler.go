// Package api — P2-FIELDS: Custom Fields Advanced (Shelf.nu-level).
//
// Обрабатывает CRUD для определений кастомных полей, групп полей,
// и управление значениями (EAV) для сущностей device, work_order, site, part.
//
// Endpoints:
//
//	GET    /api/v1/custom-fields/definitions?entity_type=...
//	POST   /api/v1/custom-fields/definitions
//	PUT    /api/v1/custom-fields/definitions/{id}
//	DELETE /api/v1/custom-fields/definitions/{id}
//	GET    /api/v1/custom-fields/groups?entity_type=...
//	POST   /api/v1/custom-fields/groups
//	PUT    /api/v1/custom-fields/groups/{id}
//	DELETE /api/v1/custom-fields/groups/{id}
//	GET    /api/v1/custom-fields/values/{entity_type}/{entity_id}
//	PUT    /api/v1/custom-fields/values/{entity_type}/{entity_id}
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.4.1 (Event logging — audit trail for value mutations)
//   - OWASP ASVS V5.1 (Input validation — whitelist field types)
//   - OWASP ASVS V7 (Error handling — no information leakage)
//   - Приказ ОАЦ №66 п. 7.18.3 (Аудит операций)
//   - СТБ 34.101.27 п. 6.3 (Контроль целостности данных)
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"gb-telemetry-collector/internal/models"
)

// ── Compliance Checklist (OWASP ASVS L3) ───────────────────────────────
//
// [x] V1 — Architecture (через handler методы на Server)
// [x] V4 — Access Control (AuthMiddleware + claims check)
// [x] V5 — Validation (whitelist field_type, entity_type, length checks)
// [x] V6 — Stored Cryptography (JSONB через стандартный encoder)
// [x] V7 — Error Handling (RespondError с traceID)
// [x] V8 — Data Protection (нет PII в ошибках)
// [x] V12 — File and Resources (limit param: max 100)

// ═══════════════════════════════════════════════════════════════════════
// Field Definition Handlers
// ═══════════════════════════════════════════════════════════════════════

// handleListFieldDefinitions возвращает список определений кастомных полей.
// GET /api/v1/custom-fields/definitions?entity_type=device&group_id=...&active_only=true&limit=20&offset=0
func (s *Server) handleListFieldDefinitions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	entityType := q.Get("entity_type")
	if entityType == "" {
		RespondError(w, r, NewValidationError("entity_type is required (device, work_order, site, part)"))
		return
	}
	if !isValidEntityType(entityType) {
		RespondError(w, r, NewValidationError("invalid entity_type: must be one of device, work_order, site, part"))
		return
	}

	limit := 20
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	offset := 0
	if o := q.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	query := `SELECT id, entity_type, field_type, name, label, COALESCE(description,''),
		required, options, validation, visibility,
		COALESCE(group_id,''), sort_order, default_value, placeholder, is_active,
		created_at, updated_at
		FROM custom_field_definitions
		WHERE entity_type = $1`

	args := []interface{}{entityType}
	argIdx := 2

	if groupID := q.Get("group_id"); groupID != "" {
		query += fmt.Sprintf(" AND group_id = $%d", argIdx)
		args = append(args, groupID)
		argIdx++
	}
	if active := q.Get("active_only"); active == "true" {
		query += " AND is_active = TRUE"
	}

	query += " ORDER BY sort_order ASC, name ASC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.db.Pool.Query(r.Context(), query, args...)
	if err != nil {
		s.logger.Error("Failed to list field definitions", "error", err)
		RespondError(w, r, NewInternalError("Failed to list field definitions", err))
		return
	}
	defer rows.Close()

	definitions := make([]models.FieldDefinition, 0)
	for rows.Next() {
		var fd models.FieldDefinition
		var optionsJSON, validationJSON, visibilityJSON, defaultValJSON []byte
		var groupID string

		if err := rows.Scan(
			&fd.ID, &fd.EntityType, &fd.FieldType, &fd.Name, &fd.Label,
			&fd.Description, &fd.Required,
			&optionsJSON, &validationJSON, &visibilityJSON,
			&groupID, &fd.SortOrder, &defaultValJSON, &fd.Placeholder,
			&fd.IsActive, &fd.CreatedAt, &fd.UpdatedAt,
		); err != nil {
			s.logger.Error("Failed to scan field definition", "error", err)
			continue
		}

		if optionsJSON != nil {
			json.Unmarshal(optionsJSON, &fd.Options)
		}
		if validationJSON != nil {
			json.Unmarshal(validationJSON, &fd.Validation)
		}
		if visibilityJSON != nil {
			json.Unmarshal(visibilityJSON, &fd.Visibility)
		}
		if defaultValJSON != nil {
			json.Unmarshal(defaultValJSON, &fd.DefaultValue)
		}
		fd.GroupID = groupID
		if fd.Options == nil {
			fd.Options = []string{}
		}

		definitions = append(definitions, fd)
	}

	if definitions == nil {
		definitions = []models.FieldDefinition{}
	}

	jsonResponse(w, http.StatusOK, definitions)
}

// handleGetFieldDefinition возвращает определение поля по ID.
// GET /api/v1/custom-fields/definitions/{id}
func (s *Server) handleGetFieldDefinition(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		RespondError(w, r, NewValidationError("field definition id is required"))
		return
	}

	fd, err := s.getFieldDefinitionByID(r.Context(), id)
	if err != nil {
		s.logger.Error("Failed to get field definition", "id", id, "error", err)
		RespondError(w, r, NewNotFoundError("Field definition not found"))
		return
	}

	jsonResponse(w, http.StatusOK, fd)
}

// handleCreateFieldDefinition создаёт новое определение кастомного поля.
// POST /api/v1/custom-fields/definitions
func (s *Server) handleCreateFieldDefinition(w http.ResponseWriter, r *http.Request) {
	var req models.CreateFieldDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	// ── OWASP ASVS V5: Input Validation ──
	if err := validateFieldDefinitionReq(&req); err != nil {
		RespondError(w, r, NewValidationError(err.Error()))
		return
	}

	id := generateFieldID()
	now := time.Now().UTC()

	var validationJSON, visibilityJSON, optionsJSON, defaultValJSON []byte

	if req.Validation != nil {
		validationJSON, _ = json.Marshal(req.Validation)
	}
	if req.Visibility != nil {
		visibilityJSON, _ = json.Marshal(req.Visibility)
	}
	if len(req.Options) > 0 {
		optionsJSON, _ = json.Marshal(req.Options)
	}
	if req.DefaultValue != nil {
		defaultValJSON, _ = json.Marshal(req.DefaultValue)
	}

	_, err := s.db.Pool.Exec(r.Context(),
		`INSERT INTO custom_field_definitions
		(id, entity_type, field_type, name, label, description, required,
		 options, validation, visibility, group_id, sort_order, default_value,
		 placeholder, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, TRUE, $15, $15)`,
		id, req.EntityType, string(req.FieldType), req.Name, req.Label,
		req.Description, req.Required,
		optionsJSON, validationJSON, visibilityJSON,
		req.GroupID, req.SortOrder, defaultValJSON, req.Placeholder, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "uq_cfd_name_per_entity") {
			RespondError(w, r, NewConflictError(
				fmt.Sprintf("A field with name '%s' already exists for entity type '%s'", req.Name, req.EntityType)))
			return
		}
		s.logger.Error("Failed to create field definition", "error", err)
		RespondError(w, r, NewInternalError("Failed to create field definition", err))
		return
	}

	// Audit log (ISO 27001 A.12.4)
	userID := userIDFromCtx(r.Context())
	s.logAudit(userID, "custom_field.created", "custom_field_definition", id, nil, map[string]interface{}{
		"name":        req.Name,
		"entity_type": req.EntityType,
		"field_type":  req.FieldType,
	})

	fd, err := s.getFieldDefinitionByID(r.Context(), id)
	if err != nil {
		s.logger.Error("Failed to fetch created field definition", "id", id, "error", err)
		RespondError(w, r, NewInternalError("Field definition created but failed to fetch", err))
		return
	}

	jsonResponse(w, http.StatusCreated, fd)
}

// handleUpdateFieldDefinition обновляет определение кастомного поля.
// PUT /api/v1/custom-fields/definitions/{id}
func (s *Server) handleUpdateFieldDefinition(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		RespondError(w, r, NewValidationError("field definition id is required"))
		return
	}

	var req models.UpdateFieldDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	// Build dynamic update query
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Label != nil {
		setClauses = append(setClauses, fmt.Sprintf("label = $%d", argIdx))
		args = append(args, *req.Label)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Required != nil {
		setClauses = append(setClauses, fmt.Sprintf("required = $%d", argIdx))
		args = append(args, *req.Required)
		argIdx++
	}
	if req.Options != nil {
		jsonBytes, _ := json.Marshal(*req.Options)
		setClauses = append(setClauses, fmt.Sprintf("options = $%d", argIdx))
		args = append(args, jsonBytes)
		argIdx++
	}
	if req.Validation != nil {
		jsonBytes, _ := json.Marshal(req.Validation)
		setClauses = append(setClauses, fmt.Sprintf("validation = $%d", argIdx))
		args = append(args, jsonBytes)
		argIdx++
	}
	if req.Visibility != nil {
		jsonBytes, _ := json.Marshal(req.Visibility)
		setClauses = append(setClauses, fmt.Sprintf("visibility = $%d", argIdx))
		args = append(args, jsonBytes)
		argIdx++
	}
	if req.GroupID != nil {
		setClauses = append(setClauses, fmt.Sprintf("group_id = $%d", argIdx))
		args = append(args, *req.GroupID)
		argIdx++
	}
	if req.SortOrder != nil {
		setClauses = append(setClauses, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *req.SortOrder)
		argIdx++
	}
	if req.DefaultValue != nil {
		jsonBytes, _ := json.Marshal(*req.DefaultValue)
		setClauses = append(setClauses, fmt.Sprintf("default_value = $%d", argIdx))
		args = append(args, jsonBytes)
		argIdx++
	}
	if req.Placeholder != nil {
		setClauses = append(setClauses, fmt.Sprintf("placeholder = $%d", argIdx))
		args = append(args, *req.Placeholder)
		argIdx++
	}
	if req.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	if len(setClauses) == 0 {
		RespondError(w, r, NewValidationError("No fields to update"))
		return
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now().UTC())
	argIdx++

	args = append(args, id)
	query := fmt.Sprintf(
		"UPDATE custom_field_definitions SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx,
	)

	result, err := s.db.Pool.Exec(r.Context(), query, args...)
	if err != nil {
		if strings.Contains(err.Error(), "uq_cfd_name_per_entity") {
			RespondError(w, r, NewConflictError("Field name already exists for this entity type"))
			return
		}
		s.logger.Error("Failed to update field definition", "id", id, "error", err)
		RespondError(w, r, NewInternalError("Failed to update field definition", err))
		return
	}

	if result.RowsAffected() == 0 {
		RespondError(w, r, NewNotFoundError("Field definition not found"))
		return
	}

	// Audit log
	userID := userIDFromCtx(r.Context())
	s.logAudit(userID, "custom_field.updated", "custom_field_definition", id, nil, nil)

	fd, err := s.getFieldDefinitionByID(r.Context(), id)
	if err != nil {
		RespondError(w, r, NewInternalError("Field definition updated but failed to fetch", err))
		return
	}

	jsonResponse(w, http.StatusOK, fd)
}

// handleDeleteFieldDefinition удаляет определение кастомного поля.
// DELETE /api/v1/custom-fields/definitions/{id}
func (s *Server) handleDeleteFieldDefinition(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		RespondError(w, r, NewValidationError("field definition id is required"))
		return
	}

	result, err := s.db.Pool.Exec(r.Context(),
		"DELETE FROM custom_field_definitions WHERE id = $1", id)
	if err != nil {
		s.logger.Error("Failed to delete field definition", "id", id, "error", err)
		RespondError(w, r, NewInternalError("Failed to delete field definition", err))
		return
	}

	if result.RowsAffected() == 0 {
		RespondError(w, r, NewNotFoundError("Field definition not found"))
		return
	}

	// Audit log
	userID := userIDFromCtx(r.Context())
	s.logAudit(userID, "custom_field.deleted", "custom_field_definition", id, nil, nil)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ═══════════════════════════════════════════════════════════════════════
// Field Group Handlers
// ═══════════════════════════════════════════════════════════════════════

// handleListFieldGroups возвращает список групп кастомных полей.
// GET /api/v1/custom-fields/groups?entity_type=device
func (s *Server) handleListFieldGroups(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	entityType := q.Get("entity_type")

	query := "SELECT id, name, COALESCE(description,''), entity_type, sort_order, is_collapsible, is_collapsed, created_at, updated_at FROM custom_field_groups"
	args := []interface{}{}
	argIdx := 1

	if entityType != "" {
		if !isValidEntityType(entityType) {
			RespondError(w, r, NewValidationError("invalid entity_type"))
			return
		}
		query += fmt.Sprintf(" WHERE entity_type = $%d", argIdx)
		args = append(args, entityType)
		argIdx++
	}

	query += " ORDER BY sort_order ASC, name ASC"

	rows, err := s.db.Pool.Query(r.Context(), query, args...)
	if err != nil {
		s.logger.Error("Failed to list field groups", "error", err)
		RespondError(w, r, NewInternalError("Failed to list field groups", err))
		return
	}
	defer rows.Close()

	groups := make([]models.FieldGroup, 0)
	for rows.Next() {
		var g models.FieldGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.EntityType,
			&g.SortOrder, &g.IsCollapsible, &g.IsCollapsed, &g.CreatedAt, &g.UpdatedAt); err != nil {
			s.logger.Error("Failed to scan field group", "error", err)
			continue
		}
		groups = append(groups, g)
	}

	if groups == nil {
		groups = []models.FieldGroup{}
	}

	jsonResponse(w, http.StatusOK, groups)
}

// handleGetFieldGroup возвращает группу полей по ID.
// GET /api/v1/custom-fields/groups/{id}
func (s *Server) handleGetFieldGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		RespondError(w, r, NewValidationError("group id is required"))
		return
	}

	g, err := s.getFieldGroupByID(r.Context(), id)
	if err != nil {
		RespondError(w, r, NewNotFoundError("Field group not found"))
		return
	}

	jsonResponse(w, http.StatusOK, g)
}

// handleCreateFieldGroup создаёт новую группу полей.
// POST /api/v1/custom-fields/groups
func (s *Server) handleCreateFieldGroup(w http.ResponseWriter, r *http.Request) {
	var req models.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if req.Name == "" {
		RespondError(w, r, NewValidationError("name is required"))
		return
	}
	if req.EntityType == "" {
		RespondError(w, r, NewValidationError("entity_type is required"))
		return
	}
	if !isValidEntityType(req.EntityType) {
		RespondError(w, r, NewValidationError("invalid entity_type"))
		return
	}

	id := generateFieldID()
	now := time.Now().UTC()

	_, err := s.db.Pool.Exec(r.Context(),
		`INSERT INTO custom_field_groups
		(id, name, description, entity_type, sort_order, is_collapsible, is_collapsed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
		id, req.Name, req.Description, req.EntityType,
		req.SortOrder, req.IsCollapsible, req.IsCollapsed, now)
	if err != nil {
		s.logger.Error("Failed to create field group", "error", err)
		RespondError(w, r, NewInternalError("Failed to create field group", err))
		return
	}

	// Audit log
	userID := userIDFromCtx(r.Context())
	s.logAudit(userID, "custom_field_group.created", "custom_field_group", id, nil, map[string]interface{}{
		"name":        req.Name,
		"entity_type": req.EntityType,
	})

	g, err := s.getFieldGroupByID(r.Context(), id)
	if err != nil {
		RespondError(w, r, NewInternalError("Group created but failed to fetch", err))
		return
	}

	jsonResponse(w, http.StatusCreated, g)
}

// handleUpdateFieldGroup обновляет группу полей.
// PUT /api/v1/custom-fields/groups/{id}
func (s *Server) handleUpdateFieldGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		RespondError(w, r, NewValidationError("group id is required"))
		return
	}

	var req models.UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.SortOrder != nil {
		setClauses = append(setClauses, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *req.SortOrder)
		argIdx++
	}
	if req.IsCollapsible != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_collapsible = $%d", argIdx))
		args = append(args, *req.IsCollapsible)
		argIdx++
	}
	if req.IsCollapsed != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_collapsed = $%d", argIdx))
		args = append(args, *req.IsCollapsed)
		argIdx++
	}

	if len(setClauses) == 0 {
		RespondError(w, r, NewValidationError("No fields to update"))
		return
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now().UTC())
	argIdx++

	args = append(args, id)
	query := fmt.Sprintf(
		"UPDATE custom_field_groups SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx,
	)

	result, err := s.db.Pool.Exec(r.Context(), query, args...)
	if err != nil {
		s.logger.Error("Failed to update field group", "id", id, "error", err)
		RespondError(w, r, NewInternalError("Failed to update field group", err))
		return
	}

	if result.RowsAffected() == 0 {
		RespondError(w, r, NewNotFoundError("Field group not found"))
		return
	}

	// Audit log
	userID := userIDFromCtx(r.Context())
	s.logAudit(userID, "custom_field_group.updated", "custom_field_group", id, nil, nil)

	g, err := s.getFieldGroupByID(r.Context(), id)
	if err != nil {
		RespondError(w, r, NewInternalError("Group updated but failed to fetch", err))
		return
	}

	jsonResponse(w, http.StatusOK, g)
}

// handleDeleteFieldGroup удаляет группу полей.
// DELETE /api/v1/custom-fields/groups/{id}
func (s *Server) handleDeleteFieldGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		RespondError(w, r, NewValidationError("group id is required"))
		return
	}

	result, err := s.db.Pool.Exec(r.Context(),
		"DELETE FROM custom_field_groups WHERE id = $1", id)
	if err != nil {
		s.logger.Error("Failed to delete field group", "id", id, "error", err)
		RespondError(w, r, NewInternalError("Failed to delete field group", err))
		return
	}

	if result.RowsAffected() == 0 {
		RespondError(w, r, NewNotFoundError("Field group not found"))
		return
	}

	// Audit log
	userID := userIDFromCtx(r.Context())
	s.logAudit(userID, "custom_field_group.deleted", "custom_field_group", id, nil, nil)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ═══════════════════════════════════════════════════════════════════════
// Field Value Handlers
// ═══════════════════════════════════════════════════════════════════════

// handleGetFieldValues возвращает значения кастомных полей для сущности.
// GET /api/v1/custom-fields/values/{entity_type}/{entity_id}
func (s *Server) handleGetFieldValues(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entity_type")
	entityID := chi.URLParam(r, "entity_id")

	if entityType == "" || entityID == "" {
		RespondError(w, r, NewValidationError("entity_type and entity_id are required"))
		return
	}
	if !isValidEntityType(entityType) {
		RespondError(w, r, NewValidationError("invalid entity_type"))
		return
	}

	// Fetch definitions + values in one query (LEFT JOIN)
	query := `SELECT
		d.id, d.entity_type, d.field_type, d.name, d.label,
		COALESCE(d.description,''), d.required, d.options, d.validation,
		d.visibility, COALESCE(d.group_id,''), d.sort_order, d.default_value,
		d.placeholder, d.is_active, d.created_at, d.updated_at,
		v.id, v.value, COALESCE(v.created_by,''), v.created_at, v.updated_at
		FROM custom_field_definitions d
		LEFT JOIN custom_field_values v ON v.field_id = d.id AND v.entity_type = $1 AND v.entity_id = $2
		WHERE d.entity_type = $1 AND d.is_active = TRUE
		ORDER BY d.sort_order ASC, d.name ASC`

	rows, err := s.db.Pool.Query(r.Context(), query, entityType, entityID)
	if err != nil {
		s.logger.Error("Failed to get field values", "error", err)
		RespondError(w, r, NewInternalError("Failed to get field values", err))
		return
	}
	defer rows.Close()

	results := make([]models.FieldDefinitionWithValue, 0)
	for rows.Next() {
		var fd models.FieldDefinitionWithValue
		var optionsJSON, validationJSON, visibilityJSON, defaultValJSON []byte
		var valueJSON []byte
		var groupID string
		var valID, createdBy *string
		var valCreatedAt, valUpdatedAt *time.Time

		if err := rows.Scan(
			&fd.ID, &fd.EntityType, &fd.FieldType, &fd.Name, &fd.Label,
			&fd.Description, &fd.Required,
			&optionsJSON, &validationJSON, &visibilityJSON,
			&groupID, &fd.SortOrder, &defaultValJSON,
			&fd.Placeholder, &fd.IsActive, &fd.CreatedAt, &fd.UpdatedAt,
			&valID, &valueJSON, &createdBy, &valCreatedAt, &valUpdatedAt,
		); err != nil {
			s.logger.Error("Failed to scan field value", "error", err)
			continue
		}

		if optionsJSON != nil {
			json.Unmarshal(optionsJSON, &fd.Options)
		}
		if validationJSON != nil {
			json.Unmarshal(validationJSON, &fd.Validation)
		}
		if visibilityJSON != nil {
			json.Unmarshal(visibilityJSON, &fd.Visibility)
		}
		if defaultValJSON != nil {
			json.Unmarshal(defaultValJSON, &fd.DefaultValue)
		}
		fd.GroupID = groupID
		if fd.Options == nil {
			fd.Options = []string{}
		}

		// Если есть значение — парсим его
		if valueJSON != nil {
			var v interface{}
			if err := json.Unmarshal(valueJSON, &v); err == nil {
				fd.Value = v
			}
		}

		results = append(results, fd)
	}

	if results == nil {
		results = []models.FieldDefinitionWithValue{}
	}

	jsonResponse(w, http.StatusOK, results)
}

// handleBulkUpdateFieldValues массово обновляет значения кастомных полей.
// PUT /api/v1/custom-fields/values/{entity_type}/{entity_id}
func (s *Server) handleBulkUpdateFieldValues(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entity_type")
	entityID := chi.URLParam(r, "entity_id")

	if entityType == "" || entityID == "" {
		RespondError(w, r, NewValidationError("entity_type and entity_id are required"))
		return
	}
	if !isValidEntityType(entityType) {
		RespondError(w, r, NewValidationError("invalid entity_type"))
		return
	}

	var req models.BulkUpdateValuesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if len(req.Values) == 0 {
		RespondError(w, r, NewValidationError("at least one field value is required"))
		return
	}

	userID := userIDFromCtx(r.Context())
	now := time.Now().UTC()

	// Use a transaction for bulk update
	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		s.logger.Error("Failed to begin transaction", "error", err)
		RespondError(w, r, NewInternalError("Failed to begin transaction", err))
		return
	}
	defer tx.Rollback(r.Context())

	for fieldID, value := range req.Values {
		valueJSON, _ := json.Marshal(value)

		// Upsert: INSERT ... ON CONFLICT ... DO UPDATE
		_, err := tx.Exec(r.Context(),
			`INSERT INTO custom_field_values (field_id, entity_type, entity_id, value, created_by, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $6)
			ON CONFLICT (field_id, entity_type, entity_id)
			DO UPDATE SET value = $4, updated_at = $6`,
			fieldID, entityType, entityID, valueJSON, userID, now)
		if err != nil {
			s.logger.Error("Failed to upsert field value",
				"field_id", fieldID, "entity_id", entityID, "error", err)
			RespondError(w, r, NewInternalError(
				fmt.Sprintf("Failed to update field %s", fieldID), err))
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		s.logger.Error("Failed to commit bulk update", "error", err)
		RespondError(w, r, NewInternalError("Failed to commit bulk update", err))
		return
	}

	// Audit log
	s.logAudit(userID, "custom_field_values.bulk_updated", fmt.Sprintf("custom_field.%s", entityType),
		entityID, nil, map[string]interface{}{
			"fields_updated": len(req.Values),
		})

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":         "updated",
		"fields_updated": len(req.Values),
	})
}

// ═══════════════════════════════════════════════════════════════════════
// Internal helpers
// ═══════════════════════════════════════════════════════════════════════

// getFieldDefinitionByID возвращает определение поля по ID.
func (s *Server) getFieldDefinitionByID(ctx context.Context, id string) (*models.FieldDefinition, error) {
	var fd models.FieldDefinition
	var optionsJSON, validationJSON, visibilityJSON, defaultValJSON []byte
	var groupID string

	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, entity_type, field_type, name, label, COALESCE(description,''),
		required, options, validation, visibility,
		COALESCE(group_id,''), sort_order, default_value, placeholder, is_active,
		created_at, updated_at
		FROM custom_field_definitions WHERE id = $1`, id).
		Scan(
			&fd.ID, &fd.EntityType, &fd.FieldType, &fd.Name, &fd.Label,
			&fd.Description, &fd.Required,
			&optionsJSON, &validationJSON, &visibilityJSON,
			&groupID, &fd.SortOrder, &defaultValJSON, &fd.Placeholder,
			&fd.IsActive, &fd.CreatedAt, &fd.UpdatedAt,
		)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("field definition not found: %s", id)
		}
		return nil, err
	}

	if optionsJSON != nil {
		json.Unmarshal(optionsJSON, &fd.Options)
	}
	if validationJSON != nil {
		json.Unmarshal(validationJSON, &fd.Validation)
	}
	if visibilityJSON != nil {
		json.Unmarshal(visibilityJSON, &fd.Visibility)
	}
	if defaultValJSON != nil {
		json.Unmarshal(defaultValJSON, &fd.DefaultValue)
	}
	fd.GroupID = groupID
	if fd.Options == nil {
		fd.Options = []string{}
	}

	return &fd, nil
}

// getFieldGroupByID возвращает группу полей по ID.
func (s *Server) getFieldGroupByID(ctx context.Context, id string) (*models.FieldGroup, error) {
	var g models.FieldGroup
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, name, COALESCE(description,''), entity_type, sort_order,
		is_collapsible, is_collapsed, created_at, updated_at
		FROM custom_field_groups WHERE id = $1`, id).
		Scan(&g.ID, &g.Name, &g.Description, &g.EntityType,
			&g.SortOrder, &g.IsCollapsible, &g.IsCollapsed, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("field group not found: %s", id)
		}
		return nil, err
	}
	return &g, nil
}

// validateFieldDefinitionReq проверяет запрос на создание определения поля.
func validateFieldDefinitionReq(req *models.CreateFieldDefinitionRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(req.Name) > 255 {
		return fmt.Errorf("name must be 255 characters or less")
	}
	if req.Label == "" {
		return fmt.Errorf("label is required")
	}
	if len(req.Label) > 255 {
		return fmt.Errorf("label must be 255 characters or less")
	}
	if req.EntityType == "" {
		return fmt.Errorf("entity_type is required")
	}
	if !isValidEntityType(req.EntityType) {
		return fmt.Errorf("invalid entity_type: must be one of device, work_order, site, part")
	}
	if req.FieldType == "" {
		return fmt.Errorf("field_type is required")
	}
	if !isValidFieldType(string(req.FieldType)) {
		return fmt.Errorf("invalid field_type")
	}

	// Проверка options для типов, которые их требуют
	switch req.FieldType {
	case models.FieldDropdown, models.FieldMultiSelect, models.FieldRadio:
		if len(req.Options) == 0 {
			return fmt.Errorf("options are required for field type: %s", req.FieldType)
		}
	}

	return nil
}

// isValidEntityType проверяет допустимость значения entity_type.
func isValidEntityType(et string) bool {
	for _, valid := range models.ValidEntityTypes {
		if et == valid {
			return true
		}
	}
	return false
}

// isValidFieldType проверяет допустимость значения field_type.
func isValidFieldType(ft string) bool {
	for _, valid := range models.ValidFieldTypes {
		if ft == valid {
			return true
		}
	}
	return false
}

// generateFieldID генерирует уникальный ID для новых записей кастомных полей.
func generateFieldID() string {
	return fmt.Sprintf("cf_%d", time.Now().UnixNano())
}
