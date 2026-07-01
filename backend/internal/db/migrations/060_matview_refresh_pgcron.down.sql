-- Migration 060: Down — откат pg_cron задач и refresh-функции
-- +migrate Down

-- =============================================================
-- 1. Удаление pg_cron задач
-- =============================================================
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_available_extensions WHERE name = 'pg_cron' AND installed_version IS NOT NULL
    ) THEN
        -- cron.unschedule(job_name) доступен в pg_cron >= 1.4
        PERFORM cron.unschedule('refresh-mv-device-reliability');
        PERFORM cron.unschedule('refresh-mv-tco-per-device');
    END IF;
END;
$$;

-- =============================================================
-- 2. Удаление функции refresh_tco_per_device()
-- =============================================================
DROP FUNCTION IF EXISTS refresh_tco_per_device();
