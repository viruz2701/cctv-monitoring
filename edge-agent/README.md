# Edge Agent

Легковесный Go-агент для запуска на OpenWrt роутерах (128MB RAM) в составе CCTV Health Monitor.

## Архитектура

```
edge-agent/
├── cmd/agent/main.go           # Точка входа
├── internal/
│   ├── agent/
│   │   ├── agent.go            # Основной цикл агента
│   │   ├── config.go           # Конфигурация (env vars)
│   │   ├── command_handler.go  # Обработка команд от Backend
│   │   ├── poller.go           # Периодический сбор телеметрии
│   │   └── offline_queue.go    # Оффлайн-очередь (BoltDB)
│   ├── discovery/
│   │   ├── discovery.go        # Интерфейс Scanner + оркестратор
│   │   ├── arp.go              # ARP scanner (/proc/net/arp)
│   │   ├── onvif.go            # ONVIF WS-Discovery (SOAP/UDP)
│   │   └── snmp.go             # SNMP discovery (raw UDP, без gosnmp)
│   ├── protocols/
│   │   ├── cache.go            # Descriptor Cache (RAM + USB)
│   │   └── sync.go             # Protocol Sync с Backend
│   └── tls/
│       └── config.go           # mTLS конфигурация
├── scripts/
│   ├── generate_certs.sh       # Генерация сертификатов
│   └── build_openwrt.sh        # Кросс-компиляция для OpenWrt
├── Dockerfile.openwrt
├── go.mod
└── README.md
```

## Функциональность

| Компонент | Описание |
|-----------|----------|
| **Discovery** | ARP scan, ONVIF WS-Discovery, SNMP — без внешних зависимостей |
| **Protocol Sync** | Синхронизация Protocol Descriptor'ов с Backend через HTTP (digest auth) |
| **Command Handler** | Обработка MQTT-команд от Backend (reboot, sync, discover, exec) |
| **Poller** | Периодический сбор health-телеметрии с устройств |
| **Offline Queue** | BoltDB-очередь для сообщений при потере MQTT-связи |
| **mTLS** | TLS 1.3 с взаимной аутентификацией для всех соединений |

## Зависимости

- [paho.mqtt.golang](https://github.com/eclipse/paho.mqtt.golang) — MQTT клиент
- [icholy/digest](https://github.com/icholy/digest) — Digest authentication
- [bbolt](https://go.etcd.io/bbolt) — Embedded key-value store

Остальной код — только стандартная библиотека Go.

## Конфигурация

Все параметры через переменные окружения (префикс `EDGE_AGENT_`):

| Переменная | Обязательно | Описание |
|-----------|-------------|----------|
| `AGENT_ID` | ✅ | Уникальный ID агента |
| `MQTT_BROKER_URL` | ✅ | URL MQTT брокера (tls://...) |
| `LAN_SUBNET` | ✅ | Подсеть для discovery (192.168.1.0/24) |
| `BACKEND_URL` | ✅ | URL Backend API |
| `BACKEND_USER` | ✅ | Пользователь для digest auth |
| `BACKEND_PASSWORD` | ✅ | Пароль для digest auth |
| `MQTT_CERT` | ✅ | Путь к клиентскому сертификату |
| `MQTT_KEY` | ✅ | Путь к ключу клиента |
| `MQTT_CA` | ✅ | Путь к CA сертификату |
| `OFFLINE_QUEUE_PATH` | ❌ | Путь к BoltDB файлу (/tmp/edge-agent-queue.db) |
| `CACHE_PATH` | ❌ | Путь к кэшу дескрипторов |
| `POLL_INTERVAL` | ❌ | Интервал телеметрии (30s) |
| `SYNC_INTERVAL` | ❌ | Интервал синхронизации (300s) |
| `DISCOVERY_INTERVAL` | ❌ | Интервал discovery (600s) |
| `LOG_LEVEL` | ❌ | debug/info/warn/error |

## Сборка

### Локальная сборка

```bash
go build -o build/edge-agent ./cmd/agent
```

### Кросс-компиляция для OpenWrt

```bash
# Все цели
./scripts/build_openwrt.sh all

# Конкретная цель
./scripts/build_openwrt.sh mips    # MT7620/MT7628
./scripts/build_openwrt.sh arm     # ARMv7
./scripts/build_openwrt.sh arm64   # ARMv8
./scripts/build_openwrt.sh amd64   # x86_64
```

### Docker сборка

```bash
docker build -t edge-agent-builder -f Dockerfile.openwrt .
docker run --rm -v $(pwd)/build:/build edge-agent-builder
```

## Генерация сертификатов

```bash
./scripts/generate_certs.sh
```

В production используйте PKI инфраструктуру.

## Compliance

| Стандарт | Уровень | Применение |
|----------|---------|------------|
| IEC 62443-3-3 | SL-3 | Zone 5 (Edge), шифрование каналов, контроль доступа |
| Приказ ОАЦ №66 | п. 7.18 | mTLS, уникальная идентификация, tamper detection |
| OWASP ASVS | L3 | Input validation, error handling, audit trail |
| ISO 27001 | A.13.1 | Защита сетей и каналов связи |
| ISO 27019 | PCC.A.13 | Безопасность ICS/SCADA |

## Лицензия

CCTV Health Monitor — КИИ РБ, класс KII-2
