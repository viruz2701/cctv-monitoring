package db

import (
	"context"
	"fmt"
	"time"

	"gb-telemetry-collector/internal/models"

	"github.com/jackc/pgx/v5"
)

// ═══════════════════════════════════════════════════════════════════════
// Devices (с поддержкой GB28181)
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) SaveDevice(dev *models.Device) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO devices (
			device_id, owner_id, site_id, name, location, vendor_type, device_type,
			status, health, recording_status, last_seen, registered_at,
			heartbeat_interval, user_agent, connection_type,
			p2p_brand, p2p_serial, p2p_security_code, p2p_cloud_user, p2p_cloud_pass, cloud_status,
			snmp_community, snmp_version, syslog_port, alarm_protocol,
			gb28181_device_id, gb28181_device_type, gb28181_parent_id, gb28181_sip_port,
			gb28181_realm, gb28181_register_expires, gb28181_last_register, gb28181_channel_count,
			onvif_url, onvif_username, onvif_password,
			manufacturer, serial_number, mac_address, firmware_version,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15,
			$16, $17, $18, $19, $20, $21,
			$22, $23, $24, $25,
			$26, $27, $28, $29,
			$30, $31, $32, $33,
			$34, $35, $36,
			$37, $38, $39, $40,
			NOW()
		)
		ON CONFLICT (device_id) DO UPDATE SET
			owner_id = EXCLUDED.owner_id,
			site_id = EXCLUDED.site_id,
			name = EXCLUDED.name,
			location = EXCLUDED.location,
			vendor_type = EXCLUDED.vendor_type,
			status = EXCLUDED.status,
			health = EXCLUDED.health,
			last_seen = EXCLUDED.last_seen,
			heartbeat_interval = EXCLUDED.heartbeat_interval,
			user_agent = EXCLUDED.user_agent,
			cloud_status = EXCLUDED.cloud_status,
			gb28181_device_id = EXCLUDED.gb28181_device_id,
			gb28181_parent_id = EXCLUDED.gb28181_parent_id,
			gb28181_last_register = EXCLUDED.gb28181_last_register,
			gb28181_channel_count = EXCLUDED.gb28181_channel_count,
			manufacturer = COALESCE(EXCLUDED.manufacturer, devices.manufacturer),
			firmware_version = COALESCE(EXCLUDED.firmware_version, devices.firmware_version),
			updated_at = NOW()
	`,
		dev.DeviceID, dev.OwnerID, nil, dev.Name, dev.Location, dev.VendorType, nil,
		dev.Status, nil, nil, dev.LastSeen, dev.RegisteredAt,
		dev.HeartbeatInterval, dev.UserAgent, nil,
		dev.P2PBrand, dev.P2PSerial, nil, nil, nil, dev.CloudStatus,
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
		nil, nil, nil, nil,
	)
	return err
}

func (db *DB) GetDeviceByID(deviceID string) (*models.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var dev models.Device
	err := db.Pool.QueryRow(ctx, `
		SELECT device_id, owner_id, name, location, vendor_type, status,
			   last_seen, registered_at, heartbeat_interval, user_agent,
			   connection_type, p2p_brand, p2p_serial, cloud_status,
			   gb28181_device_id, gb28181_parent_id, gb28181_channel_count
		FROM devices WHERE device_id = $1
	`, deviceID).Scan(
		&dev.DeviceID, &dev.OwnerID, &dev.Name, &dev.Location, &dev.VendorType, &dev.Status,
		&dev.LastSeen, &dev.RegisteredAt, &dev.HeartbeatInterval, &dev.UserAgent,
		nil, &dev.P2PBrand, &dev.P2PSerial, &dev.CloudStatus,
		nil, nil, nil,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("device %q not found", deviceID)
		}
		return nil, err
	}
	return &dev, nil
}

func (db *DB) GetDevicesByGB28181Parent(parentID string) ([]models.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT device_id, name, vendor_type, status, last_seen, gb28181_device_id
		FROM devices WHERE gb28181_parent_id = $1 ORDER BY name
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var dev models.Device
		if err := rows.Scan(&dev.DeviceID, &dev.Name, &dev.VendorType, &dev.Status, &dev.LastSeen, nil); err != nil {
			return nil, err
		}
		devices = append(devices, dev)
	}
	return devices, rows.Err()
}

// ═══════════════════════════════════════════════════════════════════════
// Telemetry & Alarms
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) SaveTelemetry(deviceID string, status models.DeviceStatus, lastSeen time.Time, heartbeatInterval int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO telemetry (time, device_id, status, last_seen, heartbeat_interval)
		VALUES ($1, $2, $3, $4, $5)
	`, time.Now(), deviceID, status, lastSeen, heartbeatInterval)
	return err
}

