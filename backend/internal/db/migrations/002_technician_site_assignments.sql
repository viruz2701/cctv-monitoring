-- Technician Site Assignments Migration
-- Закрепление техников за объектами

-- Таблица закрепления техников за объектами
CREATE TABLE IF NOT EXISTS technician_site_assignments (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    technician_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    site_id TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    is_primary BOOLEAN DEFAULT false, -- основной техник для объекта
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    assigned_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE(technician_id, site_id) -- один техник может быть закреплен за объектом только один раз
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_technician_site_assignments_technician ON technician_site_assignments(technician_id);
CREATE INDEX IF NOT EXISTS idx_technician_site_assignments_site ON technician_site_assignments(site_id);
CREATE INDEX IF NOT EXISTS idx_technician_site_assignments_primary ON technician_site_assignments(site_id, is_primary);

-- Комментарии
COMMENT ON TABLE technician_site_assignments IS 'Закрепление техников за объектами';
COMMENT ON COLUMN technician_site_assignments.is_primary IS 'Основной техник для объекта (ответственный)';
