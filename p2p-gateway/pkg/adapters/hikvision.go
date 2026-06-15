package adapters

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"p2p-gateway/internal/models"
	"p2p-gateway/pkg/hikp2p"
)

type HikvisionAdapter struct {
	sessions  map[string]*hikp2p.P2PSession
	ffmpegCmd map[string]*exec.Cmd
}

func NewHikvisionAdapter() *HikvisionAdapter {
	return &HikvisionAdapter{
		sessions:  make(map[string]*hikp2p.P2PSession),
		ffmpegCmd: make(map[string]*exec.Cmd),
	}
}

// extractUserID парсит JWT и возвращает значение поля "aud"
func extractUserID(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	// Декодируем payload (второй сегмент) из Base64URL
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return ""
	}
	if aud, ok := claims["aud"].(string); ok {
		return aud
	}
	return ""
}

func (a *HikvisionAdapter) Start(dev *models.Device) error {
	client := hikp2p.NewClient("")
	sess, err := client.Login(hikp2p.Credentials{Username: dev.Username, Password: dev.Password})
	if err != nil {
		return err
	}
	secret, err := client.GetP2PSecret(dev.Serial)
	if err != nil {
		return err
	}
	devInfo, err := client.GetDeviceInfo(dev.Serial)
	if err != nil {
		return err
	}
	cfg := hikp2p.P2PConfig{
		DeviceSerial:  dev.Serial,
		DeviceIP:      devInfo.IP,
		DevicePort:    devInfo.Port,
		P2PServers:    secret.Servers,
		P2PKey:        secret.Key,
		P2PLinkKey:    secret.Key,
		P2PKeyVersion: 101,
		P2PKeySaltIdx: secret.SaltIndex,
		P2PKeySaltVer: secret.SaltVer,
		SessionToken:  sess.SessionID,
		UserID:        extractUserID(sess.SessionID),
		ClientID:      0x0aed13f5,
		ChannelNo:     1,
		StreamType:    1,
	}
	p2p, err := hikp2p.NewP2PSession(cfg)
	if err != nil {
		return err
	}
	if err := p2p.Start(); err != nil {
		return err
	}
	a.sessions[dev.ID] = p2p

	// Запуск FFmpeg
	rtspURL := fmt.Sprintf("rtsp://127.0.0.1:%d/stream", dev.ProxyPort)
	ffmpeg := exec.Command("ffmpeg",
		"-f", "hevc",
		"-i", "pipe:0",
		"-c", "copy",
		"-f", "rtsp",
		"-rtsp_transport", "tcp",
		rtspURL,
	)
	stdin, err := ffmpeg.StdinPipe()
	if err != nil {
		p2p.Stop()
		return err
	}
	go func() {
		defer stdin.Close()
		io.Copy(stdin, p2p.VideoReader())
	}()
	if err := ffmpeg.Start(); err != nil {
		p2p.Stop()
		return err
	}
	a.ffmpegCmd[dev.ID] = ffmpeg

	dev.RTSPURL = rtspURL
	dev.Status = models.StatusOnline
	return nil
}

func (a *HikvisionAdapter) Stop(dev *models.Device) error {
	if sess, ok := a.sessions[dev.ID]; ok {
		sess.Stop()
		delete(a.sessions, dev.ID)
	}
	if cmd, ok := a.ffmpegCmd[dev.ID]; ok {
		cmd.Process.Kill()
		delete(a.ffmpegCmd, dev.ID)
	}
	return nil
}

func (a *HikvisionAdapter) GetStatus(dev *models.Device) (models.DeviceStatus, error) {
	if _, ok := a.sessions[dev.ID]; ok {
		return models.StatusOnline, nil
	}
	return models.StatusOffline, nil
}

func (a *HikvisionAdapter) Command(dev *models.Device, cmd string, params map[string]string) error {
	// PTZ через go2rtc не реализован – заглушка
	return nil
}

func (a *HikvisionAdapter) Snapshot(serial string) ([]byte, error) {
	// TODO: извлечь первый ключевой кадр из потока
	return nil, fmt.Errorf("snapshot not implemented")
}
