# План реагирования на инциденты (Incident Response Plan)
# CCTV Intelligence Platform v6.0 — ISO 27001:2022 A.16

**Версия:** 1.0
**Дата утверждения:** 2026-06-21
**Владелец:** CISO / Security Team

---

## 1. Команда реагирования

| Роль | Ответственный | Контакты |
|------|--------------|----------|
| Incident Commander | CISO | slack: #security-incidents |
| Технический лид | Lead Backend Engineer | |
| Коммуникации | PR / Compliance | |
| Восстановление | DevOps Lead | |

## 2. Классификация инцидентов

| Severity | Описание | Время реакции | Эскалация |
|----------|----------|---------------|-----------|
| **P1 — Critical** | Утечка данных, компрометация системы, активная атака | 15 минут | CISO + CEO |
| **P2 — High** | Отказ критического сервиса, подозрительная активность | 1 час | CISO |
| **P3 — Medium** | Попытка несанкционированного доступа, сбой не-critical | 4 часа | Security Team |
| **P4 — Low** | Minor policy violation, просроченный сертификат | 24 часа | Engineering |

## 3. Процедура реагирования

### 3.1 Обнаружение

Источники алертов:
- Self-Healing Agent (аномалии телеметрии)
- Audit log integrity check (HMAC mismatch)
- CMMS webhook signature failures
- Системные метрики (CPU, память, диск)
- NATS Event Bus (security events)
- Внешние уведомления (CERT, клиенты)

### 3.2 Сдерживание (Containment) — P1/P2

1. **Изолировать скомпрометированный компонент**
   - Отключить скомпрометированные API keys
   - Заблокировать IP через firewall
   - Остановить подозрительные процессы
2. **Сохранить evidence**
   - Дамп audit_log за последние 24 часа
   - Логи приложения (JSON)
   - Дамп базы данных (snapshot)
3. **Активировать резервный канал**
   - Переключить трафик на standby инстанс

### 3.3 Расследование (Investigation)

1. Анализ audit_log через `GET /api/v1/audit/verify` — проверка целостности
2. Поиск аномалий в логах: `grep ERROR /var/log/gb-telemetry/*.log`
3. Проверка CMMS webhook логов на предмет поддельных запросов
4. Анализ NATS security events

### 3.4 Устранение (Eradication)

1. Применить патч безопасности
2. Сбросить скомпрометированные ключи/токены
3. Обновить зависимости с уязвимостями
4. Пересобрать Docker-образы

### 3.5 Восстановление (Recovery)

1. Восстановить БД из бэкапа (если необходимо)
2. Проверить целостность данных
3. Перезапустить сервисы
4. Верифицировать работоспособность через health-check

### 3.6 Post-mortem

1. Задокументировать timeline инцидента
2. Определить root cause
3. Обновить security policy и процедуры
4. Провести review с командой

## 4. План восстановления (BCP)

### 4.1 RPO / RTO

| Компонент | RPO | RTO |
|-----------|-----|-----|
| PostgreSQL | 24 часа (daily pg_dump) | 4 часа |
| TimescaleDB (telemetry) | 0 (можно потерять) | 1 час |
| TimescaleDB (alarms) | 0 (можно потерять) | 1 час |
| Конфигурация | 1 час (git) | 15 минут |

### 4.2 Процедура восстановления БД

```bash
# 1. Остановить приложение
systemctl stop gb-telemetry

# 2. Восстановить из бэкапа
pg_restore -h $DB_HOST -U $DB_USER -d $DB_NAME /backups/latest.dump

# 3. Проверить целостность
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT count(*) FROM audit_log WHERE hmac_signature IS NULL"

# 4. Запустить приложение
systemctl start gb-telemetry
```

### 4.3 Восстановление из Docker

```bash
docker-compose -f docker-compose.yml -f docker-compose.nats.yml up -d
docker-compose logs -f backend
```

## 5. Контакты экстренной связи

| Ресурс | Контакт |
|--------|---------|
| Security Team Slack | #security-incidents |
| On-call телефон | +7 (XXX) XXX-XX-XX |
| CISO email | ciso@company.com |
| CERT | cert@company.com |

## 6. Обновление плана

План пересматривается:
- После каждого инцидента (post-mortem)
- Ежегодно (плановая ревизия)
- При изменении архитектуры системы

---

**Дата следующего пересмотра:** 2027-06-21