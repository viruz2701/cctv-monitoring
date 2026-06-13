package db

import (
    "context"
    "fmt"
    "time"
    "gb-telemetry-collector/internal/models"
    "github.com/jackc/pgx/v5"
)

func (db *DB) SaveDevice(dev *models.Device) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    _, err := db.Pool.Exec(ctx, `
        INSERT INTO devices (device_id, owner_id, name, location, vendor_type, status, last_seen, registered_at, heartbeat_interval, user_agent, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
        ON CONFLICT (device_id) DO UPDATE SET
            owner_id = EXCLUDED.owner_id,
            name = EXCLUDED.name,
            location = EXCLUDED.location,
            vendor_type = EXCLUDED.vendor_type,
            status = EXCLUDED.status,
            last_seen = EXCLUDED.last_seen,
            heartbeat_interval = EXCLUDED.heartbeat_interval,
            user_agent = EXCLUDED.user_agent,
            updated_at = NOW()
    `, dev.DeviceID, dev.OwnerID, dev.Name, dev.Location, dev.VendorType, dev.Status, dev.LastSeen, dev.RegisteredAt, dev.HeartbeatInterval, dev.UserAgent)
    return err
}

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
        INSERT INTO alarms (time, device_id, priority, method, description)
        VALUES ($1, $2, $3, $4, $5)
    `, time.Now(), alarm.DeviceID, alarm.Priority, alarm.Method, alarm.Description)
    return err
}

func (db *DB) SaveAudit(userUUID, action, entityType, entityID string, oldValue, newValue interface{}) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    _, err := db.Pool.Exec(ctx, `
        INSERT INTO audit_log (timestamp, user_uuid, action, entity_type, entity_id, old_value, new_value)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, time.Now(), userUUID, action, entityType, entityID, oldValue, newValue)
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
    return logs, nil
}

// --- Методы для пользователей (если ещё не добавлены в другом файле) ---
func (db *DB) CreateUser(username, passwordHash, role string, ownerID *string) (*models.User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    var id string
    err := db.Pool.QueryRow(ctx, `
        INSERT INTO users (id, username, password_hash, role, owner_id)
        VALUES (gen_random_uuid()::text, $1, $2, $3, $4)
        RETURNING id
    `, username, passwordHash, role, ownerID).Scan(&id)
    if err != nil {
        return nil, err
    }
    return &models.User{
        ID:       id,
        Username: username,
        Role:     role,
        OwnerID:  ownerID,
    }, nil
}

func (db *DB) GetUserByUsername(username string) (*models.User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    var u models.User
    err := db.Pool.QueryRow(ctx, `
        SELECT id, username, password_hash, role, owner_id, created_at
        FROM users WHERE username = $1
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

// --- Методы для прогнозов ---
func (db *DB) GetPredictions(deviceID string, limit int) ([]models.Prediction, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    var rows pgx.Rows
    var err error
    if deviceID != "" {
        rows, err = db.Pool.Query(ctx, `
            SELECT device_id, prediction_date, failure_probability, explanation
            FROM predictions
            WHERE device_id = $1
            ORDER BY prediction_date DESC
            LIMIT $2
        `, deviceID, limit)
    } else {
        rows, err = db.Pool.Query(ctx, `
            SELECT device_id, prediction_date, failure_probability, explanation
            FROM predictions
            ORDER BY prediction_date DESC
            LIMIT $1
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
    return preds, nil
}
// SearchParsedLogs - обёртка для SearchLogs (совместимость с api)
func (db *DB) SearchParsedLogs(deviceID, level, keyword, timeFrom, timeTo string) ([]models.ParsedLog, error) {
    return db.SearchLogs(deviceID, level, keyword, timeFrom, timeTo)
}
