package alarm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type AlarmEvent struct {
	DeviceID    string `json:"device_id"`
	EventType   string `json:"event_type"`
	Priority    int    `json:"priority"`
	Method      int    `json:"method"`
	Description string `json:"description"`
	Timestamp   string `json:"timestamp"`
	ImageBase64 string `json:"image_base64,omitempty"`
}

type Sender struct {
	collectorURL string
	apiKey       string
	httpClient   *http.Client
}

func NewSender(collectorURL, apiKey string) *Sender {
	return &Sender{
		collectorURL: collectorURL,
		apiKey:       apiKey,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Sender) SendAlarm(alarm *AlarmEvent) error {
	url := fmt.Sprintf("%s/api/v1/external/alarm/p2p", s.collectorURL)
	jsonData, err := json.Marshal(alarm)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("alarm rejected with status %d", resp.StatusCode)
	}
	return nil
}
