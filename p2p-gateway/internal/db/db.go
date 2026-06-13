package db

import (
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/yourorg/p2p-gateway/internal/config"
	"github.com/yourorg/p2p-gateway/internal/models"
)

type DB struct {
	*sqlx.DB
}

func New(cfg *config.DatabaseConfig) (*DB, error) {
	connStr := "host=" + cfg.Host + " port=" + cfg.Port + " user=" + cfg.User +
		" password=" + cfg.Password + " dbname=" + cfg.Name + " sslmode=" + cfg.SSLMode
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func (db *DB) CreateP2PDevice(dev *models.P2PDevice) error {
	query := `INSERT INTO p2p_devices (serial, brand, security_code, cloud_user, cloud_pass, status, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW()) RETURNING id`
	return db.QueryRowx(query, dev.Serial, dev.Brand, dev.SecurityCode, dev.CloudUser, dev.CloudPass, dev.Status).Scan(&dev.ID)
}

func (db *DB) GetP2PDeviceBySerial(serial string) (*models.P2PDevice, error) {
	var dev models.P2PDevice
	err := db.Get(&dev, "SELECT * FROM p2p_devices WHERE serial=$1", serial)
	return &dev, err
}

func (db *DB) UpdateDeviceStatus(serial string, status string, lastSeen time.Time) error {
	_, err := db.Exec("UPDATE p2p_devices SET status=$1, last_seen=$2, updated_at=NOW() WHERE serial=$3", status, lastSeen, serial)
	return err
}

func (db *DB) ListP2PDevices() ([]models.P2PDevice, error) {
	var devices []models.P2PDevice
	err := db.Select(&devices, "SELECT * FROM p2p_devices ORDER BY created_at DESC")
	return devices, err
}
