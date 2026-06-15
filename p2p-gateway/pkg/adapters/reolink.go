package adapters

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"p2p-gateway/internal/models"
)

type ReolinkAdapter struct {
	binPath string
	cmdMap  map[string]*exec.Cmd
}

func NewReolinkAdapter(binPath string) *ReolinkAdapter {
	return &ReolinkAdapter{
		binPath: binPath,
		cmdMap:  make(map[string]*exec.Cmd),
	}
}

func (a *ReolinkAdapter) Start(dev *models.Device) error {
	args := []string{
		"-serial", dev.Serial,
		"-user", dev.Username,
		"-pass", dev.Password,
		"-rtsp-port", strconv.Itoa(dev.ProxyPort),
	}
	cmd := exec.Command(a.binPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Перенаправляем вывод для отладки
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start reolinkproxy: %w", err)
	}
	a.cmdMap[dev.ID] = cmd

	// Ждём, пока порт станет доступен (максимум 10 секунд)
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		if isPortOpen(dev.ProxyPort) {
			break
		}
	}

	dev.RTSPURL = fmt.Sprintf("rtsp://127.0.0.1:%d/stream", dev.ProxyPort)
	dev.Status = models.StatusOnline
	return nil
}

func (a *ReolinkAdapter) Stop(dev *models.Device) error {
	if cmd, ok := a.cmdMap[dev.ID]; ok {
		cmd.Process.Kill()
		delete(a.cmdMap, dev.ID)
	}
	return nil
}

func (a *ReolinkAdapter) GetStatus(dev *models.Device) (models.DeviceStatus, error) {
	if _, ok := a.cmdMap[dev.ID]; ok {
		return models.StatusOnline, nil
	}
	return models.StatusOffline, nil
}

func (a *ReolinkAdapter) Command(dev *models.Device, cmd string, params map[string]string) error {
	return nil
}

func (a *ReolinkAdapter) Snapshot(serial string) ([]byte, error) {
	return nil, fmt.Errorf("snapshot not implemented")
}
