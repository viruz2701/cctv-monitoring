package adapters

import "p2p-gateway/internal/models"

type DeviceAdapter interface {
	Start(dev *models.Device) error
	Stop(dev *models.Device) error
	GetStatus(dev *models.Device) (models.DeviceStatus, error)
	Command(dev *models.Device, cmd string, params map[string]string) error
	Snapshot(serial string) ([]byte, error)
}
