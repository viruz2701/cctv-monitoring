package protocols

import (
    "context"
    "fmt"
    "gb-telemetry-collector/internal/models"
    "gb-telemetry-collector/internal/state"
    "io"
    "log/slog"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"

    ftp "goftp.io/server/v2"
)

type FTPHandler struct {
    port      int
    rootPath  string
    user      string
    password  string
    stateMgr  state.DeviceStateManager
    logger    *slog.Logger
    server    *ftp.Server
    wg        sync.WaitGroup
    ctx       context.Context
    cancel    context.CancelFunc
}

func NewFTPHandler(port int, rootPath, user, password string, stateMgr state.DeviceStateManager, logger *slog.Logger) *FTPHandler {
    return &FTPHandler{
        port:     port,
        rootPath: rootPath,
        user:     user,
        password: password,
        stateMgr: stateMgr,
        logger:   logger,
    }
}

func (h *FTPHandler) Start(ctx context.Context) error {
    h.ctx, h.cancel = context.WithCancel(ctx)

    if err := os.MkdirAll(h.rootPath, 0755); err != nil {
        return fmt.Errorf("failed to create FTP root dir: %w", err)
    }

    driver := &FTPDriver{
        stateMgr: h.stateMgr,
        rootPath: h.rootPath,
        logger:   h.logger,
    }

    opts := &ftp.Options{
        Name:           "gb-telemetry-ftp",
        Hostname:       "0.0.0.0",
        Port:           h.port,
        Driver:         driver,
        Auth:           &FTPAuth{password: h.password, logger: h.logger},
        Perm:           ftp.NewSimplePerm("root", "root"),
        WelcomeMessage: "Welcome to GB Telemetry FTP Server",
    }

    var err error
    h.server, err = ftp.NewServer(opts)
    if err != nil {
        return fmt.Errorf("failed to create FTP server: %w", err)
    }

    h.wg.Add(1)
    go func() {
        defer h.wg.Done()
        if err := h.server.ListenAndServe(); err != nil {
            h.logger.Error("FTP server error", "error", err)
        }
    }()

    h.logger.Info("FTP server started", "port", h.port, "root", h.rootPath)
    return nil
}

func (h *FTPHandler) Stop() error {
    h.cancel()
    if h.server != nil {
        if err := h.server.Shutdown(); err != nil {
            h.logger.Error("FTP shutdown error", "error", err)
        }
    }
    h.wg.Wait()
    return nil
}

type FTPDriver struct {
    stateMgr state.DeviceStateManager
    rootPath string
    logger   *slog.Logger
}

// Реализация интерфейса ftp.Driver
func (d *FTPDriver) Stat(ctx *ftp.Context, path string) (os.FileInfo, error) {
    fullPath := filepath.Join(d.rootPath, path)
    d.logger.Debug("FTP Stat", "path", path, "full", fullPath)
    return os.Stat(fullPath)
}

func (d *FTPDriver) ListDir(ctx *ftp.Context, path string, callback func(os.FileInfo) error) error {
    fullPath := filepath.Join(d.rootPath, path)
    d.logger.Debug("FTP ListDir", "path", path, "full", fullPath)
    entries, err := os.ReadDir(fullPath)
    if err != nil {
        return err
    }
    for _, entry := range entries {
        info, err := entry.Info()
        if err != nil {
            continue
        }
        if err := callback(info); err != nil {
            return err
        }
    }
    return nil
}

func (d *FTPDriver) DeleteDir(ctx *ftp.Context, path string) error {
    fullPath := filepath.Join(d.rootPath, path)
    d.logger.Debug("FTP DeleteDir", "path", path, "full", fullPath)
    return os.RemoveAll(fullPath)
}

func (d *FTPDriver) DeleteFile(ctx *ftp.Context, path string) error {
    fullPath := filepath.Join(d.rootPath, path)
    d.logger.Debug("FTP DeleteFile", "path", path, "full", fullPath)
    return os.Remove(fullPath)
}

