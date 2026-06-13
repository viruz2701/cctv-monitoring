package adapters

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"p2p-gateway/internal/models"
)

type XiongmaiAdapter struct {
	cmdMap     map[string]*exec.Cmd
	binPath    string
	scriptPath string
}

func NewXiongmaiAdapter(nodePath, scriptPath string) *XiongmaiAdapter {
	return &XiongmaiAdapter{
		cmdMap:     make(map[string]*exec.Cmd),
		binPath:    nodePath,
		scriptPath: scriptPath,
	}
}

func (a *XiongmaiAdapter) Start(dev *models.Device) error {
	port := dev.ProxyPort
	args := []string{
		a.scriptPath,
		"--serial", dev.Serial,
		"--user", dev.Username,
		"--pass", dev.Password,
		"--port", strconv.Itoa(port),
	}
	cmd := exec.Command(a.binPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	a.cmdMap[dev.ID] = cmd

	time.Sleep(3 * time.Second)
	dev.RTSPURL = fmt.Sprintf("rtsp://127.0.0.1:%d/stream", port) // зависит от реализации прокси
	dev.Status = models.StatusOnline
	return nil
}

func (a *XiongmaiAdapter) Stop(dev *models.Device) error {
	if cmd, ok := a.cmdMap[dev.ID]; ok {
		if err := cmd.Process.Kill(); err != nil {
			return err
		}
		delete(a.cmdMap, dev.ID)
	}
	return nil
}

func (a *XiongmaiAdapter) GetStatus(dev *models.Device) (models.DeviceStatus, error) {
	if cmd, ok := a.cmdMap[dev.ID]; ok {
		if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
			return models.StatusOnline, nil
		}
	}
	return models.StatusOffline, nil
}

func (a *XiongmaiAdapter) Command(dev *models.Device, cmd string, params map[string]string) error {
	// PTZ через Xiongmai API (можно добавить)
	return nil
}
