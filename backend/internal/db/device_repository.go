// Package db — device repository with full CRUD, pagination, and filtering.
// Соответствует:
//   - ISO 27001 A.12.4 (Audit trail через service layer)
//   - OWASP ASVS V5 (Parameterized queries — SQL injection prevention)
//   - СТБ 34.101.27 п. 6.2 (Целостность данных)
package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gb-telemetry-collector/internal/models"

	"github.com/jackc/pgx/v5"
)

// ── Device Repository ──────────────────────────────────────────────────

// CreateDevice создаёт новое устройство в БД.
// Использует parameterized queries (OWASP ASVS V5.2 — SQL injection prevention).
// Соответствует: ISO 27001 A.8.1.2 (Asset inventory)
func (db *DB) CreateDevice(ctx context.Context, dev *models.Device) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO devices (
			device_id, owner_id, site_id, name, location,
			latitude, longitude, vendor_type, device_type,
			status, health, asset_class,
			last_seen, registered_at, created_at, updated_at,
			connection_type, heartbeat_interval, user_agent,
			p2p_brand, p2p_serial, cloud_status,
			manufacturer, serial_number, mac_address, firmware_version
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12,
			$13, $14, NOW(), NOW(),
			$15, $16, $17,
			$18, $19, $20,
			$21, $22, $23, $24
		)
	`,
		dev.DeviceID, dev.OwnerID, dev.SiteID, dev.Name, dev.Location,
		dev.Latitude, dev.Longitude, dev.VendorType, dev.DeviceType,
		dev.Status, dev.Health, dev.AssetClass,
		dev.LastSeen, dev.RegisteredAt,
		dev.ConnectionType, dev.HeartbeatInterval, dev.UserAgent,
		dev.P2PBrand, dev.P2PSerial, dev.CloudStatus,
		dev.Manufacturer, dev.SerialNumber, dev.MacAddress, dev.FirmwareVersion,
	)
	if err != nil {
		return fmt.Errorf("create device %q: %w", dev.DeviceID, err)
	}
	return nil
}

// GetDeviceByID возвращает устройство по ID.
// Соответствует: ISO 27001 A.9.4.2 (Access control — RBAC на уровне service)
func (db *DB) GetDeviceByID(ctx context.Context, deviceID string) (*models.Device, error) {
	var dev models.Device
	var ownerID, siteID, p2pBrand, p2pSerial, cloudStatus *string
	var manufacturer, serialNumber, macAddress, firmwareVersion *string
	var latitude, longitude *float64
	var deviceType, connectionType string
	var health string
	var assetClass string
	var deletedAt *time.Time

	err := db.Pool.QueryRow(ctx, `
		SELECT
			device_id, owner_id, site_id, name, location,
			latitude, longitude, vendor_type, device_type,
			status, health, asset_class,
			last_seen, registered_at, created_at, updated_at, deleted_at,
			connection_type, heartbeat_interval, user_agent,
			p2p_brand, p2p_serial, cloud_status,
			manufacturer, serial_number, mac_address, firmware_version
		FROM devices WHERE device_id = $1
	`, deviceID).Scan(
		&dev.DeviceID, &ownerID, &siteID, &dev.Name, &dev.Location,
		&latitude, &longitude, &dev.VendorType, &deviceType,
		&dev.Status, &health, &assetClass,
		&dev.LastSeen, &dev.RegisteredAt, &dev.CreatedAt, &dev.UpdatedAt, &deletedAt,
		&connectionType, &dev.HeartbeatInterval, &dev.UserAgent,
		&p2pBrand, &p2pSerial, &cloudStatus,
		&manufacturer, &serialNumber, &macAddress, &firmwareVersion,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("device %q not found", deviceID)
		}
		return nil, fmt.Errorf("get device %q: %w", deviceID, err)
	}

	dev.OwnerID = ownerID
	dev.SiteID = siteID
	dev.DeletedAt = deletedAt
	if latitude != nil {
		dev.Latitude = *latitude
	}
	if longitude != nil {
		dev.Longitude = *longitude
	}
	dev.DeviceType = models.DeviceType(deviceType)
	dev.Health = models.HealthStatus(health)
	dev.AssetClass = models.AssetClass(assetClass)
	dev.ConnectionType = models.ConnectionType(connectionType)
	if p2pBrand != nil {
		dev.P2PBrand = *p2pBrand
	}
	if p2pSerial != nil {
		dev.P2PSerial = *p2pSerial
	}
	if cloudStatus != nil {
		dev.CloudStatus = *cloudStatus
	}
	if manufacturer != nil {
		dev.Manufacturer = *manufacturer
	}
	if serialNumber != nil {
		dev.SerialNumber = *serialNumber
	}
	if macAddress != nil {
		dev.MacAddress = *macAddress
	}
	if firmwareVersion != nil {
		dev.FirmwareVersion = *firmwareVersion
	}

	return &dev, nil
}

// ListDevices возвращает список устройств с пагинацией и фильтрацией.
// Использует parameterized queries (OWASP ASVS V5.2).
// Соответствует: ISO 27001 A.12.6.1 (Capacity management)
func (db *DB) ListDevices(ctx context.Context, filter models.ListDevicesFilter) (*models.DeviceListResponse, error) {
	// Валидация пагинации
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = models.DefaultPageSize
	}
	if filter.PageSize > models.MaxPageSize {
		filter.PageSize = models.MaxPageSize
	}

	// Строим WHERE clause динамически с parameterized queries
	where := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if !filter.WithDeleted {
		where = append(where, "deleted_at IS NULL")
	}

	if filter.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.DeviceType != "" {
		where = append(where, fmt.Sprintf("device_type = $%d", argIdx))
		args = append(args, filter.DeviceType)
		argIdx++
	}

	if filter.VendorType != "" {
		where = append(where, fmt.Sprintf("vendor_type = $%d", argIdx))
		args = append(args, filter.VendorType)
		argIdx++
	}

	if filter.SiteID != "" {
		where = append(where, fmt.Sprintf("site_id = $%d", argIdx))
		args = append(args, filter.SiteID)
		argIdx++
	}

	if filter.AssetClass != "" {
		where = append(where, fmt.Sprintf("asset_class = $%d", argIdx))
		args = append(args, filter.AssetClass)
		argIdx++
	}

	if filter.Search != "" {
		where = append(where, fmt.Sprintf(
			"(name ILIKE $%d OR device_id ILIKE $%d OR location ILIKE $%d)",
			argIdx, argIdx+1, argIdx+2,
		))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		argIdx += 3
	}

	whereClause := strings.Join(where, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM devices WHERE %s", whereClause)
	if err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count devices: %w", err)
	}

	// Calculate pagination
	totalPages := (total + filter.PageSize - 1) / filter.PageSize
	offset := (filter.Page - 1) * filter.PageSize

	// Fetch page
	query := fmt.Sprintf(`
		SELECT
			device_id, name, location, vendor_type, device_type,
			status, health, last_seen, site_id
		FROM devices
		WHERE %s
		ORDER BY updated_at DESC, name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, filter.PageSize, offset)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	devices := make([]models.DeviceSummary, 0, filter.PageSize)
	for rows.Next() {
		var d models.DeviceSummary
		var siteID *string
		if err := rows.Scan(
			&d.DeviceID, &d.Name, &d.Location, &d.VendorType, &d.DeviceType,
			&d.Status, &d.Health, &d.LastSeen, &siteID,
		); err != nil {
			return nil, fmt.Errorf("scan device row: %w", err)
		}
		d.SiteID = siteID
		devices = append(devices, d)
	}

	return &models.DeviceListResponse{
		Devices:    devices,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, rows.Err()
}

