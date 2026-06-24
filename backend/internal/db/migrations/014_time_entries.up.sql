-- Migration 014: Time Entries for Work Orders (WO-4.4.1, WO-4.4.2)
-- Добавляет таблицу time_entries для учёта времени и labour cost.
-- Соответствует: ISO 27001 A.12.4.1 (Event logging), IEC 62443 SR 2.8
-- +migrate Up

-- 1. Time Entries (WO-4.4.1)
-- Таблица может уже существовать (создана в 008), добавляем новые колонки
ALTER TABLE IF EXISTS time_entries ADD COLUMN IF NOT EXISTS start_time TIMESTAMPTZ;
ALTER TABLE IF EXISTS time_entries ADD COLUMN IF NOT EXISTS end_time TIMESTAMPTZ;
ALTER TABLE IF EXISTS time_entries ADD COLUMN IF NOT EXISTS paused_duration BIGINT DEFAULT 0;
ALTER TABLE IF EXISTS time_entries ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'running';
ALTER TABLE IF EXISTS time_entries ADD COLUMN IF NOT EXISTS notes TEXT DEFAULT '';
ALTER TABLE IF EXISTS time_entries ADD COLUMN IF NOT EXISTS hourly_rate DECIMAL(10,2) DEFAULT 0;
ALTER TABLE IF EXISTS time_entries ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- 2. Labor Costs (WO-4.4.2) + Cost tracking (WO-4.4.4, WO-4.4.5)
ALTER TABLE work_orders ADD COLUMN IF NOT EXISTS total_labor_cost DECIMAL(12,2) DEFAULT 0;
ALTER TABLE work_orders ADD COLUMN IF NOT EXISTS total_labor_seconds BIGINT DEFAULT 0;
ALTER TABLE work_orders ADD COLUMN IF NOT EXISTS total_parts_cost DECIMAL(12,2) DEFAULT 0;
ALTER TABLE work_orders ADD COLUMN IF NOT EXISTS total_cost DECIMAL(12,2) DEFAULT 0;

-- 3. Индексы
CREATE INDEX IF NOT EXISTS idx_time_entries_work_order ON time_entries(work_order_id);
CREATE INDEX IF NOT EXISTS idx_time_entries_user ON time_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_time_entries_status ON time_entries(status);
CREATE INDEX IF NOT EXISTS idx_work_orders_total_cost ON work_orders(total_cost);
