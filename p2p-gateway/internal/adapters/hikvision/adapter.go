package hikvision

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yourorg/p2p-gateway/internal/config"
)

type Adapter struct {
	cfg        *config.HikvisionConfig
	go2rtcCmd  *exec.Cmd
	httpClient *http.Client
	logger     *logrus.Logger
}

func NewAdapter(cfg *config.HikvisionConfig, logger *logrus.Logger) (*Adapter, error) {
	adapter := &Adapter{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
	// Запускаем go2rtc как подпроцесс, если бинарник существует
	if err := adapter.startGo2rtc(); err != nil {
		return nil, fmt.Errorf("failed to start go2rtc: %w", err)
	}
	return adapter, nil
}

func (a *Adapter) startGo2rtc() error {
	// go2rtc обычно слушает API на порту 1984. Запускаем с минимальной конфигурацией.
	a.go2rtcCmd = exec.Command(a.cfg.Go2rtcBinaryPath, "-api", ":"+strconv.Itoa(a.cfg.Go2rtcApiPort))
	if err := a.go2rtcCmd.Start(); err != nil {
		return err
	}
	// Даем время на инициализацию
	time.Sleep(2 * time.Second)
	a.logger.Info("go2rtc started as subprocess")
	return nil
}

// CheckDevice проверяет доступность устройства через go2rtc P2P Hik-Connect
func (a *Adapter) CheckDevice(serial, securityCode string) (bool, error) {
	// go2rtc имеет эндпоинт /api/streams для добавления источника.
	// Формат для Hikvision: "hikp2p://serial?code=securityCode"
	streamURL := fmt.Sprintf("hikp2p://%s?code=%s", serial, securityCode)
	reqBody := map[string]interface{}{
		"name":   serial,
		"source": streamURL,
	}
	jsonData, _ := json.Marshal(reqBody)
	resp, err := a.httpClient.Post(fmt.Sprintf("http://localhost:%d/api/streams", a.cfg.Go2rtcApiPort), "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	// Если успешно добавлен - возвращаем true
	return resp.StatusCode == http.StatusOK, nil
}

func (a *Adapter) GetStatus(serial, securityCode string) (string, error) {
	// Можно запросить информацию о потоке: GET /api/streams/{serial}
	resp, err := a.httpClient.Get(fmt.Sprintf("http://localhost:%d/api/streams/%s", a.cfg.Go2rtcApiPort, serial))
	if err != nil {
		return "offline", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return "online", nil
	}
	return "offline", nil
}

func (a *Adapter) GetSnapshot(serial, securityCode string) (string, error) {
	// Получить snapshot через go2rtc: /api/streams/{serial}/snapshot.jpg
	resp, err := a.httpClient.Get(fmt.Sprintf("http://localhost:%d/api/streams/%s/snapshot.jpg", a.cfg.Go2rtcApiPort, serial))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("snapshot not available")
	}
	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	base64Str := base64.StdEncoding.EncodeToString(imgData)
	return base64Str, nil
}

func (a *Adapter) SendPTZCommand(serial, securityCode, command string, params map[string]interface{}) error {
	// go2rtc может поддерживать PTZ через /api/streams/{serial}/ptz
	// Пример команды: {"action":"move","direction":"left","speed":0.5}
	ptzReq := map[string]interface{}{
		"action":    "move",
		"direction": command,
	}
	if speed, ok := params["speed"]; ok {
		ptzReq["speed"] = speed
	}
	jsonData, _ := json.Marshal(ptzReq)
	resp, err := a.httpClient.Post(fmt.Sprintf("http://localhost:%d/api/streams/%s/ptz", a.cfg.Go2rtcApiPort, serial), "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PTZ command failed")
	}
	return nil
}

func (a *Adapter) Stop() error {
	if a.go2rtcCmd != nil && a.go2rtcCmd.Process != nil {
		return a.go2rtcCmd.Process.Kill()
	}
	return nil
}