// UpdateDevice выполняет частичное обновление устройства.
// Обновляет только переданные поля (nil = не менять).
// Соответствует: ISO 27001 A.12.1.2 (Change management)
func (db *DB) UpdateDevice(ctx context.Context, deviceID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	// Whitelist разрешённых для обновления полей (OWASP ASVS V5.1)
	allowedFields := map[string]bool{
		"name": true, "location": true, "latitude": true, "longitude": true,
		"vendor_type": true, "device_type": true, "status": true, "health": true,
		"asset_class": true, "connection_type": true, "heartbeat_interval": true,
		"user_agent": true, "p2p_brand": true, "p2p_serial": true,
		"cloud_status": true, "manufacturer": true, "serial_number": true,
		"mac_address": true, "firmware_version": true, "site_id": true,
		"owner_id": true,
	}

	for field, value := range updates {
		if !allowedFields[field] {
			continue // пропускаем неразрешённые поля
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argIdx))
		args = append(args, value)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	// updated_at всегда обновляется
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, deviceID)

	query := fmt.Sprintf(
		"UPDATE devices SET %s WHERE device_id = $%d AND deleted_at IS NULL",
		strings.Join(setClauses, ", "), argIdx,
	)

	result, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update device %q: %w", deviceID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("device %q not found or already deleted", deviceID)
	}
	return nil
}

// SoftDeleteDevice выполняет мягкое удаление устройства (soft delete).
// Соответствует: ISO 27001 A.8.1.2 (Asset disposal), GDPR Art. 17 (Right to erasure — через hard delete)
func (db *DB) SoftDeleteDevice(ctx context.Context, deviceID string) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE devices
		SET deleted_at = NOW(), updated_at = NOW(), status = 'OFFLINE'
		WHERE device_id = $1 AND deleted_at IS NULL
	`, deviceID)
	if err != nil {
		return fmt.Errorf("soft delete device %q: %w", deviceID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("device %q not found or already deleted", deviceID)
	}
	return nil
}

// HardDeleteDevice выполняет полное удаление устройства (для GDPR Art. 17).
// Использовать только по запросу на удаление данных.
func (db *DB) HardDeleteDevice(ctx context.Context, deviceID string) error {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM devices WHERE device_id = $1
	`, deviceID)
	if err != nil {
		return fmt.Errorf("hard delete device %q: %w", deviceID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("device %q not found", deviceID)
	}
	return nil
}

// RestoreDevice восстанавливает soft-deleted устройство.
func (db *DB) RestoreDevice(ctx context.Context, deviceID string) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE devices
		SET deleted_at = NULL, updated_at = NOW()
		WHERE device_id = $1 AND deleted_at IS NOT NULL
	`, deviceID)
	if err != nil {
		return fmt.Errorf("restore device %q: %w", deviceID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("device %q not found or not deleted", deviceID)
	}
	return nil
}
