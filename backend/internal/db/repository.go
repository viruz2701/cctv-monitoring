package db

import (
	"context"
	"fmt"
	"strings"
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
		SELECT id, username, password_hash, role, owner_id, created_at,
		       COALESCE(email, ''), COALESCE(totp_secret, ''), COALESCE(totp_enabled, false),
		       COALESCE(telegram_chat_id, ''), COALESCE(telegram_alerts, false), COALESCE(telegram_2fa, false)
		FROM users WHERE username = $1 AND status = 'active'
	`, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.OwnerID, &u.CreatedAt, &u.Email, &u.TOTPSecret, &u.TOTPEnabled, &u.TelegramChatID, &u.TelegramAlerts, &u.Telegram2FA)
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
		SELECT id, username, password_hash, role, owner_id, created_at,
		       COALESCE(totp_secret, ''), COALESCE(totp_enabled, false),
		       COALESCE(telegram_chat_id, ''), COALESCE(telegram_alerts, false), COALESCE(telegram_2fa, false)
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.OwnerID, &u.CreatedAt, &u.TOTPSecret, &u.TOTPEnabled, &u.TelegramChatID, &u.TelegramAlerts, &u.Telegram2FA)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *DB) CreateUser(username, passwordHash, role, email string, ownerID *string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var id string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO users (id, username, password_hash, role, email, owner_id)
		VALUES (gen_random_uuid()::text, $1, $2, $3, $4, $5) RETURNING id
	`, username, passwordHash, role, email, ownerID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &models.User{ID: id, Username: username, Role: role, Email: email, OwnerID: ownerID}, nil
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

// backend/internal/db/repository.go — заменить метод GetUsers

// GetUsers возвращает список всех пользователей (без паролей)
func (db *DB) GetUsers() ([]models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT id, username, role, owner_id, email, avatar, status, last_login, created_at 
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		// Используем указатели для nullable-колонок
		var ownerID, email, avatar, status *string
		var lastLogin *time.Time

		err := rows.Scan(
			&u.ID, &u.Username, &u.Role,
			&ownerID, &email, &avatar, &status,
			&lastLogin, &u.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Присваиваем значения, если они не NULL
		if ownerID != nil {
			u.OwnerID = ownerID
		}
		if email != nil {
			u.Email = *email
		}
		if avatar != nil {
			u.Avatar = *avatar
		}
		if status != nil {
			u.Status = *status
		} else {
			u.Status = "active" // дефолтное значение, если NULL
		}
		u.LastLogin = lastLogin

		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdateUser обновляет данные пользователя
func (db *DB) UpdateUser(id string, updates map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for key, val := range updates {
		if key == "role" || key == "status" || key == "email" || key == "avatar" || key == "password_hash" {
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIdx))
			args = append(args, val)
			argIdx++
		}
	}

	if len(setClauses) == 0 {
		return nil
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	_, err := db.Pool.Exec(ctx, query, args...)
	return err
}

// DeleteUser выполняет soft delete
func (db *DB) DeleteUser(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		UPDATE users SET status = 'inactive', updated_at = NOW() WHERE id = $1
	`, id)
	return err
}

// backend/internal/db/repository.go — добавить в конец файла

// ═══════════════════════════════════════════════════════════════════════
// Password Management
// ═══════════════════════════════════════════════════════════════════════

// UpdatePassword обновляет хеш пароля пользователя
func (db *DB) UpdatePassword(userID string, newPasswordHash string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2
	`, newPasswordHash, userID)
	return err
}

// GetPasswordHash возвращает хеш пароля пользователя (для проверки при смене)
func (db *DB) GetPasswordHash(userID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var hash string
	err := db.Pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if err != nil {
		return "", err
	}
	return hash, nil
}

// ═══════════════════════════════════════════════════════════════════════
// User Sessions
// ═══════════════════════════════════════════════════════════════════════

type UserSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (db *DB) GetUserSessions(userID string) ([]UserSession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, ip_address, user_agent, expires_at, created_at
		FROM user_sessions
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []UserSession
	for rows.Next() {
		var s UserSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// ═══════════════════════════════════════════════════════════════════════
// TOTP (2FA)
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) UpdateTOTPSecret(userID string, secret string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Pool.Exec(ctx, `UPDATE users SET totp_secret = $1, updated_at = NOW() WHERE id = $2`, secret, userID)
	return err
}

func (db *DB) EnableTOTP(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Pool.Exec(ctx, `UPDATE users SET totp_enabled = true, updated_at = NOW() WHERE id = $1`, userID)
	return err
}

func (db *DB) DisableTOTP(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Pool.Exec(ctx, `UPDATE users SET totp_enabled = false, totp_secret = '', updated_at = NOW() WHERE id = $1`, userID)
	return err
}

// ═══════════════════════════════════════════════════════════════════════
// Telegram Integration
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) UpdateTelegramChatID(userID string, chatID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Pool.Exec(ctx, `UPDATE users SET telegram_chat_id = $1, updated_at = NOW() WHERE id = $2`, chatID, userID)
	return err
}

func (db *DB) UpdateTelegramSettings(userID string, alerts, tfa bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Pool.Exec(ctx, `UPDATE users SET telegram_alerts = $1, telegram_2fa = $2, updated_at = NOW() WHERE id = $3`, alerts, tfa, userID)
	return err
}

func (db *DB) GetUserByTelegramChatID(chatID string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var u models.User
	err := db.Pool.QueryRow(ctx, `
		SELECT id, username, password_hash, role, owner_id, created_at,
		       COALESCE(totp_secret, ''), COALESCE(totp_enabled, false),
		       COALESCE(telegram_chat_id, ''), COALESCE(telegram_alerts, false), COALESCE(telegram_2fa, false)
		FROM users WHERE telegram_chat_id = $1 AND status = 'active'
	`, chatID).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.OwnerID, &u.CreatedAt, &u.TOTPSecret, &u.TOTPEnabled, &u.TelegramChatID, &u.TelegramAlerts, &u.Telegram2FA)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *DB) SaveTelegramLinkToken(token, userID string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO telegram_link_tokens (token, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (token) DO UPDATE SET user_id = $2, expires_at = $3
	`, token, userID, expiresAt)
	return err
}

func (db *DB) GetTelegramLinkToken(token string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var userID string
	err := db.Pool.QueryRow(ctx, `
		SELECT user_id FROM telegram_link_tokens WHERE token = $1 AND expires_at > NOW()
	`, token).Scan(&userID)
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (db *DB) DeleteTelegramLinkToken(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.Pool.Exec(ctx, `DELETE FROM telegram_link_tokens WHERE token = $1`, token)
	return err
}

func (db *DB) RevokeSession(sessionID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		DELETE FROM user_sessions WHERE id = $1
	`, sessionID)
	return err
}

