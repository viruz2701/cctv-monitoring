package main

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

type DeviceAdapter interface {
	Start(dev *Device) error
	Stop(dev *Device) error
	GetStatus(dev *Device) (DeviceStatus, error)
	Command(dev *Device, cmd string, params map[string]string) error
}

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

func (ra *ReolinkAdapter) Start(dev *Device) error {
	// Arguments: reolinkproxy --server-advertise-host 127.0.0.1
	// Environment variables: REOLINK_CAMERA_0_NAME, REOLINK_CAMERA_0_UID, etc.
	cmd := exec.Command(ra.binPath)
	cmd.Env = append(cmd.Environ(),
		fmt.Sprintf("REOLINK_CAMERA_0_NAME=%s", dev.ID),
		fmt.Sprintf("REOLINK_CAMERA_0_UID=%s", dev.Serial),
		fmt.Sprintf("REOLINK_CAMERA_0_USERNAME=%s", dev.Username),
		fmt.Sprintf("REOLINK_CAMERA_0_PASSWORD=%s", dev.Password),
		fmt.Sprintf("REOLINK_SERVER_RTSP_ADDRESS=:%d", dev.ProxyPort),
		// optional: onvif port
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	ra.cmdMap[dev.ID] = cmd
	// Wait a bit for RTSP to become ready
	time.Sleep(2 * time.Second)
	dev.RTSPURL = fmt.Sprintf("rtsp://127.0.0.1:%d/%s/stream", dev.ProxyPort, dev.ID)
	return nil
}

func (ra *ReolinkAdapter) Stop(dev *Device) error {
	if cmd, ok := ra.cmdMap[dev.ID]; ok {
		if err := cmd.Process.Kill(); err != nil {
			return err
		}
		delete(ra.cmdMap, dev.ID)
	}
	return nil
}

func (ra *ReolinkAdapter) GetStatus(dev *Device) (DeviceStatus, error) {
	// Could attempt to connect to RTSP port or query proxy API
	return StatusOnline, nil // stub
}

func (ra *ReolinkAdapter) Command(dev *Device, cmd string, params map[string]string) error {
	// Implement PTZ via MQTT or HTTP API of reolinkproxy
	return nil
}
