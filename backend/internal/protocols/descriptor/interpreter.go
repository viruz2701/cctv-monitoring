// Package descriptor — Universal Protocol Interpreter.
//
// ═══════════════════════════════════════════════════════════════════════════
// PROTO-02: Universal Protocol Interpreter
//
// Интерпретатор выполняет операции с устройством на основе ProtocolDescriptor.
// Поддерживает:
//   - HTTP/HTTPS транспорты (GET, POST, PUT, DELETE)
//   - Digest и Basic аутентификацию
//   - Парсинг JSON, XML, key-value форматов
//   - Go templates для URL, headers, body
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS V5: Input validation
//   - OWASP ASVS V7: Error handling
//
// ═══════════════════════════════════════════════════════════════════════════
package descriptor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/icholy/digest"
)

// ProtocolInterpreter — универсальный интерпретатор ProtocolDescriptor.
type ProtocolInterpreter struct {
	httpClient *http.Client
	logger     *slog.Logger
}

// NewProtocolInterpreter создаёт новый ProtocolInterpreter.
func NewProtocolInterpreter(logger *slog.Logger) *ProtocolInterpreter {
	return &ProtocolInterpreter{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger.With("component", "protocol_interpreter"),
	}
}

// Execute выполняет операцию по дескриптору.
func (i *ProtocolInterpreter) Execute(
	ctx context.Context,
	descriptor *ProtocolDescriptor,
	endpoint string,
	params map[string]interface{},
) (*ExecutionResult, error) {
	protoName := descriptor.DefaultProtocol
	if protoName == "" {
		for name := range descriptor.Protocols {
			protoName = name
			break
		}
	}

	proto, ok := descriptor.Protocols[protoName]
	if !ok {
		return nil, fmt.Errorf("protocol %q not found", protoName)
	}

	ep, ok := proto.Endpoints[endpoint]
	if !ok {
		return nil, fmt.Errorf("endpoint %q not found in protocol %q", endpoint, protoName)
	}

	switch proto.Transport {
	case "http", "https":
		return i.executeHTTP(ctx, descriptor, &proto, &ep, params)
	case "tcp", "udp":
		return nil, fmt.Errorf("transport %q not implemented yet", proto.Transport)
	default:
		return nil, fmt.Errorf("unsupported transport: %q", proto.Transport)
	}
}

