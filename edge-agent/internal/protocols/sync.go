package protocols

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/icholy/digest"
)

// ProtocolSync synchronizes protocol descriptors with the Backend API.
// Uses digest authentication for API access.
//
// Compliance: IEC 62443-3-3 SL-3 — синхронизация через DMZ (Zone 2)
type ProtocolSync struct {
	backendURL string
	user       string
	timeout    time.Duration
	client     *http.Client
	logger     *slog.Logger
}

// NewProtocolSync creates a new protocol sync client.
func NewProtocolSync(backendURL, user string, timeout time.Duration, logger *slog.Logger) *ProtocolSync {
	return &ProtocolSync{
		backendURL: backendURL,
		user:       user,
		timeout:    timeout,
		client: &http.Client{
			Timeout: timeout,
			Transport: &digest.Transport{
				Username: user,
				Password: os.Getenv("EDGE_AGENT_BACKEND_PASSWORD"),
			},
		},
		logger: logger,
	}
}

// Sync fetches protocol descriptors from the Backend API.
func (s *ProtocolSync) Sync(ctx context.Context) ([]Descriptor, error) {
	url := fmt.Sprintf("%s/api/v1/protocols/descriptors", s.backendURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "edge-agent/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var response struct {
		Descriptors []Descriptor `json:"descriptors"`
		Count       int          `json:"count"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	s.logger.Info("protocol descriptors synced",
		"count", response.Count,
	)

	return response.Descriptors, nil
}

// SyncOne fetches a single protocol descriptor by vendor and model.
func (s *ProtocolSync) SyncOne(ctx context.Context, vendor, model string) (*Descriptor, error) {
	url := fmt.Sprintf("%s/api/v1/protocols/descriptors/%s/%s",
		s.backendURL, vendor, model)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "edge-agent/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var descriptor Descriptor
	if err := json.Unmarshal(body, &descriptor); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &descriptor, nil
}

// ReportDevice sends discovered device info to Backend.
func (s *ProtocolSync) ReportDevice(ctx context.Context, vendor, model, mac, ip string) error {
	url := fmt.Sprintf("%s/api/v1/devices/report", s.backendURL)

	payload := map[string]string{
		"vendor": vendor,
		"model":  model,
		"mac":    mac,
		"ip":     ip,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "edge-agent/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
