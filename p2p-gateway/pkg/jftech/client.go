// pkg/jftech/client.go
package jftech

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Config struct {
	UUID      string `yaml:"uuid"`
	AppKey    string `yaml:"app_key"`
	AppSecret string `yaml:"app_secret"`
	MoveCard  int    `yaml:"move_card"`
	Endpoint  string `yaml:"endpoint"` // например, api-cn.jftechws.com
}

type Client struct {
	cfg        *Config
	httpClient *http.Client
}

func NewClient(cfg *Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) doRequest(method, path string, body interface{}, headers map[string]string) ([]byte, error) {
	url := fmt.Sprintf("https://%s/gwp/v3%s", c.cfg.Endpoint, path)
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// GetDeviceToken получает deviceToken для списка SN
func (c *Client) GetDeviceToken(sns []string, accessToken string) (map[string]string, error) {
	timeMillis, signature := GenerateSignature(c.cfg.UUID, c.cfg.AppKey, c.cfg.AppSecret, c.cfg.MoveCard)
	headers := map[string]string{
		"uuid":       c.cfg.UUID,
		"appKey":     c.cfg.AppKey,
		"timeMillis": timeMillis,
		"signature":  signature,
	}
	body := map[string]interface{}{
		"sns": sns,
	}
	if accessToken != "" {
		body["accessToken"] = accessToken
	}
	respBody, err := c.doRequest("POST", "/rtc/device/token", body, headers)
	if err != nil {
		return nil, err
	}
	var result struct {
		Code int `json:"code"`
		Data []struct {
			Sn    string `json:"sn"`
			Token string `json:"token"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	if result.Code != 2000 {
		return nil, fmt.Errorf("get device token failed: %s", result.Msg)
	}
	tokenMap := make(map[string]string)
	for _, item := range result.Data {
		tokenMap[item.Sn] = item.Token
	}
	return tokenMap, nil
}

// DeviceStatus запрашивает статус устройств по списку token'ов
func (c *Client) DeviceStatus(tokens []string, region string) ([]DeviceStatusResponse, error) {
	timeMillis, signature := GenerateSignature(c.cfg.UUID, c.cfg.AppKey, c.cfg.AppSecret, c.cfg.MoveCard)
	headers := map[string]string{
		"uuid":       c.cfg.UUID,
		"appKey":     c.cfg.AppKey,
		"timeMillis": timeMillis,
		"signature":  signature,
	}
	body := map[string]interface{}{
		"deviceTokenList": tokens,
		"region":          region,
	}
	respBody, err := c.doRequest("POST", "/rtc/device/status", body, headers)
	if err != nil {
		return nil, err
	}
	var result struct {
		Code int                    `json:"code"`
		Data []DeviceStatusResponse `json:"data"`
		Msg  string                 `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	if result.Code != 2000 {
		return nil, fmt.Errorf("device status failed: %s", result.Msg)
	}
	return result.Data, nil
}

type DeviceStatusResponse struct {
	Uuid          string `json:"uuid"`
	Status        string `json:"status"` // online/offline
	WakeUpStatus  string `json:"wakeUpStatus"`
	WakeUpEnable  string `json:"wakeUpEnable"`
	WanIp         string `json:"wanIp"`
	LastHeartbeat string `json:"lastHeartbeat"`
}

// Livestream получает RTSP-URL
func (c *Client) GetLivestream(deviceToken, channel, stream, protocol, username, password string, expireTime *int64) (string, error) {
	timeMillis, signature := GenerateSignature(c.cfg.UUID, c.cfg.AppKey, c.cfg.AppSecret, c.cfg.MoveCard)
	headers := map[string]string{
		"uuid":       c.cfg.UUID,
		"appKey":     c.cfg.AppKey,
		"timeMillis": timeMillis,
		"signature":  signature,
	}
	body := map[string]interface{}{
		"channel":  channel,
		"stream":   stream,
		"protocol": protocol,
		"username": username,
		"password": password,
	}
	if expireTime != nil {
		body["expireTime"] = fmt.Sprintf("%d", *expireTime)
	}
	respBody, err := c.doRequest("POST", fmt.Sprintf("/rtc/device/livestream/%s", deviceToken), body, headers)
	if err != nil {
		return "", err
	}
	var result struct {
		Code int `json:"code"`
		Data struct {
			Ret    int    `json:"Ret"`
			Url    string `json:"url"`
			RetMsg string `json:"retMsg"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if result.Code != 2000 {
		return "", fmt.Errorf("livestream failed: %s", result.Msg)
	}
	if result.Data.Ret != 100 {
		return "", fmt.Errorf("device returned error: %s", result.Data.RetMsg)
	}
	return result.Data.Url, nil
}

// Capture получает снимок
func (c *Client) Capture(deviceToken string, channel, picType int) (string, error) {
	timeMillis, signature := GenerateSignature(c.cfg.UUID, c.cfg.AppKey, c.cfg.AppSecret, c.cfg.MoveCard)
	headers := map[string]string{
		"uuid":       c.cfg.UUID,
		"appKey":     c.cfg.AppKey,
		"timeMillis": timeMillis,
		"signature":  signature,
	}
	body := map[string]interface{}{
		"Name": "OPSNAP",
		"OPSNAP": map[string]interface{}{
			"Channel": channel,
			"PicType": picType,
		},
	}
	respBody, err := c.doRequest("POST", fmt.Sprintf("/rtc/device/capture/%s", deviceToken), body, headers)
	if err != nil {
		return "", err
	}
	var result struct {
		Code int `json:"code"`
		Data struct {
			Ret   int    `json:"Ret"`
			Image string `json:"image"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if result.Code != 2000 {
		return "", fmt.Errorf("capture failed: %s", result.Msg)
	}
	if result.Data.Ret != 100 {
		return "", fmt.Errorf("device capture error: Ret=%d", result.Data.Ret)
	}
	return result.Data.Image, nil
}

// PTZControl отправляет команду PTZ
func (c *Client) PTZControl(deviceToken string, command string, channel, preset, step int) error {
	timeMillis, signature := GenerateSignature(c.cfg.UUID, c.cfg.AppKey, c.cfg.AppSecret, c.cfg.MoveCard)
	headers := map[string]string{
		"uuid":       c.cfg.UUID,
		"appKey":     c.cfg.AppKey,
		"timeMillis": timeMillis,
		"signature":  signature,
	}
	body := map[string]interface{}{
		"Name": "OPPTZControl",
		"OPPTZControl": map[string]interface{}{
			"Command": command,
			"Parameter": map[string]interface{}{
				"Preset":  preset,
				"Channel": channel,
				"Step":    step,
			},
		},
	}
	respBody, err := c.doRequest("POST", fmt.Sprintf("/rtc/device/opdev/%s", deviceToken), body, headers)
	if err != nil {
		return err
	}
	var result struct {
		Code int `json:"code"`
		Data struct {
			Ret int `json:"Ret"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}
	if result.Code != 2000 {
		return fmt.Errorf("ptz control failed: %s", result.Msg)
	}
	if result.Data.Ret != 100 {
		return fmt.Errorf("device ptz error: Ret=%d", result.Data.Ret)
	}
	return nil
}

// Добавить в конец файла pkg/jftech/client.go:

// GetDeviceLogs получает логи устройства через OPLogQuery
func (c *Client) GetDeviceLogs(deviceToken, startTime, endTime, logType string) ([]byte, error) {
	timeMillis, signature := GenerateSignature(c.cfg.UUID, c.cfg.AppKey, c.cfg.AppSecret, c.cfg.MoveCard)
	headers := map[string]string{
		"uuid":       c.cfg.UUID,
		"appKey":     c.cfg.AppKey,
		"timeMillis": timeMillis,
		"signature":  signature,
	}
	if logType == "" {
		logType = "All"
	}
	body := map[string]interface{}{
		"Name": "OPLogQuery",
		"OPLogQuery": map[string]interface{}{
			"BeginTime":   startTime,
			"EndTime":     endTime,
			"LogPosition": "0",
			"Type":        logType,
		},
	}
	return c.doRequest("POST", fmt.Sprintf("/rtc/device/opdev/%s", deviceToken), body, headers)
}