// executeHTTP выполняет HTTP запрос по дескриптору.
func (i *ProtocolInterpreter) executeHTTP(
	ctx context.Context,
	descriptor *ProtocolDescriptor,
	proto *Protocol,
	ep *Endpoint,
	params map[string]interface{},
) (*ExecutionResult, error) {
	start := time.Now()

	renderedPath, err := renderTemplate(ep.Path, params)
	if err != nil {
		return nil, fmt.Errorf("render path: %w", err)
	}

	baseURL, err := renderTemplate(proto.BaseURL, params)
	if err != nil {
		return nil, fmt.Errorf("render base_url: %w", err)
	}

	fullURL := baseURL + renderedPath

	var bodyReader io.Reader
	if ep.Body != "" {
		renderedBody, err := renderTemplate(ep.Body, params)
		if err != nil {
			return nil, fmt.Errorf("render body: %w", err)
		}
		bodyReader = strings.NewReader(renderedBody)
	}

	req, err := http.NewRequestWithContext(ctx, ep.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if ep.ContentType != "" {
		req.Header.Set("Content-Type", ep.ContentType)
	} else {
		req.Header.Set("Content-Type", "application/xml")
	}
	for k, v := range proto.Headers {
		req.Header.Set(k, v)
	}
	for k, v := range ep.Headers {
		req.Header.Set(k, v)
	}

	client := i.httpClient
	if proto.Auth.Type == "digest" {
		authUser, _ := renderTemplate(proto.Auth.Username, params)
		authPass, _ := renderTemplate(proto.Auth.Password, params)
		client = &http.Client{
			Timeout: i.httpClient.Timeout,
			Transport: &digest.Transport{
				Username: authUser,
				Password: authPass,
			},
		}
	} else if proto.Auth.Type == "basic" {
		authUser, _ := renderTemplate(proto.Auth.Username, params)
		authPass, _ := renderTemplate(proto.Auth.Password, params)
		req.SetBasicAuth(authUser, authPass)
	} else if proto.Auth.Type == "bearer" {
		token, _ := renderTemplate(proto.Auth.Token, params)
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	result := &ExecutionResult{
		StatusCode: resp.StatusCode,
		RawBody:    body,
		DurationMs: time.Since(start).Milliseconds(),
	}

	if ep.ResponseParser.Format != "" {
		data, err := parseResponse(body, &ep.ResponseParser)
		if err != nil {
			i.logger.Warn("response parsing failed", "endpoint", ep.Path, "error", err)
			result.Data = map[string]interface{}{"raw": string(body)}
		} else {
			result.Data = data
		}
	} else {
		result.Data = map[string]interface{}{"raw": string(body)}
	}

	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	if ep.ResponseParser.SuccessCheck != "" && result.Data != nil {
		if val, ok := result.Data["success"]; ok {
			result.Success = fmt.Sprintf("%v", val) == "true" || val == true
		}
	}

	return result, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Response Parsers
// ────────────────────────────────────────────────────────────────────────────

func parseResponse(body []byte, parser *ResponseParser) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	switch parser.Format {
	case "json":
		return parseJSON(body, parser.Mappings)
	case "xml":
		return parseXML(body, parser.Mappings)
	case "key_value":
		return parseKeyValue(body, parser.Separator, parser.Mappings)
	case "raw":
		result["raw"] = string(body)
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported parser format: %q", parser.Format)
	}
}

func parseJSON(body []byte, mappings map[string]string) (map[string]interface{}, error) {
	var jsonData map[string]interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return nil, fmt.Errorf("json parse: %w", err)
	}

	result := make(map[string]interface{})
	for key, path := range mappings {
		value := resolveJSONPath(jsonData, path)
		if value != nil {
			result[key] = value
		}
	}

	if len(mappings) == 0 {
		return jsonData, nil
	}

	return result, nil
}

func resolveJSONPath(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		case []interface{}:
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					if val, exists := m[part]; exists {
						return val
					}
				}
			}
			return nil
		default:
			return nil
		}
	}

	return current
}

func parseXML(body []byte, mappings map[string]string) (map[string]interface{}, error) {
	bodyStr := string(body)
	result := make(map[string]interface{})

	for key, tag := range mappings {
		value := extractXMLTag(bodyStr, tag)
		if value != "" {
			result[key] = value
		}
	}

	return result, nil
}

func extractXMLTag(xml, tag string) string {
	if strings.Contains(tag, "/") {
		parts := strings.Split(tag, "/")
		tag = parts[len(parts)-1]
	}

	openTag := fmt.Sprintf("<%s>", tag)
	closeTag := fmt.Sprintf("</%s>", tag)

	start := strings.Index(xml, openTag)
	if start == -1 {
		return ""
	}
	start += len(openTag)

	end := strings.Index(xml[start:], closeTag)
	if end == -1 {
		return ""
	}

	return strings.TrimSpace(xml[start : start+end])
}

func parseKeyValue(body []byte, separator string, mappings map[string]string) (map[string]interface{}, error) {
	if separator == "" {
		separator = "="
	}

	result := make(map[string]interface{})
	lines := strings.Split(string(body), "\n")
	kvMap := make(map[string]string)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, separator, 2)
		if len(parts) == 2 {
			kvMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	for key, sourceKey := range mappings {
		if val, ok := kvMap[sourceKey]; ok {
			result[key] = val
		}
	}

	if len(mappings) == 0 {
		for k, v := range kvMap {
			result[k] = v
		}
	}

	return result, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Template Engine
// ────────────────────────────────────────────────────────────────────────────

func renderTemplate(tmpl string, params map[string]interface{}) (string, error) {
	if !strings.Contains(tmpl, "{{") {
		return tmpl, nil
	}

	t, err := template.New("").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, params); err != nil {
		return tmpl, nil
	}

	return buf.String(), nil
}
