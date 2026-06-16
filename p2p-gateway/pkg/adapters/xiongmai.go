package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"p2p-gateway/internal/models"
	"p2p-gateway/pkg/jftech"
)

type XiongmaiAdapter struct {
	client  *jftech.Client
	region  string
	devices map[string]*deviceSession
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

type deviceSession struct {
	device     *models.Device
	token      string
	rtspURL    string
	stopChan   chan struct{}
	lastStatus models.DeviceStatus
}

func NewXiongmaiAdapter(cfg *jftech.Config, region string) *XiongmaiAdapter {
	ctx, cancel := context.WithCancel(context.Background())
	return &XiongmaiAdapter{
		client:  jftech.NewClient(cfg),
		region:  region,
		devices: make(map[string]*deviceSession),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start регистрирует устройство и запускает сессию
func (a *XiongmaiAdapter) Start(dev *models.Device) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.devices[dev.ID]; exists {
		return nil
	}

	session := &deviceSession{
		device:   dev,
		stopChan: make(chan struct{}),
	}

	if dev.IPAddress != "" {
		rtspURL := fmt.Sprintf("rtsp://%s:%s@%s:554/streaming/channels/%d",
			dev.Username, dev.Password, dev.IPAddress, 1)
		session.rtspURL = rtspURL
		session.token = ""
		session.lastStatus = models.StatusOnline
	} else {
		tokenMap, err := a.client.GetDeviceToken([]string{dev.Serial}, "")
		if err != nil {
			return fmt.Errorf("failed to get device token: %w", err)
		}
		token, ok := tokenMap[dev.Serial]
		if !ok {
			return fmt.Errorf("device %s not found in token response", dev.Serial)
		}
		session.token = token

		protocols := []string{"rtsp-sdp", "hls-ts", "flv"}
		var lastErr error
		for _, proto := range protocols {
			url, err := a.client.GetLivestream(
				token,
				"0",
				"1",
				proto,
				dev.Username,
				dev.Password,
				nil,
			)
			if err == nil && url != "" {
				session.rtspURL = url
				break
			}
			lastErr = err
		}
		if session.rtspURL == "" {
			return fmt.Errorf("failed to get livestream: %w", lastErr)
		}
		session.lastStatus = models.StatusUnknown
	}

	a.devices[dev.ID] = session

	if session.token != "" {
		go a.keepalive(session)
	}

	return nil
}

// Stop завершает сессию устройства
func (a *XiongmaiAdapter) Stop(dev *models.Device) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	session, exists := a.devices[dev.ID]
	if !exists {
		return nil
	}
	close(session.stopChan)
	delete(a.devices, dev.ID)
	return nil
}

// GetStatus возвращает текущий статус устройства
func (a *XiongmaiAdapter) GetStatus(dev *models.Device) (models.DeviceStatus, error) {
	a.mu.RLock()
	session, exists := a.devices[dev.ID]
	a.mu.RUnlock()

	if !exists {
		if dev.IPAddress != "" {
			conn, err := net.DialTimeout("tcp", dev.IPAddress+":554", 2*time.Second)
			if err != nil {
				return models.StatusOffline, nil
			}
			conn.Close()
			return models.StatusOnline, nil
		}
		tokenMap, err := a.client.GetDeviceToken([]string{dev.Serial}, "")
		if err != nil {
			return models.StatusOffline, nil
		}
		token, ok := tokenMap[dev.Serial]
		if !ok {
			return models.StatusOffline, nil
		}
		statuses, err := a.client.DeviceStatus([]string{token}, a.region)
		if err != nil || len(statuses) == 0 {
			return models.StatusOffline, nil
		}
		if statuses[0].Status == "online" {
			return models.StatusOnline, nil
		}
		return models.StatusOffline, nil
	}

	if session.token != "" {
		statuses, err := a.client.DeviceStatus([]string{session.token}, a.region)
		if err != nil || len(statuses) == 0 {
			session.lastStatus = models.StatusOffline
			return session.lastStatus, nil
		}
		if statuses[0].Status == "online" {
			session.lastStatus = models.StatusOnline
		} else {
			session.lastStatus = models.StatusOffline
		}
		return session.lastStatus, nil
	}

	return session.lastStatus, nil
}

// GetSnapshot получает снимок с устройства
func (a *XiongmaiAdapter) GetSnapshot(dev *models.Device) ([]byte, error) {
	a.mu.RLock()
	session, exists := a.devices[dev.ID]
	a.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("device not started")
	}

	if session.token == "" {
		return nil, fmt.Errorf("snapshot not supported in direct mode")
	}

	imageURL, err := a.client.Capture(session.token, 0, 0)
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// SendPTZCommand отправляет PTZ-команду
func (a *XiongmaiAdapter) SendPTZCommand(dev *models.Device, command string, speed int) error {
	a.mu.RLock()
	session, exists := a.devices[dev.ID]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("device not started")
	}

	if session.token == "" {
		return fmt.Errorf("PTZ not supported in direct mode")
	}

	cmdMap := map[string]string{
		"up":    "DirectionUp",
		"down":  "DirectionDown",
		"left":  "DirectionLeft",
		"right": "DirectionRight",
	}
	jfCmd, ok := cmdMap[command]
	if !ok {
		return fmt.Errorf("unsupported PTZ command: %s", command)
	}
	return a.client.PTZControl(session.token, jfCmd, 0, -1, speed)
}

