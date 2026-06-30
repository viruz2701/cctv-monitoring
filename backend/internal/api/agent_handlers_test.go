// Package api — unit tests for Agent Management HTTP handlers.
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone 3 (Backend)
//   - OWASP ASVS V5.1: Input validation
//   - OWASP ASVS V3.3: RBAC
//   - OWASP ASVS V7.1: Error handling
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// ── Mock AgentStore ──────────────────────────────────────────────────────

type mockAgentStore struct {
	listAgentsFunc  func() ([]Agent, error)
	getAgentFunc    func(id string) (*Agent, error)
	deleteAgentFunc func(id string) error
	sendCommandFunc func(id string, command string, params map[string]interface{}) error
}

func (m *mockAgentStore) ListAgents() ([]Agent, error) {
	if m.listAgentsFunc != nil {
		return m.listAgentsFunc()
	}
	return []Agent{}, nil
}

func (m *mockAgentStore) GetAgent(id string) (*Agent, error) {
	if m.getAgentFunc != nil {
		return m.getAgentFunc(id)
	}
	return nil, errors.New("agent not found")
}

func (m *mockAgentStore) DeleteAgent(id string) error {
	if m.deleteAgentFunc != nil {
		return m.deleteAgentFunc(id)
	}
	return nil
}

func (m *mockAgentStore) SendCommand(id string, command string, params map[string]interface{}) error {
	if m.sendCommandFunc != nil {
		return m.sendCommandFunc(id, command, params)
	}
	return nil
}

// ── Tests: validateAgentCommand ──────────────────────────────────────────

func TestValidateAgentCommand_Valid(t *testing.T) {
	validCommands := []string{"restart", "update", "reboot", "diagnose", "sync_config", "ping"}
	for _, cmd := range validCommands {
		req := &agentCommandRequest{Command: cmd}
		err := validateAgentCommand(req)
		if err != nil {
			t.Errorf("expected no error for command %q, got %v", cmd, err)
		}
	}
}

func TestValidateAgentCommand_Invalid(t *testing.T) {
	req := &agentCommandRequest{Command: "unknown_command"}
	err := validateAgentCommand(req)
	if err == nil {
		t.Error("expected error for invalid command")
	}
}

func TestValidateAgentCommand_Empty(t *testing.T) {
	req := &agentCommandRequest{Command: ""}
	err := validateAgentCommand(req)
	if err == nil {
		t.Error("expected error for empty command")
	}
}

// ── Tests: validAgentCommands ────────────────────────────────────────────

func TestValidAgentCommands_AllExpected(t *testing.T) {
	expected := []string{"restart", "update", "reboot", "diagnose", "sync_config", "ping"}
	for _, cmd := range expected {
		if !validAgentCommands[cmd] {
			t.Errorf("expected command %q to be valid", cmd)
		}
	}
}

func TestValidAgentCommands_NoUnexpected(t *testing.T) {
	unexpected := []string{"", "exec", "rm", "shutdown", "reset", "shell", "bash"}
	for _, cmd := range unexpected {
		if validAgentCommands[cmd] {
			t.Errorf("unexpected command %q found in whitelist", cmd)
		}
	}
}

// ── Tests: handleListAgents ──────────────────────────────────────────────

