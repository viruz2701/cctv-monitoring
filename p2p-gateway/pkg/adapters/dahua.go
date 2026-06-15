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

type DahuaAdapter struct {
	cmdMap     map[string]*exec.Cmd
	pythonPath string
	scriptPath string
}

func NewDahuaAdapter(pythonPath, scriptPath string) *DahuaAdapter {
	return &DahuaAdapter{
		cmdMap:     make(map[string]*exec.Cmd),
		pythonPath: pythonPath,
		scriptPath: scriptPath,
	}
}

func (a *DahuaAdapter) Start(dev *models.Device) error {
	args := []string{
		dev.Serial,
		"-u", dev.Username,
		"-p", dev.Password,
		"--type", "1",
		"--port", strconv.Itoa(dev.ProxyPort),
	}
	// Если нужен security_code – раскомментировать:
	// if dev.SecurityCode != "" {
	//     args = append(args, "--security-code", dev.SecurityCode)
	// }
	allArgs := append([]string{a.scriptPath}, args...)
	cmd := exec.Command(a.pythonPath, allArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start dahua script: %w", err)
	}
	a.cmdMap[dev.ID] = cmd

	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		if isPortOpen(dev.ProxyPort) {
			break
		}
	}

	dev.RTSPURL = fmt.Sprintf("rtsp://127.0.0.1:%d/cam/realmonitor?channel=1&subtype=0", dev.ProxyPort)
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
	if _, ok := a.cmdMap[dev.ID]; ok {
		return models.StatusOnline, nil
	}
	return models.StatusOffline, nil
}

func (a *DahuaAdapter) Command(dev *models.Device, cmd string, params map[string]string) error {
	return nil
}

func (a *DahuaAdapter) Snapshot(serial string) ([]byte, error) {
	return nil, fmt.Errorf("snapshot not implemented")
}
