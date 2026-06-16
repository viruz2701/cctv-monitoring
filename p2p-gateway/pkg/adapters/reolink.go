package adapters

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"p2p-gateway/internal/models"
)

// NeolinkAdapter управляет процессом neolink.
type NeolinkAdapter struct {
	binPath   string
	mu        sync.Mutex
	processes map[string]*neolinkInstance
}

type neolinkInstance struct {
	cmd        *exec.Cmd
	rtspPort   int
	serial     string
	configFile string
	httpClient *http.Client
}

// NewReolinkAdapter создаёт адаптер для Reolink (использует neolink).
func NewReolinkAdapter(binPath string) *NeolinkAdapter {
	return &NeolinkAdapter{
		binPath:   binPath,
		processes: make(map[string]*neolinkInstance),
	}
}

// Start запускает neolink для устройства.
func (a *NeolinkAdapter) Start(dev *models.Device) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.processes[dev.ID]; exists {
		return fmt.Errorf("device %s already started", dev.ID)
	}

	if _, err := os.Stat(a.binPath); os.IsNotExist(err) {
		return fmt.Errorf("neolink binary not found at %s", a.binPath)
	}

	// Выделяем порт для RTSP
	rtspPort := dev.ProxyPort

	// Создаём временный конфигурационный файл neolink
	configDir, err := os.MkdirTemp("", "neolink_")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	configFile := filepath.Join(configDir, "config.toml")

	// Имя камеры в neolink должно быть без пробелов и слешей
	camName := fmt.Sprintf("%s_%s", dev.Brand, dev.Serial)

	// Формируем конфигурацию (TOML)
	// discovery = "relay" – использование облачного ретранслятора Reolink
	configContent := fmt.Sprintf(`
bind = "0.0.0.0"
bind_port = %d

[[cameras]]
name = "%s"
uid = "%s"
username = "%s"
password = "%s"
discovery = "relay"
`, rtspPort, camName, dev.Serial, dev.Username, dev.Password)

	if err := os.WriteFile(configFile, []byte(configContent), 0600); err != nil {
		os.RemoveAll(configDir)
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Команда: neolink rtsp --config <файл>
	cmd := exec.Command(a.binPath, "rtsp", "--config", configFile)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		os.RemoveAll(configDir)
		return fmt.Errorf("failed to start neolink: %w", err)
	}

	inst := &neolinkInstance{
		cmd:        cmd,
		rtspPort:   rtspPort,
		serial:     dev.Serial,
		configFile: configFile,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	a.processes[dev.ID] = inst

	// RTSP URL, который предоставляет neolink: rtsp://IP:порт/имя_камеры
	// Бэкенд будет использовать этот URL напрямую.
	dev.RTSPURL = fmt.Sprintf("rtsp://127.0.0.1:%d/%s", rtspPort, camName)
	dev.Status = models.StatusOnline
	dev.LastSeen = time.Now().Format(time.RFC3339)

	log.Printf("Neolink device %s started: RTSP port %d, URL=%s", dev.ID, rtspPort, dev.RTSPURL)

	go a.monitorProcess(dev, inst)

	return nil
}

// Stop останавливает процесс neolink.
func (a *NeolinkAdapter) Stop(dev *models.Device) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	inst, exists := a.processes[dev.ID]
	if !exists {
		return fmt.Errorf("device %s not running", dev.ID)
	}

	// Убить процесс
	if err := inst.cmd.Process.Kill(); err != nil {
		log.Printf("WARN: failed to kill neolink for %s: %v", dev.ID, err)
	}
	_ = inst.cmd.Wait()

	// Удалить временную конфигурацию
	_ = os.RemoveAll(filepath.Dir(inst.configFile))

	delete(a.processes, dev.ID)
	dev.Status = models.StatusOffline
	dev.RTSPURL = ""

	log.Printf("Neolink device %s stopped", dev.ID)
	return nil
}

// GetStatus проверяет, жив ли процесс и доступен ли RTSP-порт.
func (a *NeolinkAdapter) GetStatus(dev *models.Device) (models.DeviceStatus, error) {
	a.mu.Lock()
	inst, exists := a.processes[dev.ID]
	a.mu.Unlock()

	if !exists {
		return models.StatusOffline, nil
	}

	if inst.cmd.ProcessState != nil && inst.cmd.ProcessState.Exited() {
		return models.StatusOffline, nil
	}

	// Проверка доступности RTSP-порта
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", inst.rtspPort), 2*time.Second)
	if err != nil {
		return models.StatusOffline, nil
	}
	conn.Close()

	return models.StatusOnline, nil
}

// Command отправляет PTZ-команду через neolink (не реализовано, но можно расширить).
func (a *NeolinkAdapter) Command(dev *models.Device, cmd string, params map[string]string) error {
	// Neolink не имеет прямого HTTP API для PTZ, но может быть расширен через MQTT.
	// Пока заглушка.
	log.Printf("Command %s for %s not implemented in neolink adapter (stub)", cmd, dev.ID)
	return nil
}

// Snapshot возвращает снимок (заглушка, neolink не поддерживает /api/snapshot).
func (a *NeolinkAdapter) Snapshot(serial string) ([]byte, error) {
	// Возвращаем пустой срез, можно доработать через RTSP-канал.
	log.Printf("Snapshot requested for serial %s, not implemented", serial)
	return []byte{}, nil
}

// monitorProcess следит за неожиданным завершением процесса.
func (a *NeolinkAdapter) monitorProcess(dev *models.Device, inst *neolinkInstance) {
	err := inst.cmd.Wait()
	log.Printf("Neolink device %s process exited: %v", dev.ID, err)

	a.mu.Lock()
	defer a.mu.Unlock()
	if current, exists := a.processes[dev.ID]; exists && current == inst {
		delete(a.processes, dev.ID)
		dev.Status = models.StatusOffline
		dev.RTSPURL = ""
		log.Printf("Neolink device %s became offline", dev.ID)
	}
}
