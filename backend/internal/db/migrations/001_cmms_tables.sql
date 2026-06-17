-- CMMS Tables Migration
-- Maintenance Schedules, Work Orders, Spare Parts, SLA Configuration

-- 1. Maintenance Schedules (графики планового ТО)
CREATE TABLE IF NOT EXISTS maintenance_schedules (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    device_id TEXT REFERENCES devices(device_id) ON DELETE CASCADE,
    schedule_type TEXT NOT NULL CHECK (schedule_type IN ('daily', 'weekly', 'monthly', 'quarterly', 'custom')),
    interval_days INT,
    custom_cron TEXT, -- для сложных расписаний
    last_completed TIMESTAMPTZ,
    next_due TIMESTAMPTZ NOT NULL,
    assigned_to TEXT REFERENCES users(id) ON DELETE SET NULL,
    checklist JSONB NOT NULL DEFAULT '[]', -- массив задач: [{task: "Проверить HDD", completed: false}]
    estimated_minutes INT DEFAULT 30,
    priority TEXT DEFAULT 'medium' CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Work Orders (наряды на работу)
CREATE TABLE IF NOT EXISTS work_orders (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    schedule_id TEXT REFERENCES maintenance_schedules(id) ON DELETE SET NULL, -- если плановое ТО
    device_id TEXT REFERENCES devices(device_id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('preventive', 'corrective', 'emergency')),
    status TEXT DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'completed', 'cancelled')),
    priority TEXT DEFAULT 'medium' CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    assigned_to TEXT REFERENCES users(id) ON DELETE SET NULL,
    sla_deadline TIMESTAMPTZ,
    checklist JSONB NOT NULL DEFAULT '[]',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    notes TEXT,
    photos JSONB DEFAULT '[]', -- массив URL фотографий
    parts_used JSONB DEFAULT '[]', -- использованные запчасти
    created_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Technician Skills & Workload
ALTER TABLE users ADD COLUMN IF NOT EXISTS skills TEXT[] DEFAULT '{}';
ALTER TABLE users ADD COLUMN IF NOT EXISTS max_workload INT DEFAULT 5;
ALTER TABLE users ADD COLUMN IF NOT EXISTS current_workload INT DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS base_location TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS certifications TEXT[] DEFAULT '{}';

-- 4. SLA Configuration
CREATE TABLE IF NOT EXISTS sla_config (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    priority TEXT NOT NULL CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    response_time_minutes INT NOT NULL, -- время реакции
    resolution_time_minutes INT NOT NULL, -- время решения
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 5. Spare Parts Inventory
CREATE TABLE IF NOT EXISTS spare_parts (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name TEXT NOT NULL,
    sku TEXT UNIQUE,
    category TEXT,
    stock INT DEFAULT 0,
    min_stock INT DEFAULT 5,
    location TEXT,
    compatible_devices TEXT[], -- модели устройств
    cost DECIMAL(10, 2),
    supplier TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 6. Part Usage Log
CREATE TABLE IF NOT EXISTS part_usage (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    work_order_id TEXT REFERENCES work_orders(id) ON DELETE CASCADE,
    part_id TEXT REFERENCES spare_parts(id) ON DELETE CASCADE,
    quantity INT NOT NULL,
    used_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    used_at TIMESTAMPTZ DEFAULT NOW()
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_device ON maintenance_schedules(device_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_next_due ON maintenance_schedules(next_due);
CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_assigned ON maintenance_schedules(assigned_to);
CREATE INDEX IF NOT EXISTS idx_work_orders_device ON work_orders(device_id);
CREATE INDEX IF NOT EXISTS idx_work_orders_status ON work_orders(status);
CREATE INDEX IF NOT EXISTS idx_work_orders_assigned ON work_orders(assigned_to);
CREATE INDEX IF NOT EXISTS idx_work_orders_sla ON work_orders(sla_deadline);
CREATE INDEX IF NOT EXISTS idx_spare_parts_sku ON spare_parts(sku);
CREATE INDEX IF NOT EXISTS idx_part_usage_work_order ON part_usage(work_order_id);

-- Дефолтные SLA
INSERT INTO sla_config (priority, response_time_minutes, resolution_time_minutes) VALUES
('critical', 15, 60),
('high', 30, 240),
('medium', 60, 480),
('low', 240, 1440)
ON CONFLICT DO NOTHING;