// keepalive периодически обновляет статус и продлевает сессию
func (a *XiongmaiAdapter) keepalive(session *deviceSession) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_, _ = a.GetStatus(session.device)
		case <-session.stopChan:
			return
		case <-a.ctx.Done():
			return
		}
	}
}

// GetLogs возвращает логи устройства за указанный период
func (a *XiongmaiAdapter) GetLogs(dev *models.Device, startTime, endTime string) ([]string, error) {
	a.mu.RLock()
	session, exists := a.devices[dev.ID]
	a.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("device not started")
	}

	if session.token == "" {
		return nil, fmt.Errorf("logs are only available in cloud mode (device token required)")
	}

	rawData, err := a.client.GetDeviceLogs(session.token, startTime, endTime, "All")
	if err != nil {
		return nil, fmt.Errorf("failed to get device logs: %w", err)
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			Name       string `json:"Name"`
			Ret        int    `json:"Ret"`
			OPLogQuery []struct {
				Data     string `json:"Data"`
				Position int    `json:"Position"`
				Time     string `json:"Time"`
				Type     string `json:"Type"`
				User     string `json:"User"`
			} `json:"OPLogQuery"`
		} `json:"data"`
		Msg string `json:"msg"`
	}

	if err := json.Unmarshal(rawData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse logs response: %w", err)
	}

	if result.Code != 2000 {
		return nil, fmt.Errorf("logs request failed: %s", result.Msg)
	}

	if result.Data.Ret != 100 {
		return nil, fmt.Errorf("device returned error: Ret=%d", result.Data.Ret)
	}

	logs := make([]string, 0, len(result.Data.OPLogQuery))
	for _, entry := range result.Data.OPLogQuery {
		logLine := fmt.Sprintf("[%s] [%s] [%s] %s", entry.Time, entry.Type, entry.User, entry.Data)
		logs = append(logs, logLine)
	}

	return logs, nil
}

// Command реализует метод интерфейса DeviceAdapter
func (a *XiongmaiAdapter) Command(dev *models.Device, cmd string, params map[string]string) error {
	switch cmd {
	case "ptz":
		command, ok := params["command"]
		if !ok {
			return fmt.Errorf("missing command parameter")
		}
		speed := 5
		if s, ok := params["speed"]; ok {
			if sp, err := strconv.Atoi(s); err == nil && sp > 0 {
				speed = sp
			}
		}
		return a.SendPTZCommand(dev, command, speed)
	default:
		return fmt.Errorf("unsupported command: %s", cmd)
	}
}

// Snapshot реализует метод интерфейса DeviceAdapter (по serial)
func (a *XiongmaiAdapter) Snapshot(serial string) ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, session := range a.devices {
		if session.device.Serial == serial {
			return a.GetSnapshot(session.device)
		}
	}
	return nil, fmt.Errorf("device with serial %s not found", serial)
}