func (db *DB) SaveAlarm(alarm *models.Alarm) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO alarms (time, device_id, priority, method, description, image_path)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, alarm.Timestamp, alarm.DeviceID, alarm.Priority, alarm.Method, alarm.Description, alarm.ImagePath)
	return err
}

func (db *DB) SaveParsedLog(log *models.ParsedLog) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO parsed_logs (time, device_id, log_level, event_code, message, source, raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, log.Time, log.DeviceID, log.LogLevel, log.EventCode, log.Message, log.Source, log.Raw)
	return err
}

// ═══════════════════════════════════════════════════════════════════════
// Users
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var u models.User
	err := db.Pool.QueryRow(ctx, `
		SELECT id, username, password_hash, role, owner_id, created_at
		FROM users WHERE username = $1 AND status = 'active'
	`, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.OwnerID, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *DB) GetUserByID(id string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var u models.User
	err := db.Pool.QueryRow(ctx, `
		SELECT id, username, password_hash, role, owner_id, created_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.OwnerID, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *DB) CreateUser(username, passwordHash, role string, ownerID *string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var id string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO users (id, username, password_hash, role, owner_id)
		VALUES (gen_random_uuid()::text, $1, $2, $3, $4) RETURNING id
	`, username, passwordHash, role, ownerID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &models.User{ID: id, Username: username, Role: role, OwnerID: ownerID}, nil
}

// ═══════════════════════════════════════════════════════════════════════
// Predictions
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) GetPredictions(deviceID string, limit int) ([]models.Prediction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rows pgx.Rows
	var err error
	if deviceID != "" {
		rows, err = db.Pool.Query(ctx, `
			SELECT device_id, prediction_date, failure_probability, explanation
			FROM predictions WHERE device_id = $1
			ORDER BY prediction_date DESC LIMIT $2
		`, deviceID, limit)
	} else {
		rows, err = db.Pool.Query(ctx, `
			SELECT DISTINCT ON (device_id) device_id, prediction_date, failure_probability, explanation
			FROM predictions ORDER BY device_id, prediction_date DESC LIMIT $1
		`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var preds []models.Prediction
	for rows.Next() {
		var p models.Prediction
		if err := rows.Scan(&p.DeviceID, &p.PredictionDate, &p.FailureProbability, &p.Explanation); err != nil {
			return nil, err
		}
		preds = append(preds, p)
	}
	return preds, rows.Err()
}

// ═══════════════════════════════════════════════════════════════════════
// Logs Search
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) SearchLogs(deviceID, level, keyword, timeFrom, timeTo string) ([]models.ParsedLog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `SELECT time, device_id, log_level, event_code, message, source, raw FROM parsed_logs WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if deviceID != "" {
		query += fmt.Sprintf(" AND device_id = $%d", argIdx)
		args = append(args, deviceID)
		argIdx++
	}
	if level != "" {
		query += fmt.Sprintf(" AND log_level = $%d", argIdx)
		args = append(args, level)
		argIdx++
	}
	if keyword != "" {
		query += fmt.Sprintf(" AND (message ILIKE $%d OR raw ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+keyword+"%")
		argIdx++
	}
	if timeFrom != "" {
		query += fmt.Sprintf(" AND time >= $%d", argIdx)
		args = append(args, timeFrom)
		argIdx++
	}
	if timeTo != "" {
		query += fmt.Sprintf(" AND time <= $%d", argIdx)
		args = append(args, timeTo)
		argIdx++
	}
	query += " ORDER BY time DESC LIMIT 1000"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.ParsedLog
	for rows.Next() {
		var l models.ParsedLog
		if err := rows.Scan(&l.Time, &l.DeviceID, &l.LogLevel, &l.EventCode, &l.Message, &l.Source, &l.Raw); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// ═══════════════════════════════════════════════════════════════════════
// Audit
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) SaveAudit(userUUID, action, entityType, entityID string, oldValue, newValue interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO audit_log (timestamp, user_id, action, entity_type, entity_id, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, time.Now(), userUUID, action, entityType, entityID, oldValue, newValue)
	return err
}
