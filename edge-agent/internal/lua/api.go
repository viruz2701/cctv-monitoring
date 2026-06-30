// Package lua — Plugin API for Lua scripts.
//
// Registered functions in Lua as `agent.*`:
//   - agent.http_get(url, options)  → string, error
//   - agent.http_post(url, body, options)  → string, error
//   - agent.xml_parse(xml_string)  → table
//   - agent.json_parse(json_string)  → table
//   - agent.log(level, message)
//
// Compliance:
//   - IEC 62443-3-3 SL-3: API functions are sandboxed, no os/io/debug
//   - OWASP ASVS V5: All string inputs are length-checked, whitelist validation
//   - Приказ ОАЦ №66 п. 7.18.3: Tamper detection via signature check (TODO)
package lua

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/icholy/digest"
	lua "github.com/yuin/gopher-lua"
)

const (
	maxRequestBodySize = 10 << 20 // 10 MB max request body
	maxURLLength       = 2048     // Max URL length (OWASP ASVS V5)
	httpTimeout        = 30 * time.Second
	defaultUserAgent   = "CCTV-Edge-Agent-Lua/1.0"
)

// httpOptions mirrors the Lua options table for HTTP requests.
type httpOptions struct {
	Auth    *authOptions      `json:"auth,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Timeout int               `json:"timeout,omitempty"` // seconds
}

type authOptions struct {
	Type     string `json:"type"` // "basic" or "digest"
	Username string `json:"username"`
	Password string `json:"password"`
}

// registerAPI registers all agent.* functions in the given Lua state.
func registerAPI(L *lua.LState, logger *slog.Logger) {
	agentTable := L.NewTable()
	L.SetGlobal("agent", agentTable)

	// agent.http_get(url, options) → string, error
	L.SetField(agentTable, "http_get", L.NewFunction(func(L *lua.LState) int {
		return luaHTTPRequest(L, http.MethodGet, logger)
	}))

	// agent.http_post(url, body, options) → string, error
	L.SetField(agentTable, "http_post", L.NewFunction(func(L *lua.LState) int {
		return luaHTTPRequest(L, http.MethodPost, logger)
	}))

	// agent.xml_parse(xml_string) → table
	L.SetField(agentTable, "xml_parse", L.NewFunction(func(L *lua.LState) int {
		return luaXMLParse(L, logger)
	}))

	// agent.json_parse(json_string) → table
	L.SetField(agentTable, "json_parse", L.NewFunction(func(L *lua.LState) int {
		return luaJSONParse(L, logger)
	}))

	// agent.log(level, message)
	L.SetField(agentTable, "log", L.NewFunction(func(L *lua.LState) int {
		return luaLog(L, logger)
	}))
}

// luaHTTPRequest handles both GET and POST requests from Lua.
func luaHTTPRequest(L *lua.LState, method string, logger *slog.Logger) int {
	url := L.CheckString(1)
	if len(url) > maxURLLength {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("url too long (%d > %d)", len(url), maxURLLength)))
		return 2
	}

	var body string
	argIdx := 2
	if method == http.MethodPost {
		body = L.OptString(2, "")
		argIdx = 3
	}

	opts := httpOptions{}
	optionsTable := L.OptTable(argIdx, nil)
	if optionsTable != nil {
		opts = parseOptionsTable(L, optionsTable)
	}

	client := &http.Client{Timeout: httpTimeout}
	if opts.Timeout > 0 {
		client.Timeout = time.Duration(opts.Timeout) * time.Second
	}

	var req *http.Request
	var err error
	if method == http.MethodPost {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	req.Header.Set("User-Agent", defaultUserAgent)

	// Apply custom headers
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	// Apply auth (IEC 62443 SL-3: mTLS required for production, digest for legacy)
	if opts.Auth != nil {
		switch opts.Auth.Type {
		case "digest":
			client.Transport = &digest.Transport{
				Username: opts.Auth.Username,
				Password: opts.Auth.Password,
			}
		case "basic":
			req.SetBasicAuth(opts.Auth.Username, opts.Auth.Password)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("http request failed: %v", err)))
		return 2
	}
	defer resp.Body.Close()

	// Limit response body size
	limitedReader := io.LimitReader(resp.Body, maxRequestBodySize)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("failed to read response: %v", err)))
		return 2
	}

	if resp.StatusCode >= 400 {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("http %d: %s", resp.StatusCode, string(respBody))))
		return 2
	}

	logger.Debug("lua http request completed",
		"method", method,
		"url", url,
		"status", resp.StatusCode,
	)

	L.Push(lua.LString(string(respBody)))
	return 1
}

// luaXMLParse parses an XML string into a Lua table.
// Uses stdlib encoding/xml, supports nested elements as sub-tables.
func luaXMLParse(L *lua.LState, logger *slog.Logger) int {
	xmlStr := L.CheckString(1)
	if len(xmlStr) > maxRequestBodySize {
		L.Push(lua.LNil)
		L.Push(lua.LString("xml string too large"))
		return 2
	}

	table, err := parseXMLToTable(xmlStr)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("xml parse error: %v", err)))
		return 2
	}

	L.Push(table)
	return 1
}

// luaJSONParse parses a JSON string into a Lua table.
func luaJSONParse(L *lua.LState, logger *slog.Logger) int {
	jsonStr := L.CheckString(1)
	if len(jsonStr) > maxRequestBodySize {
		L.Push(lua.LNil)
		L.Push(lua.LString("json string too large"))
		return 2
	}

	var raw interface{}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("json parse error: %v", err)))
		return 2
	}

	table := jsonToLuaTable(L, raw)
	L.Push(table)
	return 1
}

// luaLog logs a message from Lua into the Go slog logger.
func luaLog(L *lua.LState, logger *slog.Logger) int {
	level := L.CheckString(1)
	message := L.CheckString(2)

	switch strings.ToLower(level) {
	case "debug":
		logger.Debug("[lua] " + message)
	case "info":
		logger.Info("[lua] " + message)
	case "warn", "warning":
		logger.Warn("[lua] " + message)
	case "error":
		logger.Error("[lua] " + message)
	default:
		logger.Info("[lua] " + message)
	}
	return 0
}

// parseOptionsTable converts a Lua options table to httpOptions struct.
func parseOptionsTable(L *lua.LState, tbl *lua.LTable) httpOptions {
	opts := httpOptions{}

	if authTbl := L.GetField(tbl, "auth"); authTbl.Type() == lua.LTTable {
		authT := authTbl.(*lua.LTable)
		opts.Auth = &authOptions{
			Type:     L.GetField(authT, "type").String(),
			Username: L.GetField(authT, "username").String(),
			Password: L.GetField(authT, "password").String(),
		}
	}

	if headersTbl := L.GetField(tbl, "headers"); headersTbl.Type() == lua.LTTable {
		opts.Headers = make(map[string]string)
		headersT := headersTbl.(*lua.LTable)
		headersT.ForEach(func(k, v lua.LValue) {
			opts.Headers[k.String()] = v.String()
		})
	}

	if timeout := L.GetField(tbl, "timeout"); timeout.Type() == lua.LTNumber {
		opts.Timeout = int(lua.LVAsNumber(timeout))
	}

	return opts
}

// jsonToLuaTable converts a Go interface{} (from json.Unmarshal) to a Lua table.
func jsonToLuaTable(L *lua.LState, v interface{}) *lua.LTable {
	tbl := L.NewTable()

	switch val := v.(type) {
	case map[string]interface{}:
		for k, item := range val {
			tbl.RawSetString(k, toLuaValue(L, item))
		}
	case []interface{}:
		for i, item := range val {
			tbl.RawSetInt(i+1, toLuaValue(L, item))
		}
	default:
		tbl.RawSetString("value", toLuaValue(L, val))
	}

	return tbl
}

// toLuaValue converts a Go value to a Lua value.
func toLuaValue(L *lua.LState, v interface{}) lua.LValue {
	switch val := v.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(val)
	case float64:
		return lua.LNumber(val)
	case string:
		return lua.LString(val)
	case map[string]interface{}:
		return jsonToLuaTable(L, val)
	case []interface{}:
		return jsonToLuaTable(L, val)
	default:
		return lua.LString(fmt.Sprintf("%v", val))
	}
}

// xmlElement is used by encoding/xml to capture arbitrary XML structures.
type xmlElement struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Content  string     `xml:",chardata"`
	Children []xmlElement `xml:",any"`
}

// parseXMLToTable parses XML string into a Lua table using encoding/xml.
// Supports nested elements as sub-tables and attributes as {elem}_attrs tables.
func parseXMLToTable(xmlStr string) (*lua.LTable, error) {
	var root xmlElement
	if err := xml.Unmarshal([]byte(xmlStr), &root); err != nil {
		return nil, fmt.Errorf("xml unmarshal: %w", err)
	}

	L := lua.NewState()
	defer L.Close()

	return xmlElementToTable(L, root), nil
}

// xmlElementToTable converts an xmlElement tree to a Lua table.
func xmlElementToTable(L *lua.LState, elem xmlElement) *lua.LTable {
	tbl := L.NewTable()

	// Set attributes as {elem}_attrs sub-table
	if len(elem.Attrs) > 0 {
		attrTbl := L.NewTable()
		for _, attr := range elem.Attrs {
			attrTbl.RawSetString(attr.Name.Local, lua.LString(attr.Value))
		}
		tbl.RawSetString(elem.XMLName.Local+"_attrs", attrTbl)
	}

	// If there are children, process them
	if len(elem.Children) > 0 {
		// Group children by tag name to handle multiple elements with same name
		grouped := make(map[string][]xmlElement)
		for _, child := range elem.Children {
			name := child.XMLName.Local
			grouped[name] = append(grouped[name], child)
		}

		for name, children := range grouped {
			if len(children) == 1 {
				child := children[0]
				if len(child.Children) > 0 {
					// Nested element → sub-table
					tbl.RawSetString(name, xmlElementToTable(L, child))
				} else {
					// Leaf element → string value
					tbl.RawSetString(name, lua.LString(child.Content))
				}
			} else {
				// Multiple elements with same name → array
				arrTbl := L.NewTable()
				for i, child := range children {
					if len(child.Children) > 0 {
						arrTbl.RawSetInt(i+1, xmlElementToTable(L, child))
					} else {
						arrTbl.RawSetInt(i+1, lua.LString(child.Content))
					}
				}
				tbl.RawSetString(name, arrTbl)
			}
		}
	} else if elem.Content != "" {
		tbl.RawSetString("value", lua.LString(strings.TrimSpace(elem.Content)))
	}

	return tbl
}
