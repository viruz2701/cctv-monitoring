-- Migration 060: Materialized View Auto-Refresh via pg_cron
-- P1-HI-02: REFRESH MATERIALIZED VIEW CONCURRENTLY для всех matview
--
-- Что делаем:
--   1. Добавляем refresh_tco_per_device() для mv_tco_per_device (аналог refresh_device_reliability из 018)
--   2. Подключаем pg_cron и настраиваем автоматическое обновление обоих matview
--
-- Compliance:
--   - IEC 62443 SR 7.1 (Resource availability — актуальные метрики)
--   - ISO 27001 A.12.6.1 (Capacity management — своевременное обновление)
--   - СТБ 34.101.27 п. 7.3 (Анализ защищённости — актуальность данных)
--
-- ВАЖНО: pg_cron требует superuser privileges.
-- Если pg_cron не установлен, выполните от имени superuser:
--   CREATE EXTENSION IF NOT EXISTS pg_cron;
-- Если pg_cron не доступен — удалите EXTENSION часть, оставив только функцию.
-- Автообновление также работает через Go-тикер в maintenance_cron.go (P3-2.1).
-- +migrate Up

-- =============================================================
-- 1. Функция обновления mv_tco_per_device
-- =============================================================
-- Аналог refresh_device_reliability() из миграции 018.
-- Использует CONCURRENTLY для неблокирующего обновления.

CREATE OR REPLACE FUNCTION refresh_tco_per_device()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_tco_per_device;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION refresh_tco_per_device() IS
    'AN-10.1.3: Обновляет mv_tco_per_device конкурентно (без блокировок чтения). '
    'Использует UNIQUE INDEX idx_mv_tco_device для CONCURRENTLY.';

-- =============================================================
-- 2. pg_cron — автоматическое обновление matview
-- =============================================================
-- Обновляем каждые 60 минут, в разное время чтобы избежать contention:
--   mv_device_reliability — в 0 минут каждого часа
--   mv_tco_per_device    — в 30 минут каждого часа
--
-- pg_cron.create_job или cron.schedule в зависимости от версии.
-- Используем cron.schedule (pg_cron >= 1.4).

-- Проверяем, что pg_cron установлен, через блок DO
DO $$
BEGIN
    -- Если расширение не установлено — пропускаем, не роняем миграцию
    IF EXISTS (
        SELECT 1 FROM pg_available_extensions WHERE name = 'pg_cron' AND installed_version IS NOT NULL
    ) THEN
        -- Задача для mv_device_reliability (каждый час в 0 минут)
        PERFORM cron.schedule(
            'refresh-mv-device-reliability',  -- job name
            '0 * * * *',                       -- каждый час
            $$SELECT refresh_device_reliability()$$
        );

        -- Задача для mv_tco_per_device (каждый час в 30 минут)
        PERFORM cron.schedule(
            'refresh-mv-tco-per-device',       -- job name
            '30 * * * *',                      -- каждый час в 30 минут
            $$SELECT refresh_tco_per_device()$$
        );
    ELSE
        RAISE WARNING 'pg_cron extension not available. '
                      'Materialized views will be refreshed via Go ticker (maintenance_cron.go). '
                      'Install pg_cron manually: CREATE EXTENSION IF NOT EXISTS pg_cron;';
    END IF;
END;
$$;

-- Комментарии к задачам (в information_schema не хранятся, но документируем здесь)
COMMENT ON FUNCTION refresh_device_reliability() IS
    'AN-10.1.1: pg_cron: каждый час — REFRESH MATERIALIZED VIEW CONCURRENTLY mv_device_reliability';

COMMENT ON FUNCTION refresh_tco_per_device() IS
    'AN-10.1.3: pg_cron: каждый час в 30 минут — REFRESH MATERIALIZED VIEW CONCURRENTLY mv_tco_per_device';