func TestHandleListAgents_Success(t *testing.T) {
	now := time.Now()
	s := &Server{
		agentStore: &mockAgentStore{
			listAgentsFunc: func() ([]Agent, error) {
				return []Agent{
					{ID: "agent-1", Name: "Edge-Agent-01", Status: "online", LastSeen: &now, Version: "v2.5.1", CreatedAt: now},
					{ID: "agent-2", Name: "Edge-Agent-02", Status: "offline", Version: "v2.4.0", CreatedAt: now},
				}, nil
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/v1/agents", s.handleListAgents)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	agents, ok := resp["agents"].([]interface{})
	if !ok {
		t.Fatal("expected agents array in response")
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
	if resp["total"] != float64(2) {
		t.Errorf("expected total 2, got %v", resp["total"])
	}
}

func TestHandleListAgents_NoStore(t *testing.T) {
	s := &Server{}

	r := chi.NewRouter()
	r.Get("/api/v1/agents", s.handleListAgents)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestHandleListAgents_StoreError(t *testing.T) {
	s := &Server{
		agentStore: &mockAgentStore{
			listAgentsFunc: func() ([]Agent, error) {
				return nil, errors.New("database error")
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/v1/agents", s.handleListAgents)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestHandleListAgents_EmptyList(t *testing.T) {
	s := &Server{
		agentStore: &mockAgentStore{
			listAgentsFunc: func() ([]Agent, error) {
				return []Agent{}, nil
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/v1/agents", s.handleListAgents)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	agents := resp["agents"].([]interface{})
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}
	if resp["total"] != float64(0) {
		t.Errorf("expected total 0, got %v", resp["total"])
	}
}

// ── Tests: handleGetAgent ────────────────────────────────────────────────

func TestHandleGetAgent_Success(t *testing.T) {
	now := time.Now()
	s := &Server{
		agentStore: &mockAgentStore{
			getAgentFunc: func(id string) (*Agent, error) {
				return &Agent{
					ID:        id,
					Name:      "Edge-Agent-01",
					Status:    "online",
					Version:   "v2.5.1",
					LastSeen:  &now,
					CreatedAt: now,
				}, nil
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/v1/agents/{id}", s.handleGetAgent)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/agent-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var agent Agent
	if err := json.NewDecoder(w.Body).Decode(&agent); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if agent.ID != "agent-1" {
		t.Errorf("expected ID 'agent-1', got %q", agent.ID)
	}
	if agent.Name != "Edge-Agent-01" {
		t.Errorf("expected Name 'Edge-Agent-01', got %q", agent.Name)
	}
}

func TestHandleGetAgent_NotFound(t *testing.T) {
	s := &Server{
		agentStore: &mockAgentStore{
			getAgentFunc: func(id string) (*Agent, error) {
				return nil, errors.New("not found")
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/v1/agents/{id}", s.handleGetAgent)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleGetAgent_MissingID(t *testing.T) {
	s := &Server{}

	r := chi.NewRouter()
	r.Get("/api/v1/agents/{id}", s.handleGetAgent)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Chi router returns 404 for empty {id} param - the handler never runs
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 (chi route mismatch), got %d", w.Code)
	}
}

// ── Tests: sanitizeAgent ─────────────────────────────────────────────────

func TestSanitizeAgent_NoConfig(t *testing.T) {
	agent := Agent{ID: "agent-1", Name: "Test Agent", Status: "online"}
	sanitized := sanitizeAgent(agent)

	if sanitized.Config != nil {
		t.Error("expected nil config when agent has no config")
	}
}

func TestSanitizeAgent_MasksSensitiveKeys(t *testing.T) {
	agent := Agent{
		ID:     "agent-1",
		Name:   "Test Agent",
		Status: "online",
		Config: map[string]interface{}{
			"log_level": "info",
			"password":  "supersecret",
			"api_key":   "abc123",
			"interval":  float64(5000),
		},
	}

	sanitized := sanitizeAgent(agent)

	if sanitized.Config["log_level"] != "info" {
		t.Errorf("expected log_level to remain unchanged, got %v", sanitized.Config["log_level"])
	}
	if sanitized.Config["password"] != "****" {
		t.Errorf("expected password to be masked, got %v", sanitized.Config["password"])
	}
	if sanitized.Config["api_key"] != "****" {
		t.Errorf("expected api_key to be masked, got %v", sanitized.Config["api_key"])
	}
	if sanitized.Config["interval"] != float64(5000) {
		t.Errorf("expected interval to remain unchanged, got %v", sanitized.Config["interval"])
	}
}

func TestSanitizeAgent_NoSensitiveKeys(t *testing.T) {
	agent := Agent{
		ID:     "agent-1",
		Name:   "Test Agent",
		Status: "online",
		Config: map[string]interface{}{
			"log_level": "info",
			"interval":  float64(5000),
		},
	}

	sanitized := sanitizeAgent(agent)

	if sanitized.Config["log_level"] != "info" {
		t.Errorf("expected log_level to remain 'info', got %v", sanitized.Config["log_level"])
	}
	if len(sanitized.Config) != 2 {
		t.Errorf("expected 2 config keys, got %d", len(sanitized.Config))
	}
}
