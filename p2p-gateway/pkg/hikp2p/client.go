package hikp2p

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	DefaultBaseURL = "https://api.hik-connect.com"
	ClientType     = "55"
	FeatureCode    = "deadbeef"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	session    *Session
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Login(creds Credentials) (*Session, error) {
	body := fmt.Sprintf("account=%s&password=%x&featureCode=%s",
		creds.Username, md5.Sum([]byte(creds.Password)), FeatureCode)
	req, err := http.NewRequest("POST", c.baseURL+"/v3/users/login/v2", bytes.NewBufferString(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("clientType", ClientType)
	req.Header.Set("featureCode", FeatureCode)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Meta struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"meta"`
		LoginSession struct {
			SessionID   string `json:"sessionId"`
			RfSessionID string `json:"rfSessionId"`
		} `json:"loginSession"`
		LoginArea struct {
			ApiDomain string `json:"apiDomain"`
		} `json:"loginArea"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Meta.Code != 200 {
		return nil, fmt.Errorf("login failed: %s", result.Meta.Message)
	}
	c.session = &Session{
		SessionID:    result.LoginSession.SessionID,
		RefreshToken: result.LoginSession.RfSessionID,
		ApiDomain:    result.LoginArea.ApiDomain,
		ExpiresAt:    time.Now().Add(30 * time.Minute),
	}
	return c.session, nil
}

func (c *Client) GetP2PSecret(deviceSerial string) (*P2PSecret, error) {
	url := fmt.Sprintf("%s/api/p2p/configurations", c.baseURL)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("sessionId", c.session.SessionID)
	req.Header.Set("clientType", ClientType)
	req.Header.Set("featureCode", FeatureCode)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		ServerInfos []struct {
			Ip   string `json:"ip"`
			Port int    `json:"port"`
		} `json:"serverInfos"`
		Secret struct {
			Version    int    `json:"version"`
			SaltIndex  int    `json:"saltIndex"`
			ExpireTime int64  `json:"expireTime"`
			Data       string `json:"data"` // "[b0,b1,...,b31]"
		} `json:"secret"`
		ResultCode string `json:"resultCode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if raw.ResultCode != "0" {
		return nil, fmt.Errorf("get secret failed: %s", raw.ResultCode)
	}
	// парсим data в 32 байта
	var bytes []int
	if err := json.Unmarshal([]byte(raw.Secret.Data), &bytes); err != nil {
		return nil, err
	}
	if len(bytes) != 32 {
		return nil, fmt.Errorf("invalid secret length: %d", len(bytes))
	}
	key := make([]byte, 32)
	for i, b := range bytes {
		key[i] = byte(b)
	}
	servers := make([]P2PServer, len(raw.ServerInfos))
	for i, s := range raw.ServerInfos {
		servers[i] = P2PServer{Host: s.Ip, Port: s.Port}
	}
	return &P2PSecret{
		Key:       key,
		SaltIndex: raw.Secret.SaltIndex,
		SaltVer:   raw.Secret.Version,
		Servers:   servers,
	}, nil
}

func (c *Client) GetDeviceInfo(deviceSerial string) (*Device, error) {
	url := fmt.Sprintf("%s/v3/userdevices/v1/resources/pagelist?groupId=-1&limit=50&offset=0&filter=CONNECTION,P2P", c.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("sessionId", c.session.SessionID)
	req.Header.Set("clientType", ClientType)
	req.Header.Set("featureCode", FeatureCode)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Meta struct {
			Code int `json:"code"`
		} `json:"meta"`
		Connection map[string]struct {
			NetIp         string `json:"netIp"`
			NetStreamPort int    `json:"netStreamPort"`
			LocalIp       string `json:"localIp"`
			LocalCmdPort  int    `json:"localCmdPort"`
		} `json:"CONNECTION"`
		P2P map[string][]struct {
			Ip   string `json:"ip"`
			Port int    `json:"port"`
		} `json:"P2P"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Meta.Code != 200 {
		return nil, fmt.Errorf("get device info failed: code=%d", data.Meta.Code)
	}
	conn, ok := data.Connection[deviceSerial]
	if !ok {
		return nil, fmt.Errorf("device %s not found", deviceSerial)
	}
	dev := &Device{
		Serial:    deviceSerial,
		IP:        conn.NetIp,
		Port:      conn.NetStreamPort,
		LocalIP:   conn.LocalIp,
		LocalPort: conn.LocalCmdPort,
	}
	return dev, nil
}