func (db *DB) RevokeAllOtherSessions(userID string, currentSessionID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		DELETE FROM user_sessions WHERE user_id = $1 AND id != $2
	`, userID, currentSessionID)
	return err
}

// ═══════════════════════════════════════════════════════════════════════
// Tickets (for Telegram integration)
// ═══════════════════════════════════════════════════════════════════════

type Ticket struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DeviceID    string    `json:"device_id"`
	Priority    string    `json:"priority"`
	Status      string    `json:"status"`
	Assignee    string    `json:"assignee"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (db *DB) GetTicketsByUserID(userID string) ([]Ticket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT id, title, description, device_id, priority, status, assignee, created_at, updated_at
		FROM tickets
		WHERE assignee = $1 AND status IN ('open', 'in_progress')
		ORDER BY created_at DESC
		LIMIT 10
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var t Ticket
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.DeviceID, &t.Priority, &t.Status, &t.Assignee, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (db *DB) UpdateTicketStatus(ticketID, status, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		UPDATE tickets
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`, status, ticketID)
	return err
}

// ═══════════════════════════════════════════════════════════════════════
// Telegram Login Codes
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) SaveTelegramLoginCode(userID, code string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO telegram_login_codes (user_id, code, expires_at, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id) DO UPDATE SET code = $2, expires_at = $3, created_at = NOW()
	`, userID, code, expiresAt)
	return err
}

func (db *DB) ValidateTelegramLoginCode(userID, code string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var storedCode string
	var expiresAt time.Time
	err := db.Pool.QueryRow(ctx, `
		SELECT code, expires_at FROM telegram_login_codes WHERE user_id = $1
	`, userID).Scan(&storedCode, &expiresAt)
	if err != nil {
		return false, err
	}

	if time.Now().After(expiresAt) {
		return false, nil
	}

	if storedCode != code {
		return false, nil
	}

	// Delete used code
	_, _ = db.Pool.Exec(ctx, `DELETE FROM telegram_login_codes WHERE user_id = $1`, userID)
	return true, nil
}

// ═══════════════════════════════════════════════════════════════════════
// API Keys
// ═══════════════════════════════════════════════════════════════════════

type APIKey struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"-"`
	KeyPrefix   string     `json:"-"`
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (db *DB) CreateAPIKey(id, name, keyHash, keyPrefix string, permissions []string, expiresAt *time.Time, createdBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO api_keys (id, name, key_hash, key_prefix, permissions, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, name, keyHash, keyPrefix, permissions, expiresAt, createdBy)
	return err
}

func (db *DB) GetAPIKeys(createdBy string) ([]APIKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT id, name, key_hash, COALESCE(key_prefix, ''), permissions, expires_at, last_used_at, created_by, created_at
		FROM api_keys
		WHERE created_by = $1
		ORDER BY created_at DESC
	`, createdBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var key APIKey
		if err := rows.Scan(&key.ID, &key.Name, &key.KeyHash, &key.KeyPrefix, &key.Permissions, &key.ExpiresAt, &key.LastUsedAt, &key.CreatedBy, &key.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// GetAPIKeysByPrefix returns all API keys matching the given prefix (for bcrypt lookup)
func (db *DB) GetAPIKeysByPrefix(prefix string) ([]APIKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT id, name, key_hash, COALESCE(key_prefix, ''), permissions, expires_at, last_used_at, created_by, created_at
		FROM api_keys
		WHERE key_prefix = $1
	`, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var key APIKey
		if err := rows.Scan(&key.ID, &key.Name, &key.KeyHash, &key.KeyPrefix, &key.Permissions, &key.ExpiresAt, &key.LastUsedAt, &key.CreatedBy, &key.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func (db *DB) RevokeAPIKey(id, createdBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		DELETE FROM api_keys
		WHERE id = $1 AND created_by = $2
	`, id, createdBy)
	return err
}

// Password Reset Token methods
func (db *DB) CreatePasswordResetToken(userID, token string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET token = $2, expires_at = $3
	`, userID, token, expiresAt)
	return err
}

func (db *DB) GetPasswordResetToken(token string) (string, time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var userID string
	var expiresAt time.Time
	err := db.Pool.QueryRow(ctx, `
		SELECT user_id, expires_at FROM password_reset_tokens WHERE token = $1
	`, token).Scan(&userID, &expiresAt)
	if err != nil {
		return "", time.Time{}, err
	}
	return userID, expiresAt, nil
}

func (db *DB) DeletePasswordResetToken(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `DELETE FROM password_reset_tokens WHERE token = $1`, token)
	return err
}

func (db *DB) GetUserByEmail(email string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var u models.User
	err := db.Pool.QueryRow(ctx, `
		SELECT id, username, password_hash, role, owner_id, created_at,
		       COALESCE(email, ''), COALESCE(totp_secret, ''), COALESCE(totp_enabled, false),
		       COALESCE(telegram_chat_id, ''), COALESCE(telegram_alerts, false), COALESCE(telegram_2fa, false)
		FROM users WHERE email = $1 AND status = 'active'
	`, email).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.OwnerID, &u.CreatedAt, &u.Email, &u.TOTPSecret, &u.TOTPEnabled, &u.TelegramChatID, &u.TelegramAlerts, &u.Telegram2FA)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *DB) UpdateAPIKeyLastUsed(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		UPDATE api_keys
		SET last_used_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

// SiteInfo содержит координаты объекта для Gatekeeper-верификации.
type SiteInfo struct {
	SiteID               string
	SiteName             string
	Latitude             float64
	Longitude            float64
	GeofenceRadiusMeters float64
}

// GetSiteInfo возвращает информацию об объекте (координаты) для наряда.
// Используется Gatekeeper Service для верификации GPS и EXIF.
func (db *DB) GetSiteInfo(ctx context.Context, workOrderID string) (*SiteInfo, error) {
	var info SiteInfo
	err := db.Pool.QueryRow(ctx, `
		SELECT d.device_id, COALESCE(d.name, d.device_id),
		       COALESCE(d.latitude, 0), COALESCE(d.longitude, 0),
		       COALESCE(d.geofence_radius_meters, 500)
		FROM work_orders wo
		JOIN devices d ON wo.device_id = d.device_id
		WHERE wo.id = $1
	`, workOrderID).Scan(&info.SiteID, &info.SiteName, &info.Latitude, &info.Longitude, &info.GeofenceRadiusMeters)
	if err != nil {
		return nil, fmt.Errorf("get site info for work_order %s: %w", workOrderID, err)
	}
	return &info, nil
}