func (d *FTPDriver) Rename(ctx *ftp.Context, from, to string) error {
    fromPath := filepath.Join(d.rootPath, from)
    toPath := filepath.Join(d.rootPath, to)
    d.logger.Debug("FTP Rename", "from", from, "to", to, "fromFull", fromPath, "toFull", toPath)
    return os.Rename(fromPath, toPath)
}

func (d *FTPDriver) MakeDir(ctx *ftp.Context, path string) error {
    fullPath := filepath.Join(d.rootPath, path)
    d.logger.Debug("FTP MakeDir", "path", path, "full", fullPath)
    return os.MkdirAll(fullPath, 0755)
}

func (d *FTPDriver) GetFile(ctx *ftp.Context, path string, offset int64) (int64, io.ReadCloser, error) {
    fullPath := filepath.Join(d.rootPath, path)
    d.logger.Debug("FTP GetFile", "path", path, "full", fullPath, "offset", offset)
    f, err := os.Open(fullPath)
    if err != nil {
        return 0, nil, err
    }
    info, err := f.Stat()
    if err != nil {
        f.Close()
        return 0, nil, err
    }
    if offset > 0 {
        if _, err := f.Seek(offset, 0); err != nil {
            f.Close()
            return 0, nil, err
        }
    }
    return info.Size() - offset, f, nil
}

func (d *FTPDriver) PutFile(ctx *ftp.Context, destPath string, data io.Reader, offset int64) (int64, error) {
    fullPath := filepath.Join(d.rootPath, destPath)
    d.logger.Debug("FTP PutFile", "destPath", destPath, "fullPath", fullPath, "offset", offset)

    // Создаём директорию, если нужно
    dir := filepath.Dir(fullPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        d.logger.Error("FTP PutFile: MkdirAll failed", "dir", dir, "error", err)
        return 0, err
    }

    var file *os.File
    var err error
    if offset == 0 {
        file, err = os.Create(fullPath)
    } else {
        file, err = os.OpenFile(fullPath, os.O_APPEND|os.O_WRONLY, 0644)
    }
    if err != nil {
        d.logger.Error("FTP PutFile: open file error", "path", fullPath, "error", err)
        return 0, err
    }
    defer file.Close()

    n, err := io.Copy(file, data)
    if err != nil {
        d.logger.Error("FTP PutFile: copy error", "error", err)
        return n, err
    }

    // Асинхронно генерируем событие
    go func() {
        // Извлекаем имя камеры из пути: обычно первый сегмент пути или имя файла без расширения
        parts := strings.Split(destPath, "/")
        cameraName := "ftp_unknown"
        if len(parts) > 0 && parts[0] != "" {
            cameraName = parts[0]
        }
        deviceID := fmt.Sprintf("ftp_%s", cameraName)

        if _, ok := d.stateMgr.Get(deviceID); !ok {
            dev := &models.Device{
                DeviceID:     deviceID,
                Status:       models.StatusOnline,
                LastSeen:     time.Now(),
                RegisteredAt: time.Now(),
                VendorType:   "ftp",
                Name:         cameraName,
                Location:     "",
            }
            d.stateMgr.Set(dev)
        } else {
            d.stateMgr.UpdateLastSeen(deviceID)
        }

        alarm := &models.Alarm{
            DeviceID:    deviceID,
            Priority:    models.AlarmPriorityLow,
            Method:      models.AlarmMethodMotionDetection,
            Timestamp:   time.Now(),
            Description: fmt.Sprintf("FTP file uploaded: %s", destPath),
        }
        d.stateMgr.AddAlarm(deviceID, alarm)
        d.logger.Info("FTP upload", "device", deviceID, "file", destPath)
    }()

    return n, nil
}

type FTPAuth struct {
    password string
    logger   *slog.Logger
}

func (a *FTPAuth) CheckPasswd(ctx *ftp.Context, user, pass string) (bool, error) {
    ok := pass == a.password
    if !ok && a.logger != nil {
        a.logger.Warn("FTP auth failed", "user", user)
    }
    return ok, nil
}