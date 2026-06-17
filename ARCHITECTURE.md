# CCTV Health Monitor — Architecture Documentation

> **Версия**: 1.0  
> **Последнее обновление**: 2026-06-17  
> **Автор**: System Architect  
> **Статус**: Active Development

---

## 📋 Оглавление

1. [Обзор системы](#1-обзор-системы)
2. [High-Level Architecture](#2-high-level-architecture)
3. [Компоненты и сервисы](#3-компоненты-и-сервисы)
4. [Потоки данных](#4-потоки-данных)
5. [Протокольный зоопарк](#5-протокольный-зоопарк)
6. [P2P туннелирование](#6-p2p-туннелирование)
7. [ML-аналитика](#7-ml-аналитика)
8. [База данных](#8-база-данных)
9. [Безопасность](#9-безопасность)
10. [Deployment & Roadmap](#10-deployment--roadmap)

---

## 1. Обзор системы

**CCTV Health Monitor (gb-telemetry-collector)** — enterprise-платформа для мониторинга здоровья систем видеонаблюдения. Собирает телеметрию с DVR/NVR/IP-камер через 8+ протоколов, анализирует данные с помощью ML (XGBoost) и предоставляет unified dashboard для инженеров и менеджеров.

**Ключевые возможности**:
- Агрегация событий с камер (motion, video loss, HDD errors, tamper)
- P2P-туннелирование через NAT (Dahua PTCP, Hikvision Cloud, Xiongmai Jftech)
- Прогнозирование отказов оборудования (failure probability + DeepSeek LLM explanations)
- Ticketing system с RBAC (6 ролей)
- Multi-tenant ready (planned)

---

## 2. High-Level Architecture

```mermaid
flowchart TB
    subgraph Edge["🎥 Edge Layer (Devices)"]
        direction LR
        CAM1[IP Camera<br/>Hikvision/Dahua]
        CAM2[NVR/DVR<br/>TVT/Hisilicon]
        CAM3[P2P Camera<br/>Reolink/Xiongmai]
        CAM4[IoT Sensor<br/>SNMP]
    end

    subgraph Core["⚙️ Core Services"]
        direction TB
        BACKEND[Backend API<br/>Go 1.25 + Chi]
        P2P[P2P Gateway<br/>Go 1.25]
        DHP2P[DH-P2P PoC<br/>Rust + Python]
        ANALYTICS[Analytics Engine<br/>Python 3.11]
    end

    subgraph Storage["💾 Storage Layer"]
        direction LR
        TSDB[(TimescaleDB<br/>Timeseries)]
        PG[(PostgreSQL<br/>Metadata)]
        FS[(Filesystem<br/>Images/Logs)]
    end

    subgraph Clients["👥 Client Layer"]
        direction LR
        WEB[Web Dashboard<br/>React 19 + Vite]
        MOB[Mobile App<br/>Planned]
        API_EXT[External API<br/>Integrations]
    end

    CAM1 -->|RTSP/ISAPI/SIP| BACKEND
    CAM2 -->|Private TCP<br/>Dahua/Hisilicon/TVT| BACKEND
    CAM3 -->|PTCP over UDP| DHP2P
    DHP2P -->|RTSP tunnel| P2P
    P2P -->|Proxy| WEB
    CAM4 -->|SNMP Traps| BACKEND
    
    BACKEND --> TSDB
    BACKEND --> PG
    BACKEND --> FS
    
    ANALYTICS -->|ETL| TSDB
    ANALYTICS -->|Predictions| PG
    
    WEB -->|REST + JWT| BACKEND
    WEB -->|P2P Stream| P2P
    API_EXT -->|Webhook| BACKEND