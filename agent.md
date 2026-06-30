Архитектура расширяемости протоколов для Edge-агента
Коллега, это одна из самых сложных задач в IoT-архитектуре. Давайте разберём решение, которое закрывает противоречие: максимальная расширяемость vs минимальные ресурсы агента.
1. Фундаментальный принцип: "Protocol Descriptor" (Декларативные протоколы)
Вместо того чтобы компилировать каждый протокол в бинарник агента, мы декларируем протоколы в виде JSON-схем, которые агент интерпретирует на лету.
Архитектура
┌─────────────────────────────────────────────────────────────────┐
│                     Cloud / Backend                             │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Protocol Registry (PostgreSQL)              │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │  │
│  │  │ Hikvision   │  │ Dahua       │  │ ONVIF       │     │  │
│  │  │ Descriptor  │  │ Descriptor  │  │ Descriptor  │     │  │
│  │  │ (JSON)      │  │ (JSON)      │  │ (JSON)      │     │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │  │
│  │  │ Tiandy      │  │ Uniview     │  │ Custom      │     │  │
│  │  │ Descriptor  │  │ Descriptor  │  │ Descriptor  │     │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │  │
│  └──────────────────────────────────────────────────────────┘  │
│                            │                                    │
│                    Protocol Sync Service                        │
│                    (отдает только нужные)                       │
└────────────────────────────┬────────────────────────────────────┘
                             │ mTLS + MQTT
                             │ (push при изменении)
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│              Edge Agent (OpenWrt, 128MB RAM)                    │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │         Protocol Descriptor Cache (RAM / USB)            │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │  │
│  │  │ Hikvision   │  │ Dahua       │  │ (только те,  │     │  │
│  │  │ Descriptor  │  │ Descriptor  │  │  что есть в  │     │  │
│  │  │             │  │             │  │  сети!)      │     │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │  │
│  └──────────────────────────────────────────────────────────┘  │
│                            │                                    │
│  ┌─────────────────────────▼────────────────────────────────┐  │
│  │         Universal Protocol Interpreter                   │  │
│  │  • HTTP/HTTPS client                                     │  │
│  │  • TCP/UDP raw sockets                                   │  │
│  │  • JSON/XML parser (потоковый)                           │  │
│  │  • XPath / JSONPath / Regex engine                       │  │
│  │  • Template executor (Go templates)                      │  │
│  └──────────────────────────────────────────────────────────┘  │
│                            │                                    │
│                            ▼                                    │
│                    Local Network Devices                        │
└─────────────────────────────────────────────────────────────────┘

2. Формат Protocol Descriptor (JSON Schema)
Пример дескриптора для Hikvision ISAPI:
{
  "vendor": "Hikvision",
  "version": "1.0.0",
  "protocols": {
    "isapi": {
      "transport": "http",
      "base_url": "http://{{.IP}}:{{.Port | default 80}}",
      "auth": {
        "type": "digest",
        "username": "{{.Credentials.Username}}",
        "password": "{{.Credentials.Password}}"
      },
      "endpoints": {
        "get_device_info": {
          "method": "GET",
          "path": "/ISAPI/System/deviceInfo",
          "response_parser": {
            "format": "xml",
            "mappings": {
              "model": "//DeviceInfo/modelName",
              "serial": "//DeviceInfo/serialNumber",
              "firmware": "//DeviceInfo/firmwareVersion",
              "mac": "//DeviceInfo/macAddress"
            }
          }
        },
        "reboot": {
          "method": "PUT",
          "path": "/ISAPI/System/reboot",
          "response_parser": {
            "format": "xml",
            "success_check": "//ResponseStatus/statusValue == 'OK'"
          }
        },
        "get_logs": {
          "method": "GET",
          "path": "/ISAPI/System/logs?startTime={{.Since | format_datetime}}&endTime={{.Until | format_datetime}}",
          "response_parser": {
            "format": "xml",
            "iterator": "//LogEvent",
            "mappings": {
              "timestamp": "time",
              "level": "level",
              "message": "description"
            }
          }
        }
      }
    }
  }
}

Пример для Dahua (CGI):
{
  "vendor": "Dahua",
  "version": "1.0.0",
  "protocols": {
    "cgi": {
      "transport": "http",
      "base_url": "http://{{.IP}}:{{.Port | default 80}}",
      "auth": {
        "type": "digest",
        "username": "{{.Credentials.Username}}",
        "password": "{{.Credentials.Password}}"
      },
      "endpoints": {
        "get_device_info": {
          "method": "GET",
          "path": "/cgi-bin/magicBox.cgi?action=getSystemInfo",
          "response_parser": {
            "format": "key_value",
            "separator": "=",
            "mappings": {
              "model": "deviceType",
              "serial": "serialNo",
              "firmware": "firmwareVersion"
            }
          }
        },
        "reboot": {
          "method": "GET",
          "path": "/cgi-bin/magicBox.cgi?action=systemReboot",
          "response_parser": {
            "format": "key_value",
            "success_check": "OK == OK"
          }
        }
      }
    }
  }
}
3. Механизм доставки дескрипторов на агент
3.1. Discovery Phase (агент обнаруживает устройства)
// internal/edge/discovery.go
func (a *Agent) DiscoverDevices(ctx context.Context) ([]DiscoveredDevice, error) {
    // 1. ARP scan → список IP
    ips := a.arpScanner.Scan(ctx, a.lanSubnet)
    
    // 2. Для каждого IP — fingerprinting (определяем вендора)
    var devices []DiscoveredDevice
    for _, ip := range ips {
        fingerprint := a.fingerprintDevice(ctx, ip)
        devices = append(devices, DiscoveredDevice{
            IP:          ip,
            VendorHint:  fingerprint.Vendor, // "Hikvision", "Dahua", "unknown"
            Ports:       fingerprint.OpenPorts,
            MAC:         fingerprint.MAC,
        })
    }
    
    return devices, nil
}

