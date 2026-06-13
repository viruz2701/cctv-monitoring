package adapters

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"p2p-gateway/internal/models"
)

type DahuaAdapter struct {
	cmdMap     map[string]*exec.Cmd
	binPath    string
	scriptPath string
}

func NewDahuaAdapter(pythonPath, scriptPath string) *DahuaAdapter {
	return &DahuaAdapter{
		cmdMap:     make(map[string]*exec.Cmd),
		binPath:    pythonPath,
		scriptPath: scriptPath,
	}
}

func (a *DahuaAdapter) Start(dev *models.Device) error {
	port := dev.ProxyPort
	args := []string{
		a.scriptPath,
		"--serial", dev.Serial,
		"--username", dev.Username,
		"--password", dev.Password,
		"--port", strconv.Itoa(port),
		"--type", "1", // используем тип с авторизацией
	}
	if dev.SecurityCode != "" {
		args = append(args, "--security-code", dev.SecurityCode)
	}
	cmd := exec.Command(a.binPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	a.cmdMap[dev.ID] = cmd

	// Ждём готовности RTSP (можно проверить появлением порта)
	time.Sleep(3 * time.Second)
	dev.RTSPURL = fmt.Sprintf("rtsp://127.0.0.1:%d/cam/realmonitor?channel=1&subtype=0", port)
	dev.Status = models.StatusOnline
	return nil
}

func (a *DahuaAdapter) Stop(dev *models.Device) error {
	if cmd, ok := a.cmdMap[dev.ID]; ok {
		if err := cmd.Process.Kill(); err != nil {
			return err
		}
		delete(a.cmdMap, dev.ID)
	}
	return nil
}

func (a *DahuaAdapter) GetStatus(dev *models.Device) (models.DeviceStatus, error) {
	// Можно проверить, жив ли процесс
	if cmd, ok := a.cmdMap[dev.ID]; ok {
		if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
			return models.StatusOnline, nil
		}
	}
	return models.StatusOffline, nil
}

func (a *DahuaAdapter) Command(dev *models.Device, cmd string, params map[string]string) error {
	// Реализация PTZ через HTTP API dh-p2p (не реализовано в оригинале, заглушка)
	return nil
}