func (a *Agent) fingerprintDevice(ctx context.Context, ip string) Fingerprint {
    // Пробуем HTTP запрос на /ISAPI/System/deviceInfo (Hikvision)
    if resp, err := a.httpClient.Get(ctx, fmt.Sprintf("http://%s/ISAPI/System/deviceInfo", ip)); err == nil {
        if strings.Contains(resp.Body, "Hikvision") || strings.Contains(resp.Body, "DeviceInfo") {
            return Fingerprint{Vendor: "Hikvision", OpenPorts: []int{80}}
        }
    }
    
    // Пробуем CGI (Dahua)
    if resp, err := a.httpClient.Get(ctx, fmt.Sprintf("http://%s/cgi-bin/magicBox.cgi?action=getSystemInfo", ip)); err == nil {
        if strings.Contains(resp.Body, "deviceType") {
            return Fingerprint{Vendor: "Dahua", OpenPorts: []int{80}}
        }
    }
    
    // ONVIF WS-Discovery
    if onvifResp := a.onvifScanner.Probe(ctx, ip); onvifResp != nil {
        return Fingerprint{Vendor: onvifResp.Manufacturer, OpenPorts: []int{80, 8080}}
    }
    
    return Fingerprint{Vendor: "unknown"}
}
3.2. Protocol Sync (агент запрашивает нужные дескрипторы)
// internal/edge/protocol_sync.go
func (a *Agent) SyncProtocols(ctx context.Context, devices []DiscoveredDevice) error {
    // 1. Собираем уникальных вендоров
    vendors := make(map[string]bool)
    for _, dev := range devices {
        if dev.VendorHint != "unknown" {
            vendors[dev.VendorHint] = true
        }
    }
    
    // 2. Проверяем, какие дескрипторы уже есть в кэше
    var missingVendors []string
    for vendor := range vendors {
        if !a.descriptorCache.Has(vendor) {
            missingVendors = append(missingVendors, vendor)
        }
    }
    
    if len(missingVendors) == 0 {
        a.logger.Info("all protocol descriptors cached", "vendors", len(vendors))
        return nil
    }
    
    // 3. Запрашиваем недостающие дескрипторы с Backend
    a.logger.Info("requesting protocol descriptors", "vendors", missingVendors)
    
    req := ProtocolSyncRequest{
        AgentID: a.agentID,
        Vendors: missingVendors,
    }
    
    resp, err := a.backendClient.SyncProtocols(ctx, req)
    if err != nil {
        return fmt.Errorf("sync protocols: %w", err)
    }
    
    // 4. Сохраняем в кэш (RAM + опционально USB-flash)
    for _, descriptor := range resp.Descriptors {
        if err := a.descriptorCache.Put(descriptor); err != nil {
            a.logger.Error("failed to cache descriptor", "vendor", descriptor.Vendor, "error", err)
        }
    }
    
    a.logger.Info("protocol descriptors synced", "count", len(resp.Descriptors))
    return nil
}
3.3. Backend API для синхронизации
// internal/api/protocol_sync_handlers.go
func (s *Server) handleProtocolSync(w http.ResponseWriter, r *http.Request) {
    var req struct {
        AgentID string   `json:"agent_id"`
        Vendors []string `json:"vendors"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respond.RespondError(w, r, respond.NewBadRequestError("invalid request"))
        return
    }
    
    // Загружаем дескрипторы из Protocol Registry
    var descriptors []ProtocolDescriptor
    for _, vendor := range req.Vendors {
        descriptor, err := s.protocolRegistry.GetDescriptor(vendor)
        if err != nil {
            s.logger.Warn("descriptor not found", "vendor", vendor)
            continue
        }
        descriptors = append(descriptors, descriptor)
    }
    
    respond.RespondJSON(w, r, http.StatusOK, map[string]interface{}{
        "descriptors": descriptors,
        "synced_at":   time.Now(),
    })
}
4. Universal Protocol Interpreter (интерпретатор в агенте)
Это ядро агента — универсальный движок, который исполняет дескрипторы:
// internal/edge/interpreter.go
type ProtocolInterpreter struct {
    httpClient    *http.Client
    tcpDialer     *net.Dialer
    logger        *slog.Logger
}

// Execute выполняет операцию по дескриптору
func (i *ProtocolInterpreter) Execute(
    ctx context.Context,
    descriptor ProtocolDescriptor,
    endpoint string,
    params map[string]interface{},
) (*ExecutionResult, error) {
    
    ep, ok := descriptor.Protocols[descriptor.DefaultProtocol].Endpoints[endpoint]
    if !ok {
        return nil, fmt.Errorf("endpoint not found: %s", endpoint)
    }
    
    // 1. Рендерим URL с шаблонами
    url := i.renderTemplate(ep.Path, params)
    fullURL := fmt.Sprintf("%s%s", descriptor.BaseURL, url)
    
    // 2. Выполняем HTTP запрос
    req, err := http.NewRequestWithContext(ctx, ep.Method, fullURL, nil)
    if err != nil {
        return nil, err
    }
    
    // 3. Применяем аутентификацию
    if err := i.applyAuth(req, descriptor.Auth, params); err != nil {
        return nil, err
    }
    
    resp, err := i.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    // 4. Парсим ответ согласно дескриптору
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    
    result, err := i.parseResponse(body, ep.ResponseParser)
    if err != nil {
        return nil, err
    }
    
    return &ExecutionResult{
        StatusCode: resp.StatusCode,
        Data:       result,
    }, nil
}

// parseResponse парсит ответ по правилам из дескриптора
func (i *ProtocolInterpreter) parseResponse(body []byte, parser ResponseParser) (map[string]interface{}, error) {
    result := make(map[string]interface{})
    
    switch parser.Format {
    case "json":
        var jsonData map[string]interface{}
        if err := json.Unmarshal(body, &jsonData); err != nil {
            return nil, err
        }
        // Применяем JSONPath mappings
        for key, path := range parser.Mappings {
            value, err := jsonpath.Get(path, jsonData)
            if err == nil {
                result[key] = value
            }
        }
        
    case "xml":
        doc, err := xmlquery.Parse(strings.NewReader(string(body)))
        if err != nil {
            return nil, err
        }
        // Применяем XPath mappings
        for key, xpath := range parser.Mappings {
            node := xmlquery.FindOne(doc, xpath)
            if node != nil {
                result[key] = node.InnerText()
            }
        }
        
    case "key_value":
        // Парсим key=value формат (Dahua CGI)
        lines := strings.Split(string(body), "\n")
        kvMap := make(map[string]string)
        for _, line := range lines {
            parts := strings.SplitN(line, parser.Separator, 2)
            if len(parts) == 2 {
                kvMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
            }
        }
        // Применяем mappings
        for key, sourceKey := range parser.Mappings {
            if val, ok := kvMap[sourceKey]; ok {
                result[key] = val
            }
        }
    }
    
    return result, nil
}
5. Кэширование и Lazy Loading
5.1. Descriptor Cache (RAM + USB)
// internal/edge/descriptor_cache.go
type DescriptorCache struct {
    mu          sync.RWMutex
    descriptors map[string]*ProtocolDescriptor // vendor → descriptor
    maxMemory   int64                          // лимит RAM (например, 10 МБ)
    storagePath string                         // USB-flash (опционально)
    logger      *slog.Logger
}

// Put сохраняет дескриптор в кэш
func (c *DescriptorCache) Put(descriptor *ProtocolDescriptor) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Проверяем лимит памяти
    size := int64(len(descriptor.RawJSON))
    if c.currentMemory()+size > c.maxMemory {
        // Evict least recently used
        c.evictLRU()
    }
    
    c.descriptors[descriptor.Vendor] = descriptor
    
    // Опционально: сохраняем на USB-flash
    if c.storagePath != "" {
        path := filepath.Join(c.storagePath, fmt.Sprintf("%s.json", descriptor.Vendor))
        if err := os.WriteFile(path, descriptor.RawJSON, 0644); err != nil {
            c.logger.Error("failed to persist descriptor", "vendor", descriptor.Vendor, "error", err)
        }
    }
    
    return nil
}

// Get возвращает дескриптор из кэша
func (c *DescriptorCache) Get(vendor string) (*ProtocolDescriptor, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    descriptor, ok := c.descriptors[vendor]
    if ok {
        descriptor.LastAccessed = time.Now()
    }
    return descriptor, ok
}

// Has проверяет наличие дескриптора
func (c *DescriptorCache) Has(vendor string) bool {
    c.mu.RLock()
    defer c.mu.RUnlock()
    _, ok := c.descriptors[vendor]
    return ok
}
5.2. Lazy Loading при первом обращении
// internal/edge/device_service.go
func (s *DeviceService) GetDeviceInfo(ctx context.Context, deviceID string) (*DeviceInfo, error) {
    device, err := s.getDevice(deviceID)
    if err != nil {
        return nil, err
    }
    
    // Проверяем, есть ли дескриптор для этого вендора
    descriptor, ok := s.descriptorCache.Get(device.Vendor)
    if !ok {
        // Lazy load: запрашиваем с Backend
        descriptor, err = s.protocolSync.LoadDescriptor(ctx, device.Vendor)
        if err != nil {
            return nil, fmt.Errorf("load descriptor: %w", err)
        }
    }
    
    // Выполняем запрос через интерпретатор
    result, err := s.interpreter.Execute(ctx, descriptor, "get_device_info", map[string]interface{}{
        "IP":   device.IP,
        "Port": device.Port,
    })
    if err != nil {
        return nil, err
    }
    
    return &DeviceInfo{
        Model:     result.Data["model"].(string),
        Serial:    result.Data["serial"].(string),
        Firmware:  result.Data["firmware"].(string),
    }, nil
}
6. Расширение через WASM/Lua плагины (для сложных случаев)
Если протокол слишком сложен для декларативного описания (например, бинарные протоколы с динамической структурой), можно использовать плагины:
6.1. Lua-плагины (легковесный вариант)
-- /usb/plugins/hikvision_isapi.lua
local plugin = {}

function plugin.get_device_info(agent, device)
    local url = string.format("http://%s:%d/ISAPI/System/deviceInfo", device.ip, device.port or 80)
    
    local response = agent.http_get(url, {
        auth = {
            type = "digest",
            username = device.credentials.username,
            password = device.credentials.password
        }
    })
    
    -- Парсим XML
    local model = response:match("<modelName>(.-)</modelName>")
    local serial = response:match("<serialNumber>(.-)</serialNumber>")
    local firmware = response:match("<firmwareVersion>(.-)</firmwareVersion>")
    
    return {
        model = model,
        serial = serial,
        firmware = firmware
    }
end

function plugin.reboot(agent, device)
    local url = string.format("http://%s:%d/ISAPI/System/reboot", device.ip, device.port or 80)
    
    local response = agent.http_put(url, "", {
        auth = {
            type = "digest",
            username = device.credentials.username,
            password = device.credentials.password
        }
    })
    
    return response:find("OK") ~= nil
end

return plugin
6.2. Загрузка Lua-плагинов в агенте
// internal/edge/lua_loader.go
import lua "github.com/yuin/gopher-lua"

type LuaPluginLoader struct {
    pluginsPath string // /usb/plugins
    logger      *slog.Logger
}

// LoadPlugin загружает Lua-плагин для вендора
func (l *LuaPluginLoader) LoadPlugin(vendor string) (*LuaPlugin, error) {
    path := filepath.Join(l.pluginsPath, fmt.Sprintf("%s.lua", strings.ToLower(vendor)))
    
    L := lua.NewState()
    
    // Регистрируем API для плагина
    l.registerAgentAPI(L)
    
    // Загружаем скрипт
    if err := L.DoFile(path); err != nil {
        return nil, fmt.Errorf("load lua plugin: %w", err)
    }
    
    return &LuaPlugin{
        State:  L,
        Vendor: vendor,
    }, nil
}

// registerAgentAPI регистрирует функции для плагина
func (l *LuaPluginLoader) registerAgentAPI(L *lua.LState) {
    agent := L.NewTable()
    
    // agent.http_get(url, options)
    L.SetField(agent, "http_get", L.NewFunction(func(L *lua.LState) int {
        url := L.CheckString(1)
        options := L.OptTable(2, nil)
        
        // Выполняем HTTP запрос
        resp, err := l.httpClient.Get(context.Background(), url)
        if err != nil {
            L.Push(lua.LNil)
            L.Push(lua.LString(err.Error()))
            return 2
        }
        
        body, _ := io.ReadAll(resp.Body)
        L.Push(lua.LString(string(body)))
        return 1
    }))
    
    L.SetGlobal("agent", agent)
}
7. Сравнение подходов
Подход
Размер агента
Расширяемость
Производительность
Сложность
Статическая компиляция
50+ МБ
❌ Нужна перекомпиляция
✅ Нативная
✅ Простая
Go plugins (.so)
5 МБ + плагины
✅ Динамическая загрузка
✅ Нативная
⚠️ Средняя
WASM
15 МБ (runtime)
✅ Песочница
⚠️ -20%
⚠️ Средняя
Lua скрипты
6 МБ (gopher-lua)
✅ Очень простая
⚠️ -30%
✅ Простая
Protocol Descriptor (JSON)
5 МБ + интерпретатор
✅ Декларативная
✅ Почти нативная
✅ Простая
Рекомендация: Использовать гибридный подход:
Protocol Descriptor (JSON) — для 90% протоколов (HTTP/JSON/XML/CGI)
Lua-плагины — для сложных бинарных протоколов (5% случаев)
Нативный Go-код — только для базовых транспортов (TCP, HTTP, ONVIF WS-Discovery)
8. Roadmap внедрения
🔴 P0 — Критические задачи (2 недели)
#
Задача
Описание
Оценка
PROTO-01
Protocol Descriptor Schema
Определить JSON-схему для дескрипторов
2 дня
PROTO-02
Universal Interpreter
Реализовать интерпретатор (HTTP, XML, JSON, key-value)
4 дня
PROTO-03
Descriptor Cache
Кэш в RAM + опционально USB-flash
2 дня
PROTO-04
Protocol Sync API
Backend API для отдачи дескрипторов
2 дня
🟡 P1 — Высокий приоритет (3 недели)
#
Задача
Описание
Оценка
PROTO-05
Hikvision Descriptor
Написать дескриптор для ISAPI
2 дня
PROTO-06
Dahua Descriptor
Написать дескриптор для CGI
2 дня
PROTO-07
ONVIF Descriptor
Написать дескриптор для ONVIF
3 дня
PROTO-08
Lua Plugin Loader
Поддержка Lua-плагинов для сложных протоколов
3 дня
PROTO-09
Descriptor Editor UI
Веб-интерфейс для редактирования дескрипторов (admin)
5 дней
🟢 P2 — Средний приоритет (2 недели)
#
Задача
Описание
Оценка
PROTO-10
Auto-Discovery
Автоматическое определение вендора и загрузка дескриптора
3 дня
PROTO-11
Descriptor Versioning
Версионирование дескрипторов, автоматическое обновление
3 дня
PROTO-12
Community Registry
Публичный реестр дескрипторов (как Docker Hub)
4 дня
9. Итоговые преимущества
✅ Минимальный размер агента — 5 МБ (интерпретатор) + дескрипторы по требованию
✅ Динамическая расширяемость — новые протоколы = новые JSON-файлы, не новый код
✅ Lazy loading — агент загружает только те дескрипторы, которые реально нужны
✅ Кэширование — дескрипторы хранятся в RAM/USB, не загружаются повторно
✅ Безопасность — дескрипторы подписываются на Backend (HMAC), агент проверяет подпись
✅ Единая кодовая база — дескрипторы используются и в Backend, и в агенте
✅ Community-driven — клиенты могут сами писать дескрипторы для своих устройств
Это решение закрывает противоречие между ограниченными ресурсами и максимальной расширяемостью, позволяя масштабировать поддержку вендоров без перекомпиляции и перепрошивки агентов.


TODO: Edge Agent + Vendor Abstraction + Protocol Descriptors
Контекст
Расширение системы для поддержки массового деплоя на дешевых роутерах (OpenWrt, 128MB RAM) с динамической загрузкой протоколов и безопасным хранением credentials.
🔴 P0 — Критические задачи (Q3 2026, 8 недель)
Блок 1: Credential Storage (Безопасное хранение паролей)
CRED-01: Database Schema для credentials
Файлы: backend/internal/db/migrations/043_device_credentials.up.sql
Зависимости: Нет
Оценка: 1 день
-- Создать таблицу device_credentials
-- Поля: id, device_id (FK), username_enc (BYTEA), password_enc (BYTEA)
-- created_at, updated_at, created_by, expires_at
-- Уникальный индекс на device_id
-- Trigger для updated_at
Acceptance Criteria:
Миграция проходит без ошибок
Foreign key на devices(id) с ON DELETE CASCADE
Индекс для быстрого поиска по device_id
CRED-02: Credential Manager Interface + DB Implementation
Файлы:
backend/internal/crypto/credential_manager.go (interface)
backend/internal/crypto/db_credential_manager.go (реализация)
backend/internal/crypto/db_credential_manager_test.go
Зависимости: CRED-01
Оценка: 3 дня
Задачи:
Определить интерфейс:
type CredentialManager interface {
    Store(ctx context.Context, deviceID, username, password string) error
    Retrieve(ctx context.Context, deviceID string) (username, password string, err error)
    Rotate(ctx context.Context, deviceID, newUsername, newPassword string) error
    Delete(ctx context.Context, deviceID string) error
}
Реализовать DBCredentialManager:
Использовать существующий Encryptor (AES-256-GCM из internal/crypto/aes.go)
Шифровать username/password перед записью в БД
Дешифровать при чтении
Логировать все операции в audit_log
Написать тесты:
Test Store/Retrieve (round-trip)
Test Rotate
Test Delete
Test Retrieve non-existent device
Test encryption (проверить, что в БД хранятся зашифрованные данные)
Acceptance Criteria:
Все тесты проходят
Passwords не хранятся в открытом виде
Audit log записывает операции (кто, когда, какое устройство)
Интеграция с существующим Encryptor
CRED-03: API Endpoints для управления credentials
Файлы:
backend/internal/api/credential_handlers.go
backend/internal/api/credential_routes.go
Зависимости: CRED-02
Оценка: 2 дня
Задачи:
Реализовать endpoints:
POST   /api/v1/devices/{id}/credentials  // Сохранить credentials (admin only)
GET    /api/v1/devices/{id}/credentials  // Получить credentials (admin only, маскировать password)
PUT    /api/v1/devices/{id}/credentials  // Обновить credentials (admin only)
DELETE /api/v1/devices/{id}/credentials  // Удалить credentials (admin only)
Middleware:
Проверка роли admin (RBAC)
Валидация входных данных
Логирование в audit_log
Response format:
{
  "device_id": "cam-001",
  "username": "admin",
  "password": "****", // маскировать при GET
  "updated_at": "2026-06-30T12:00:00Z"
}
Acceptance Criteria:
Только роль admin может управлять credentials
Password маскируется в GET response
Все операции логируются
Валидация входных данных (min length, special chars)
CRED-04: Интеграция CredentialManager с VendorDevice Factory
Файлы:
backend/internal/vendor/factory.go (обновить)
backend/internal/vendor/registry.go (обновить)
Зависимости: CRED-02, VENDOR-01
Оценка: 2 дня
Задачи:
Обновить DeviceFactory.NewDevice():
func (f *DeviceFactory) NewDevice(ctx context.Context, deviceID string) (VendorDevice, error) {
    // 1. Получить метаданные устройства из БД
    device, err := f.registry.GetDeviceMeta(deviceID)
    
    // 2. Получить credentials из CredentialManager
    username, password, err := f.credentialMgr.Retrieve(ctx, deviceID)
    
    // 3. Создать VendorDevice через фабрику
    factory := f.registry.GetFactory(device.VendorType)
    return factory(device.IPAddress, username, password)
}
Обновить Agent.dispatchAction() для использования CredentialManager вместо step.Params["username"]
Acceptance Criteria:
Agent получает credentials из CredentialManager
Нет хардкода паролей в config.yaml
Fallback на step.Params для обратной совместимости
Блок 2: Vendor Abstraction Layer
VENDOR-01: VendorDevice Interface
Файлы:
backend/internal/vendor/vendor.go (interface)
backend/internal/vendor/types.go (DTOs)
Зависимости: Нет
Оценка: 2 дня
Задачи:
Определить интерфейс:
go
type VendorDevice interface {
    // Идентификация
    GetInfo(ctx context.Context) (*DeviceInfo, error)
    
    // Логи
    GetLogs(ctx context.Context, since time.Time, max int) ([]LogEntry, error)
    GetEvents(ctx context.Context, since time.Time) ([]Event, error)
    
    // Настройки
    GetSettings(ctx context.Context, category string) (map[string]interface{}, error)
    SetSettings(ctx context.Context, settings map[string]interface{}) error
    
    // Self-Healing
    HealthCheck(ctx context.Context) (*HealthStatus, error)
    Reboot(ctx context.Context) error
    FactoryReset(ctx context.Context) error
    RestoreConfig(ctx context.Context, config []byte) error
    
    // PTZ
    PTZControl(ctx context.Context, command PTZCommand) error
    
    // Async events
    SubscribeEvents(ctx context.Context) (<-chan Event, error)
}
Определить DTOs:
type DeviceInfo struct {
    Model     string
    Serial    string
    Firmware  string
    MAC       string
    Uptime    time.Duration
}

type LogEntry struct {
    Timestamp time.Time
    Level     string
    Message   string
    Source    string
}

type HealthStatus struct {
    Online     bool
    CPUUsage   float64
    MemoryFree int64
    DiskFree   int64
    Errors     []string
}
Acceptance Criteria:
Interface покрывает все существующие действия из ActionExecutor
DTOs сериализуются в JSON
Документация в godoc
VENDOR-02: Vendor Registry + Factory
Файлы:
backend/internal/vendor/registry.go
backend/internal/vendor/factory.go
backend/internal/vendor/registry_test.go
Зависимости: VENDOR-01
Оценка: 2 дня
Задачи:
Реализовать Registry:
type VendorFactory func(ip, username, password string, opts ...Option) (VendorDevice, error)

type Registry struct {
    mu        sync.RWMutex
    factories map[string]VendorFactory
}

func (r *Registry) Register(vendor string, factory VendorFactory) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.factories[vendor] = factory
}

func (r *Registry) GetFactory(vendor string) (VendorFactory, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    f, ok := r.factories[vendor]
    return f, ok
}
Реализовать Factory:
type DeviceFactory struct {
    registry      *Registry
    credentialMgr crypto.CredentialManager
    logger        *slog.Logger
}

func (f *DeviceFactory) NewDevice(ctx context.Context, deviceID string) (VendorDevice, error) {
    // Получить credentials из CredentialManager
    // Вызвать factory из registry
}
Написать тесты:
Test Register/GetFactory
Test NewDevice с валидным vendor
Test NewDevice с неизвестным vendor
Test concurrent access
Acceptance Criteria:
Registry thread-safe
Factory интегрирован с CredentialManager
Все тесты проходят
VENDOR-03: Hikvision VendorDevice Implementation
Файлы:
backend/internal/vendor/hikvision/device.go
backend/internal/vendor/hikvision/info.go
backend/internal/vendor/hikvision/logs.go
backend/internal/vendor/hikvision/settings.go
backend/internal/vendor/hikvision/healing.go
backend/internal/vendor/hikvision/device_test.go
Зависимости: VENDOR-01, VENDOR-02
Оценка: 5 дней
Задачи:
Реализовать HikvisionDevice struct:
type HikvisionDevice struct {
    ip       string
    username string
    password string
    client   *http.Client
    logger   *slog.Logger
}
Реализовать методы интерфейса:
GetInfo() → ISAPI /ISAPI/System/deviceInfo
GetLogs() → ISAPI /ISAPI/System/logs
Reboot() → ISAPI /ISAPI/System/reboot
HealthCheck() → ISAPI /ISAPI/System/status
GetSettings() → ISAPI /ISAPI/System/Network/interfaces
SetSettings() → ISAPI PUT requests
PTZControl() → ISAPI /ISAPI/PTZCtrl/channels/1/continuous
Переиспользовать существующий код из internal/protocols/hikvision/
Написать тесты:
Mock HTTP server для каждого метода
Test error handling (401, 403, 500)
Test timeout handling
Acceptance Criteria:
Все методы интерфейса реализованы
Переиспользует существующий ISAPI код
Тесты покрывают 80%+ кода
Документация в godoc
VENDOR-04: Dahua VendorDevice Implementation
Файлы:
backend/internal/vendor/dahua/device.go
backend/internal/vendor/dahua/info.go
backend/internal/vendor/dahua/logs.go
backend/internal/vendor/dahua/healing.go
backend/internal/vendor/dahua/device_test.go
Зависимости: VENDOR-01, VENDOR-02
Оценка: 4 дня
Задачи:
Реализовать DahuaDevice struct
Реализовать методы через CGI API:
GetInfo() → /cgi-bin/magicBox.cgi?action=getSystemInfo
GetLogs() → /cgi-bin/log.cgi?action=getLog
Reboot() → /cgi-bin/magicBox.cgi?action=systemReboot
HealthCheck() → /cgi-bin/global.cgi?action=getSystemInfo
Переиспользовать существующий код из internal/protocols/dahua/
Acceptance Criteria:
Все методы реализованы
Переиспользует существующий CGI код
Тесты проходят
VENDOR-05: ONVIF VendorDevice Implementation
Файлы:
backend/internal/vendor/onvif/device.go
backend/internal/vendor/onvif/ptz.go
backend/internal/vendor/onvif/device_test.go
Зависимости: VENDOR-01, VENDOR-02
Оценка: 3 дня
Задачи:
Реализовать ONVIFDevice struct
Реализовать методы через ONVIF SOAP:
GetInfo() → GetDeviceInformation
Reboot() → SystemReboot
PTZControl() → ContinuousMove
HealthCheck() → GetSystemDateAndTime
Переиспользовать существующий код из internal/protocols/onvif/
Acceptance Criteria:
Все методы реализованы
Поддержка Profile S/T
Тесты проходят
Блок 3: Protocol Descriptor System
PROTO-01: Protocol Descriptor JSON Schema
Файлы:
backend/internal/protocols/descriptor/schema.go
backend/internal/protocols/descriptor/schema.json
backend/internal/protocols/descriptor/schema_test.go
Зависимости: Нет
Оценка: 2 дня
Задачи:
Определить JSON Schema:
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "vendor": { "type": "string" },
    "version": { "type": "string" },
    "protocols": {
      "type": "object",
      "properties": {
        "http": {
          "type": "object",
          "properties": {
            "base_url": { "type": "string" },
            "auth": { "$ref": "#/definitions/auth" },
            "endpoints": { "$ref": "#/definitions/endpoints" }
          }
        }
      }
    }
  }
}
Определить Go structs:
type ProtocolDescriptor struct {
    Vendor    string                `json:"vendor"`
    Version   string                `json:"version"`
    Protocols map[string]Protocol   `json:"protocols"`
}

type Protocol struct {
    Transport string               `json:"transport"` // http, tcp, udp
    BaseURL   string               `json:"base_url"`
    Auth      AuthConfig           `json:"auth"`
    Endpoints map[string]Endpoint  `json:"endpoints"`
}

type Endpoint struct {
    Method         string         `json:"method"`
    Path           string         `json:"path"`
    Headers        map[string]string `json:"headers"`
    Body           string         `json:"body"`
    ResponseParser ResponseParser `json:"response_parser"`
}

type ResponseParser struct {
    Format   string            `json:"format"` // json, xml, key_value
    Mappings map[string]string `json:"mappings"`
}
Написать валидатор схемы
Acceptance Criteria:
JSON Schema валидируется
Go structs сериализуются/десериализуются
Примеры дескрипторов для Hikvision, Dahua
PROTO-02: Universal Protocol Interpreter
Файлы:
backend/internal/protocols/descriptor/interpreter.go
backend/internal/protocols/descriptor/interpreter_test.go
Зависимости: PROTO-01
Оценка: 5 дней
Задачи:
Реализовать ProtocolInterpreter:
type ProtocolInterpreter struct {
    httpClient *http.Client
    tcpDialer  *net.Dialer
    logger     *slog.Logger
}

func (i *ProtocolInterpreter) Execute(
    ctx context.Context,
    descriptor ProtocolDescriptor,
    endpoint string,
    params map[string]interface{},
) (*ExecutionResult, error)
Реализовать парсеры:
JSON parser (JSONPath)
XML parser (XPath)
Key-value parser (regex)
Реализовать template engine (Go templates для URL, headers, body)
Написать тесты:
Test HTTP GET/POST/PUT
Test JSON parsing
Test XML parsing
Test template rendering
Acceptance Criteria:
Интерпретатор выполняет HTTP запросы
Парсит JSON/XML/key-value ответы
Поддерживает Go templates
Тесты покрывают 80%+ кода
PROTO-03: Protocol Registry (Backend)
Файлы:
backend/internal/protocols/descriptor/registry.go
backend/internal/protocols/descriptor/registry_test.go
backend/internal/db/migrations/044_protocol_descriptors.up.sql
Зависимости: PROTO-01, PROTO-02
Оценка: 3 дня
Задачи:
Создать таблицу protocol_descriptors:
sql
CREATE TABLE protocol_descriptors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor VARCHAR(100) NOT NULL UNIQUE,
    version VARCHAR(50) NOT NULL,
    descriptor JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
Реализовать DescriptorRegistry:
type DescriptorRegistry struct {
    db     *db.DB
    cache  map[string]*ProtocolDescriptor // vendor → descriptor
    mu     sync.RWMutex
}

func (r *DescriptorRegistry) GetDescriptor(vendor string) (*ProtocolDescriptor, error)
func (r *DescriptorRegistry) SaveDescriptor(descriptor *ProtocolDescriptor) error
Реализовать кэширование в памяти
Acceptance Criteria:
Дескрипторы хранятся в БД
Кэширование в памяти
API для CRUD операций
PROTO-04: Protocol Sync API (для агента)
Файлы:
backend/internal/api/protocol_sync_handlers.go
backend/internal/api/protocol_sync_routes.go
Зависимости: PROTO-03
Оценка: 2 дня
Задачи:
Реализовать endpoint:
POST /api/v1/edge/protocols/sync
Request:
{
  "agent_id": "agent-001",
  "vendors": ["Hikvision", "Dahua"]
}
Response:
{
  "descriptors": [
    { "vendor": "Hikvision", "version": "1.0.0", ... },
    { "vendor": "Dahua", "version": "1.0.0", ... }
  ],
  "synced_at": "2026-06-30T12:00:00Z"
}
Аутентификация через mTLS (agent certificate)
Логирование запросов (какой агент, какие протоколы)
Acceptance Criteria:
API возвращает дескрипторы для запрошенных вендоров
mTLS аутентификация
Логирование запросов
Блок 4: Edge Agent (Go)
EDGE-01: Agent Core (Discovery + MQTT)
Файлы:
edge-agent/cmd/agent/main.go
edge-agent/internal/agent/agent.go
edge-agent/internal/agent/config.go
edge-agent/go.mod
Зависимости: Нет
Оценка: 5 дней
Задачи:
Создать отдельный Go module edge-agent
Реализовать основной цикл агента:
type Agent struct {
    config      *Config
    mqttClient  *mqtt.Client
    descriptorCache *DescriptorCache
    interpreter *ProtocolInterpreter
    logger      *slog.Logger
}

func (a *Agent) Run(ctx context.Context) error {
    // 1. Connect to MQTT broker (mTLS)
    // 2. Discover devices (ARP, ONVIF, SNMP)
    // 3. Sync protocol descriptors
    // 4. Start polling loop
    // 5. Handle commands from Backend
}
Реализовать конфигурацию:
type Config struct {
    AgentID        string
    MQTTBrokerURL  string
    MQTTClientCert string
    MQTTClientKey  string
    LANSubnet      string
    PollInterval   time.Duration
}
Acceptance Criteria:
Агент компилируется в один бинарник
Размер < 10 МБ (после UPX)
Подключается к MQTT broker
Логирует действия
EDGE-02: Device Discovery
Файлы:
edge-agent/internal/discovery/arp.go
edge-agent/internal/discovery/onvif.go
edge-agent/internal/discovery/snmp.go
edge-agent/internal/discovery/discovery.go
Зависимости: EDGE-01
Оценка: 4 дня
Задачи:
Реализовать ARP scanner:
func (s *ARPScanner) Scan(ctx context.Context, subnet string) ([]DiscoveredDevice, error)
Реализовать ONVIF WS-Discovery:
func (s *ONVIFScanner) Probe(ctx context.Context, ip string) (*ONVIFDevice, error)
Реализовать SNMP discovery:
func (s *SNMPScanner) Scan(ctx context.Context, subnet string) ([]DiscoveredDevice, error)
Реализовать fingerprinting (определение вендора по ответам)
Acceptance Criteria:
ARP scan находит устройства в LAN
ONVIF discovery находит IP-камеры
Fingerprinting определяет вендора
Результат публикуется в MQTT
EDGE-03: Protocol Sync + Descriptor Cache
Файлы:
edge-agent/internal/protocols/sync.go
edge-agent/internal/protocols/cache.go
Зависимости: EDGE-01, PROTO-04
Оценка: 3 дня
Задачи:
Реализовать Protocol Sync:
func (s *ProtocolSync) Sync(ctx context.Context, vendors []string) error {
    // 1. Проверить, какие дескрипторы уже есть в кэше
    // 2. Запросить недостающие с Backend
    // 3. Сохранить в кэш
}
Реализовать Descriptor Cache:
type DescriptorCache struct {
    mu          sync.RWMutex
    descriptors map[string]*ProtocolDescriptor
    maxMemory   int64
    storagePath string // USB-flash (optional)
}

func (c *DescriptorCache) Put(descriptor *ProtocolDescriptor) error
func (c *DescriptorCache) Get(vendor string) (*ProtocolDescriptor, bool)
Реализовать lazy loading (загрузка при первом обращении)
Acceptance Criteria:
Агент запрашивает только нужные дескрипторы
Кэш работает в RAM (tmpfs)
Опциональное сохранение на USB-flash
Lazy loading при первом обращении
EDGE-04: Command Handler
Файлы:
edge-agent/internal/agent/command_handler.go
Зависимости: EDGE-01, EDGE-03, PROTO-02
Оценка: 3 дня
Задачи:
Реализовать обработку команд от Backend:
func (h *CommandHandler) HandleCommand(msg *mqtt.Message) {
    var cmd struct {
        DeviceID string          `json:"device_id"`
        Action   string          `json:"action"` // reboot, get_logs, get_settings
        Params   json.RawMessage `json:"params"`
    }
    
    // 1. Найти устройство в локальной сети
    // 2. Получить дескриптор для вендора
    // 3. Выполнить команду через Interpreter
    // 4. Отправить результат обратно в Backend
}
Поддерживаемые действия:
reboot → Execute(descriptor, "reboot", params)
get_logs → Execute(descriptor, "get_logs", params)
get_settings → Execute(descriptor, "get_settings", params)
health_check → Execute(descriptor, "health_check", params)
Acceptance Criteria:
Агент выполняет команды от Backend
Результат публикуется в MQTT
Обработка ошибок (устройство недоступно, таймаут)
Логирование всех команд
EDGE-05: Telemetry Poller
Файлы:
edge-agent/internal/agent/poller.go
Зависимости: EDGE-01, EDGE-03, PROTO-02
Оценка: 2 дня
Задачи:
Реализовать периодический опрос устройств:
func (p *Poller) Run(ctx context.Context) {
    ticker := time.NewTicker(p.interval)
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            for _, device := range p.devices {
                telemetry := p.collectTelemetry(device)
                p.publishTelemetry(device, telemetry)
            }
        }
    }
}
Собирать телеметрию:
Health status (online/offline)
CPU/Memory usage (если доступно)
Uptime
Error count
Публиковать в MQTT:
Topic: edge.{agent_id}.{device_id}.telemetry
Payload:
{
  "timestamp": "2026-06-30T12:00:00Z",
  "status": "online",
  "cpu_usage": 45.2,
  "memory_free": 1024000,
  "uptime": 86400
}
Acceptance Criteria:
Периодический опрос (настраиваемый интервал)
Публикация в MQTT
Обработка ошибок (устройство недоступно)
Минимальный трафик (~1-5 КБ/мин на устройство)
EDGE-06: Offline Queue
Файлы:
edge-agent/internal/agent/offline_queue.go
Зависимости: EDGE-01
Оценка: 2 дня
Задачи:
Реализовать очередь для offline-режима:
type OfflineQueue struct {
    mu       sync.Mutex
    messages []*MQTTMessage
    maxSize  int
    storage  Storage // RAM или USB-flash
}

func (q *OfflineQueue) Push(msg *MQTTMessage) error
func (q *OfflineQueue) Pop() (*MQTTMessage, error)
func (q *OfflineQueue) Flush() error // отправить все при восстановлении связи
Использовать BoltDB для persistence (опционально на USB-flash)
Ограничить размер очереди (например, 1000 сообщений)
Acceptance Criteria:
Очередь работает при обрыве связи
Автоматическая отправка при восстановлении
Ограничение размера очереди
Persistence на USB-flash (опционально)
EDGE-07: mTLS Configuration
Файлы:
edge-agent/internal/tls/config.go
edge-agent/scripts/generate_certs.sh
Зависимости: EDGE-01
Оценка: 2 дня
Задачи:
Реализовать mTLS конфигурацию:
func NewTLSConfig(clientCert, clientKey, caCert string) (*tls.Config, error) {
    cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)
    
    return &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caCertPool,
        ClientAuth:   tls.RequireAndVerifyClientCert,
    }, nil
}
Написать скрипт для генерации сертификатов:
#!/bin/bash
# Generate CA
openssl req -x509 -newkey rsa:4096 -keyout ca-key.pem -out ca-cert.pem -days 365

# Generate agent certificate
openssl req -newkey rsa:4096 -keyout agent-key.pem -out agent-req.pem -days 365
openssl x509 -req -in agent-req.pem -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out agent-cert.pem -days 365
Документация по развертыванию сертификатов
Acceptance Criteria:
mTLS работает
Скрипт генерирует сертификаты
Документация по развертыванию
EDGE-08: OpenWrt Build Script
Файлы:
edge-agent/scripts/build_openwrt.sh
edge-agent/Dockerfile.openwrt
Зависимости: EDGE-01
Оценка: 2 дня
Задачи:
Написать скрипт для кросс-компиляции:
# Расширить matrix кросс-компиляции
GOOS=linux GOARCH=arm   GOARM=7   → Keenetic, MikroTik ARM, GL.iNet
GOOS=linux GOARCH=arm64           → MikroTik Container (RB5009, CCR)
GOOS=linux GOARCH=mipsle          → OpenWrt MT7621 (основной)
GOOS=linux GOARCH=mips            → MikroTik MIPSBE (старые)

#!/bin/bash
# Cross-compile for MIPS (MT7621)
GOOS=linux GOARCH=mipsle GOARM= go build -ldflags="-s -w" -o agent-mips
upx --best agent-mips

# Cross-compile for ARM (GL-XE300)
GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o agent-arm
upx --best agent-arm
Создать Dockerfile для сборки:
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags="-s -w" -o agent

FROM alpine:3.18
COPY --from=builder /app/agent /usr/local/bin/agent
ENTRYPOINT ["/usr/local/bin/agent"]
Документация по установке на роутер
Acceptance Criteria:
Бинарник компилируется для MIPS/ARM
Размер < 5 МБ (после UPX)
Dockerfile работает
Документация по установке
Блок 5: Unified Ingestion Layer
INGEST-01: MQTT Ingress Handler
Файлы:
backend/internal/ingestion/mqtt_ingress.go
backend/internal/ingestion/mqtt_ingress_test.go
Зависимости: PROTO-03, VENDOR-02
Оценка: 3 дня
Задачи:
Реализовать MQTT Ingress:
type MQTTIngress struct {
    broker     *nats.Conn
    normalizer *VendorNormalizer
    stateMgr   state.DeviceStateManager
    logger     *slog.Logger
}

func (m *MQTTIngress) Subscribe() error {
    _, err := m.broker.Subscribe("edge.>.>", func(msg *nats.Msg) {
        // Парсить топик: edge.{agent_id}.{device_id}.{type}
        // Нормализовать данные
        // Отправить в Event Store
    })
    return err
}
Поддерживаемые типы данных:
telemetry → обновить DeviceState
alarm → создать Alarm
log → сохранить в TimescaleDB
event → опубликовать в Event Bus
Acceptance Criteria:
Подписка на MQTT топики
Парсинг топиков
Нормализация данных
Интеграция с Event Store
INGEST-02: Vendor Normalizer
Файлы:
backend/internal/ingestion/normalizer.go
backend/internal/ingestion/normalizer_test.go
Зависимости: VENDOR-02
Оценка: 2 дня
Задачи:
Реализовать нормализатор:
type VendorNormalizer struct {
    vendorRegistry *vendor.Registry
}

func (v *VendorNormalizer) Normalize(dataType string, data []byte) (*models.Event, error) {
    // 1. Распаковать JSON (vendor, model, type, payload)
    // 2. Получить VendorDevice из registry
    // 3. Делегировать парсинг конкретному вендору
    // 4. Вернуть нормализованный Event
}
Нормализация форматов:
Hikvision ISAPI → внутренний формат
Dahua CGI → внутренний формат
ONVIF → внутренний формат
Acceptance Criteria:
Нормализует данные от разных вендоров
Использует VendorDevice для парсинга
Тесты для каждого вендора
Блок 6: API Endpoints
API-01: Device Settings Endpoints
Файлы:
backend/internal/api/device_settings_handlers.go
backend/internal/api/device_settings_routes.go
Зависимости: VENDOR-02
Оценка: 2 дня
Задачи:
Реализовать endpoints:
GET  /api/v1/devices/{id}/settings?category=network
PUT  /api/v1/devices/{id}/settings
POST /api/v1/devices/{id}/settings/apply
Использовать VendorDevice.GetSettings() / SetSettings()
Валидация входных данных
Acceptance Criteria:
GET возвращает настройки устройства
PUT обновляет настройки
POST применяет изменения
Валидация данных
API-02: Device Logs Endpoints
Файлы:
backend/internal/api/device_logs_handlers.go
backend/internal/api/device_logs_routes.go
Зависимости: VENDOR-02
Оценка: 1 день
Задачи:
Реализовать endpoint:
GET /api/v1/devices/{id}/logs?since=2026-06-01T00:00:00Z&limit=100
Использовать VendorDevice.GetLogs()
Пагинация и фильтрация
Acceptance Criteria:
GET возвращает логи устройства
Пагинация работает
Фильтрация по времени
API-03: Agent Management Endpoints
Файлы:
backend/internal/api/agent_handlers.go
backend/internal/api/agent_routes.go
Зависимости: EDGE-01
Оценка: 2 дня
Задачи:
Реализовать endpoints:
GET    /api/v1/agents                    // Список всех агентов
GET    /api/v1/agents/{id}               // Детали агента
POST   /api/v1/agents/{id}/command       // Отправить команду
DELETE /api/v1/agents/{id}               // Удалить агента
Таблица agents:
CREATE TABLE agents (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255),
    site_id UUID REFERENCES sites(id),
    status VARCHAR(50), -- online, offline, error
    last_seen TIMESTAMPTZ,
    version VARCHAR(50),
    config JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
Acceptance Criteria:
CRUD операции для агентов
Отправка команд через MQTT
Статус агента (online/offline)
🟡 P1 — Высокий приоритет (Q4 2026, 6 недель)
Tiandy Support
VENDOR-06: Tiandy VendorDevice Implementation (3 дня)
ONVIF + HTTP API
Логи через Syslog
Reboot через ONVIF
Uniview Support
VENDOR-07: Uniview VendorDevice Implementation (3 дня)
ONVIF + HTTP API
Логи через Syslog
Reboot через ONVIF
Tantos Support
VENDOR-08: Tantos VendorDevice Implementation (2 дня)
ONVIF базовая поддержка
Reboot через ONVIF
Lua Plugin System
PROTO-05: Lua Plugin Loader (4 дня)
Интеграция gopher-lua
API для плагинов (http_get, http_post, xml_parse)
Загрузка плагинов из /usb/plugins/
Примеры плагинов для сложных протоколов
Edge Agent Advanced Features
EDGE-09: Traffic Shaping (2 дня)
Приоритизация телеметрии над диагностикой
QoS для MQTT
EDGE-10: OTA Updates (3 дня)
Автоматическая проверка обновлений
Download + install
Rollback при ошибке
EDGE-11: Agent Monitoring Dashboard (3 дня)
UI для мониторинга агентов
Статистика (online/offline, трафик, ошибки)
Алерты при проблемах
🟢 P2 — Средний приоритет (Q1 2027, 4 недели)
Protocol Descriptor Editor UI
PROTO-06: Web UI для редактирования дескрипторов (5 дней)
Форма для создания/редактирования
Валидация JSON Schema
Тестирование дескрипторов
Community Protocol Registry
PROTO-07: Public Protocol Registry (4 дня)
Публичный API для обмена дескрипторами
Рейтинги и отзывы
Модерация
Credential Rotation
CRED-05: Automatic Credential Rotation (3 дня)
Ротация паролей (для поддерживающих вендоров)
Уведомления перед истечением
Интеграция с Vault
Advanced Discovery
EDGE-12: mDNS/SSDP Discovery (2 дня)
Обнаружение IoT устройств
Интеграция с существующим discovery
Зависимости между задачами
CRED-01 → CRED-02 → CRED-03 → CRED-04
                  ↘
VENDOR-01 → VENDOR-02 → VENDOR-03 (Hikvision)
                  ↓
            VENDOR-04 (Dahua)
                  ↓
            VENDOR-05 (ONVIF)
                  ↓
PROTO-01 → PROTO-02 → PROTO-03 → PROTO-04
                              ↓
EDGE-01 → EDGE-02 → EDGE-03 → EDGE-04
                  ↓
            EDGE-05 (Poller)
                  ↓
            EDGE-06 (Offline)
                  ↓
            EDGE-07 (mTLS)
                  ↓
            EDGE-08 (Build)
                              ↓
INGEST-01 → INGEST-02
                  ↓
API-01, API-02, API-03
Acceptance Criteria (глобальные)
Безопасность (ISO 27001)
Credentials зашифрованы (AES-256-GCM)
mTLS для агент ↔ Backend
Audit log для всех операций
RBAC для доступа к credentials
Производительность
Размер агента < 10 МБ
RAM потребление < 50 МБ
Трафик < 5 КБ/мин на устройство
Время отклика API < 500 мс
Масштабируемость
Поддержка 1000+ агентов
Поддержка 10000+ устройств
Lazy loading протоколов
Кэширование дескрипторов
Расширяемость
Добавление нового вендора за 1-3 дня
Декларативные протоколы (JSON)
Lua-плагины для сложных случаев
Единый интерфейс VendorDevice
Следующие шаги для Roo
Начать с CRED-01 (Database Schema) — это фундамент для всего
Параллельно VENDOR-01 (Interface) — можно делать независимо
Затем CRED-02 + VENDOR-02 — основа для интеграции
Потом VENDOR-03/04/05 — реализации для конкретных вендоров
PROTO-01/02/03 — система дескрипторов
EDGE-01/02/03 — ядро агента
INGEST-01/02 — интеграция с Backend
API-01/02/03 — endpoints для UI
Каждая задача должна:
Иметь feature branch: feature/P0-CRED-01-credential-storage
Покрываться тестами (unit + integration)
Иметь документацию в godoc
Проходить code review
Обновлять TODO после merge