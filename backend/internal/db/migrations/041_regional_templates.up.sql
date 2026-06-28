-- +migrate Up
-- P2-REG.8: Regional Maintenance Templates — TR, VN, ID, BR, ZA
--
-- Создаёт таблицы maintenance_regulations и maintenance_checklists
-- с pre-loaded шаблонами для 5 регионов.
--
-- Compliance:
--   - KVKK №6698 (TR) — Закон о защите персональных данных Турции
--   - TS EN 62676 (TR) — CCTV стандарт Турции
--   - TCVN 11930:2017 (VN) — Информационная безопасность Вьетнама
--   - Camera Standard 2025 (VN) — CCTV стандарт Вьетнама
--   - SNI 27001 (ID) — ISMS стандарт Индонезии
--   - UU PDP 2022 (ID) — Закон о защите ПД Индонезии
--   - ABNT NBR series (BR) — CCTV стандарт Бразилии
--   - LGPD (BR) — Закон о защите ПД Бразилии
--   - SANS 10160-4 (ZA) — Безопасность объектов ЮАР
--   - POPIA (ZA) — Закон о защите ПД ЮАР
--   - ISO 27001 A.12.4 (Audit trail)
--   - IEC 62443-3-3 SL-3 (Zone 3 — Application integrity)

-- ═══════════════════════════════════════════════════════════════════════════
-- P2-REG.8.1: Таблица maintenance_regulations — шаблоны регламентов ТО
-- ═══════════════════════════════════════════════════════════════════════════

CREATE TABLE maintenance_regulations (
    id                  TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    region_code         VARCHAR(2) NOT NULL,
    regulation_code     VARCHAR(20) NOT NULL UNIQUE,
    name                TEXT NOT NULL,
    regulation_type     VARCHAR(4) NOT NULL CHECK (regulation_type IN ('TO-1', 'TO-2', 'TO-3')),
    interval_months     INT NOT NULL CHECK (interval_months > 0),
    estimated_minutes   INT NOT NULL CHECK (estimated_minutes > 0),
    total_items         INT NOT NULL CHECK (total_items > 0),
    compliance_standards TEXT[] NOT NULL DEFAULT '{}',
    license_requirements TEXT,
    docs_required       JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_maint_reg_region
    ON maintenance_regulations (region_code);
CREATE INDEX IF NOT EXISTS idx_maint_reg_type
    ON maintenance_regulations (regulation_type);
CREATE INDEX IF NOT EXISTS idx_maint_reg_active
    ON maintenance_regulations (is_active)
    WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_maint_reg_region_type
    ON maintenance_regulations (region_code, regulation_type);

COMMENT ON TABLE maintenance_regulations IS
    'P2-REG.8: Pre-loaded maintenance regulation templates per region. '
    'Содержит регламенты ТО для BY, RU, KZ (ранее) и TR, VN, ID, BR, ZA.';

COMMENT ON COLUMN maintenance_regulations.region_code IS
    'ISO 3166-1 alpha-2 код региона (TR, VN, ID, BR, ZA)';
COMMENT ON COLUMN maintenance_regulations.regulation_code IS
    'Уникальный код регламента, например TR-KVKK-TO1';
COMMENT ON COLUMN maintenance_regulations.regulation_type IS
    'Тип ТО: TO-1 (базовое), TO-2 (расширенное), TO-3 (комплексное)';
COMMENT ON COLUMN maintenance_regulations.compliance_standards IS
    'Массив применимых стандартов compliance';
COMMENT ON COLUMN maintenance_regulations.license_requirements IS
    'Требования к лицензии/регистрации для выполнения работ';
COMMENT ON COLUMN maintenance_regulations.docs_required IS
    'Массив обязательных документов для оформления';

-- ═══════════════════════════════════════════════════════════════════════════
-- P2-REG.8.2: Таблица maintenance_checklists — пункты чек-листов ТО
-- ═══════════════════════════════════════════════════════════════════════════

CREATE TABLE maintenance_checklists (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    regulation_id   TEXT NOT NULL REFERENCES maintenance_regulations(id) ON DELETE CASCADE,
    item_order      INT NOT NULL CHECK (item_order > 0),
    description     TEXT NOT NULL,
    category        VARCHAR(50) NOT NULL DEFAULT 'inspection'
                    CHECK (category IN (
                        'inspection', 'cleaning', 'test', 'network',
                        'firmware', 'storage', 'power', 'mounting',
                        'cabling', 'documentation', 'compliance',
                        'security', 'backup', 'performance',
                        'calibration', 'certification'
                    )),
    is_required     BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_maint_checklist_reg
    ON maintenance_checklists (regulation_id);
CREATE INDEX IF NOT EXISTS idx_maint_checklist_order
    ON maintenance_checklists (regulation_id, item_order);
CREATE INDEX IF NOT EXISTS idx_maint_checklist_category
    ON maintenance_checklists (category);

COMMENT ON TABLE maintenance_checklists IS
    'P2-REG.8: Пункты чек-листов для регламентов ТО. '
    'Связаны с maintenance_regulations через regulation_id.';

COMMENT ON COLUMN maintenance_checklists.category IS
    'Категория пункта: inspection, cleaning, test, network, firmware, '
    'storage, power, mounting, cabling, documentation, compliance, '
    'security, backup, performance, calibration, certification';

-- ═══════════════════════════════════════════════════════════════════════════
-- P2-REG.8.3: Pre-loaded данные — TR (Turkey: KVKK + TS EN 62676)
-- ═══════════════════════════════════════════════════════════════════════════

-- TR: TO-1 (3 мес, 10 пунктов, 60 мин)
INSERT INTO maintenance_regulations (
    region_code, regulation_code, name, regulation_type,
    interval_months, estimated_minutes, total_items,
    compliance_standards, license_requirements, docs_required
) VALUES (
    'TR', 'TR-KVKK-TO1', 'KVKK + TS EN 62676 — ТО-1 (Ежеквартальное)',
    'TO-1', 3, 60, 10,
    ARRAY['KVKK №6698', 'TS EN 62676'],
    'KVKK VERBIS registration',
    '["KVKK compliance report (quarterly)", "TS EN 62676 checklist"]'::jsonb
);

WITH tr_to1 AS (SELECT id FROM maintenance_regulations WHERE regulation_code = 'TR-KVKK-TO1')
INSERT INTO maintenance_checklists (regulation_id, item_order, description, category) VALUES
    ((SELECT id FROM tr_to1), 1,  'Проверка работоспособности всех камер', 'inspection'),
    ((SELECT id FROM tr_to1), 2,  'Очистка объективов и корпусов камер', 'cleaning'),
    ((SELECT id FROM tr_to1), 3,  'Проверка качества записи и архивации', 'storage'),
    ((SELECT id FROM tr_to1), 4,  'Проверка угла обзора и фокусировки', 'calibration'),
    ((SELECT id FROM tr_to1), 5,  'Проверка питания и PoE-коммутаторов', 'power'),
    ((SELECT id FROM tr_to1), 6,  'Проверка сетевого соединения и задержек', 'network'),
    ((SELECT id FROM tr_to1), 7,  'Тест PTZ-функций (поворот, наклон, зум)', 'test'),
    ((SELECT id FROM tr_to1), 8,  'Проверка инфракрасной подсветки (ночной режим)', 'inspection'),
    ((SELECT id FROM tr_to1), 9,  'Обновление прошивки устройств (если доступно)', 'firmware'),
    ((SELECT id FROM tr_to1), 10, 'Проверка целостности хранения данных NVR/DVR', 'storage');

-- TR: TO-2 (12 мес, 20 пунктов, 120 мин)
INSERT INTO maintenance_regulations (
    region_code, regulation_code, name, regulation_type,
    interval_months, estimated_minutes, total_items,
    compliance_standards, license_requirements, docs_required
) VALUES (
    'TR', 'TR-KVKK-TO2', 'KVKK + TS EN 62676 — ТО-2 (Годовое)',
    'TO-2', 12, 120, 20,
    ARRAY['KVKK №6698', 'TS EN 62676'],
    'KVKK VERBIS registration',
    '["KVKK compliance report (annual)", "TS EN 62676 full checklist", "CCTV privacy signage audit"]'::jsonb
);

WITH tr_to2 AS (SELECT id FROM maintenance_regulations WHERE regulation_code = 'TR-KVKK-TO2')
INSERT INTO maintenance_checklists (regulation_id, item_order, description, category) VALUES
    ((SELECT id FROM tr_to2), 1,  'Все пункты ТО-1 (10 позиций)', 'inspection'),
    ((SELECT id FROM tr_to2), 2,  'Проверка целостности кабельных трасс', 'cabling'),
    ((SELECT id FROM tr_to2), 3,  'Проверка кронштейнов и креплений камер', 'mounting'),
    ((SELECT id FROM tr_to2), 4,  'Тест резервного копирования конфигурации', 'backup'),
    ((SELECT id FROM tr_to2), 5,  'Проверка системы бесперебойного питания (UPS)', 'power'),
    ((SELECT id FROM tr_to2), 6,  'Анализ дискового пространства NVR', 'storage'),
    ((SELECT id FROM tr_to2), 7,  'Проверка логов ошибок и предупреждений', 'security'),
    ((SELECT id FROM tr_to2), 8,  'Тест восстановления после сбоя (DR test)', 'test'),
    ((SELECT id FROM tr_to2), 9,  'Проверка целостности базы данных', 'security'),
    ((SELECT id FROM tr_to2), 10, 'KVKK audit trail — проверка логов доступа к ПД', 'compliance');

-- ═══════════════════════════════════════════════════════════════════════════
-- P2-REG.8.4: Pre-loaded данные — VN (Vietnam: TCVN 11930 + Camera Standard 2025)
-- ═══════════════════════════════════════════════════════════════════════════

-- VN: TO-1 (3 мес, 12 пунктов, 45 мин)
INSERT INTO maintenance_regulations (
    region_code, regulation_code, name, regulation_type,
    interval_months, estimated_minutes, total_items,
    compliance_standards, license_requirements, docs_required
) VALUES (
    'VN', 'VN-TCVN-TO1', 'TCVN 11930 + Camera Standard — ТО-1 (Ежеквартальное)',
    'TO-1', 3, 45, 12,
    ARRAY['TCVN 11930:2017', 'Camera Standard (15.02.2025)'],
    'VN camera certification',
    '["Camera Standard compliance checklist", "Data residency verification report"]'::jsonb
);

WITH vn_to1 AS (SELECT id FROM maintenance_regulations WHERE regulation_code = 'VN-TCVN-TO1')
INSERT INTO maintenance_checklists (regulation_id, item_order, description, category) VALUES
    ((SELECT id FROM vn_to1), 1,  'Проверка работоспособности камер видеонаблюдения', 'inspection'),
    ((SELECT id FROM vn_to1), 2,  'Очистка оптики и корпусов камер', 'cleaning'),
    ((SELECT id FROM vn_to1), 3,  'Проверка качества видеозаписи и сжатия', 'storage'),
    ((SELECT id FROM vn_to1), 4,  'Тест сетевой связности и пропускной способности', 'network'),
    ((SELECT id FROM vn_to1), 5,  'Проверка инфракрасной подсветки', 'inspection'),
    ((SELECT id FROM vn_to1), 6,  'Проверка электропитания и PoE', 'power'),
    ((SELECT id FROM vn_to1), 7,  'Проверка угла обзора и зон покрытия', 'calibration'),
    ((SELECT id FROM vn_to1), 8,  'Тест PTZ-функций (если применимо)', 'test'),
    ((SELECT id FROM vn_to1), 9,  'Проверка герметичности корпусов (IP-защита)', 'mounting'),
    ((SELECT id FROM vn_to1), 10, 'Проверка датчиков движения и детекции', 'test'),
    ((SELECT id FROM vn_to1), 11, 'Проверка синхронизации времени (NTP)', 'network'),
    ((SELECT id FROM vn_to1), 12, 'Data residency — проверка локализации данных во Вьетнаме', 'compliance');

-- VN: TO-2 (12 мес, 22 пункта, 120 мин)
INSERT INTO maintenance_regulations (
    region_code, regulation_code, name, regulation_type,
    interval_months, estimated_minutes, total_items,
    compliance_standards, license_requirements, docs_required
) VALUES (
    'VN', 'VN-TCVN-TO2', 'TCVN 11930 + Camera Standard — ТО-2 (Годовое)',
    'TO-2', 12, 120, 22,
    ARRAY['TCVN 11930:2017', 'Camera Standard (15.02.2025)'],
    'VN camera certification',
    '["Camera Standard full compliance audit", "Data residency annual report", "TCVN 11930 assessment"]'::jsonb
);

WITH vn_to2 AS (SELECT id FROM maintenance_regulations WHERE regulation_code = 'VN-TCVN-TO2')
INSERT INTO maintenance_checklists (regulation_id, item_order, description, category) VALUES
    ((SELECT id FROM vn_to2), 1,  'Все пункты ТО-1 (12 позиций)', 'inspection'),
    ((SELECT id FROM vn_to2), 2,  'Проверка целостности кабельных трасс и разъёмов', 'cabling'),
    ((SELECT id FROM vn_to2), 3,  'Проверка креплений и несущих конструкций', 'mounting'),
    ((SELECT id FROM vn_to2), 4,  'Тест резервирования NVR/DVR (failover)', 'test'),
    ((SELECT id FROM vn_to2), 5,  'Проверка системы бесперебойного питания', 'power'),
    ((SELECT id FROM vn_to2), 6,  'Анализ производительности системы видеонаблюдения', 'performance'),
    ((SELECT id FROM vn_to2), 7,  'Проверка логов безопасности системы', 'security'),
    ((SELECT id FROM vn_to2), 8,  'Camera Standard compliance — аудит соответствия', 'compliance'),
    ((SELECT id FROM vn_to2), 9,  'Обновление ПО и прошивок всех устройств', 'firmware'),
    ((SELECT id FROM vn_to2), 10, 'Проверка retention политик хранения записей', 'compliance');

-- ═══════════════════════════════════════════════════════════════════════════
-- P2-REG.8.5: Pre-loaded данные — ID (Indonesia: SNI 27001 + UU PDP)
-- ═══════════════════════════════════════════════════════════════════════════

-- ID: TO-1 (3 мес, 10 пунктов, 60 мин)
INSERT INTO maintenance_regulations (
    region_code, regulation_code, name, regulation_type,
    interval_months, estimated_minutes, total_items,
    compliance_standards, license_requirements, docs_required
) VALUES (
    'ID', 'ID-SNI-TO1', 'SNI 27001 + UU PDP — ТО-1 (Ежеквартальное)',
    'TO-1', 3, 60, 10,
    ARRAY['SNI 27001', 'UU PDP 2022'],
    'Kominfo registration',
    '["SNI 27001 controls checklist", "PDP compliance verification"]'::jsonb
);

WITH id_to1 AS (SELECT id FROM maintenance_regulations WHERE regulation_code = 'ID-SNI-TO1')
INSERT INTO maintenance_checklists (regulation_id, item_order, description, category) VALUES
    ((SELECT id FROM id_to1), 1,  'Проверка работоспособности всех камер', 'inspection'),
    ((SELECT id FROM id_to1), 2,  'Очистка оптики и корпусов', 'cleaning'),
    ((SELECT id FROM id_to1), 3,  'Проверка качества записи и архива', 'storage'),
    ((SELECT id FROM id_to1), 4,  'Проверка сетевой связности', 'network'),
    ((SELECT id FROM id_to1), 5,  'Проверка электропитания и PoE', 'power'),
    ((SELECT id FROM id_to1), 6,  'Проверка ИК-подсветки (ночной режим)', 'inspection'),
    ((SELECT id FROM id_to1), 7,  'Тест PTZ-функций (если применимо)', 'test'),
    ((SELECT id FROM id_to1), 8,  'Проверка креплений и корпусов', 'mounting'),
    ((SELECT id FROM id_to1), 9,  'SNI 27001 — проверка средств контроля доступа', 'compliance'),
    ((SELECT id FROM id_to1), 10, 'UU PDP — проверка соответствия защите ПД', 'compliance');

-- ID: TO-2 (12 мес, 18 пунктов, 120 мин)
INSERT INTO maintenance_regulations (
    region_code, regulation_code, name, regulation_type,
    interval_months, estimated_minutes, total_items,
    compliance_standards, license_requirements, docs_required
) VALUES (
    'ID', 'ID-SNI-TO2', 'SNI 27001 + UU PDP — ТО-2 (Годовое)',
    'TO-2', 12, 120, 18,
    ARRAY['SNI 27001', 'UU PDP 2022'],
    'Kominfo registration',
    '["SNI 27001 internal audit report", "PDP compliance annual report", "Kominfo registration renewal"]'::jsonb
);

WITH id_to2 AS (SELECT id FROM maintenance_regulations WHERE regulation_code = 'ID-SNI-TO2')
INSERT INTO maintenance_checklists (regulation_id, item_order, description, category) VALUES
    ((SELECT id FROM id_to2), 1,  'Все пункты ТО-1 (10 позиций)', 'inspection'),
    ((SELECT id FROM id_to2), 2,  'Проверка целостности кабельных трасс', 'cabling'),
    ((SELECT id FROM id_to2), 3,  'Тест резервирования NVR (failover)', 'test'),
    ((SELECT id FROM id_to2), 4,  'Проверка системы бесперебойного питания', 'power'),
    ((SELECT id FROM id_to2), 5,  'Анализ безопасности системы видеонаблюдения', 'security'),
    ((SELECT id FROM id_to2), 6,  'Проверка логов доступа к системе', 'security'),
    ((SELECT id FROM id_to2), 7,  'UU PDP — обновление инвентаря ПД (data inventory)', 'compliance'),
    ((SELECT id FROM id_to2), 8,  'SNI 27001 — внутренний аудит ИБ', 'compliance'),
    ((SELECT id FROM id_to2), 9,  'Kominfo — проверка соответствия регистрации', 'certification');

-- ═══════════════════════════════════════════════════════════════════════════
-- P2-REG.8.6: Pre-loaded данные — BR (Brazil: ABNT NBR + LGPD)
-- ═══════════════════════════════════════════════════════════════════════════

-- BR: TO-1 (3 мес, 12 пунктов, 60 мин)
INSERT INTO maintenance_regulations (
    region_code, regulation_code, name, regulation_type,
    interval_months, estimated_minutes, total_items,
    compliance_standards, license_requirements, docs_required
) VALUES (
    'BR', 'BR-ABNT-TO1', 'ABNT NBR + LGPD — ТО-1 (Ежеквартальное)',
    'TO-1', 3, 60, 12,
    ARRAY['ABNT NBR series', 'LGPD'],
    'ANPD registration',
    '["ABNT NBR basic checklist", "LGPD privacy notices audit"]'::jsonb
);

WITH br_to1 AS (SELECT id FROM maintenance_regulations WHERE regulation_code = 'BR-ABNT-TO1')
INSERT INTO maintenance_checklists (regulation_id, item_order, description, category) VALUES
    ((SELECT id FROM br_to1), 1,  'Verificação de funcionamento de todas as câmeras', 'inspection'),
    ((SELECT id FROM br_to1), 2,  'Limpeza de lentes e carcaças', 'cleaning'),
    ((SELECT id FROM br_to1), 3,  'Verificação da qualidade de gravação', 'storage'),
    ((SELECT id FROM br_to1), 4,  'Verificação de conectividade de rede', 'network'),
    ((SELECT id FROM br_to1), 5,  'Verificação de alimentação e PoE', 'power'),
    ((SELECT id FROM br_to1), 6,  'Verificação de iluminação infravermelha', 'inspection'),
    ((SELECT id FROM br_to1), 7,  'Teste de funções PTZ (se aplicável)', 'test'),
    ((SELECT id FROM br_to1), 8,  'Verificação de fixações e suportes', 'mounting'),
    ((SELECT id FROM br_to1), 9,  'Verificação ABNT NBR — conformidade básica', 'compliance'),
    ((SELECT id FROM br_to1), 10, 'LGPD — verificação de avisos de privacidade', 'compliance'),
    ((SELECT id FROM br_to1), 11, 'Verificação de controle de acesso a gravações', 'security'),
    ((SELECT id FROM br_to1), 12, 'Teste de políticas de retenção de dados', 'storage');

-- BR: TO-2 (6 мес, 20 пунктов, 90 мин)
INSERT INTO maintenance_regulations (
    region_code, regulation_code, name, regulation_type,
    interval_months, estimated_minutes, total_items,
    compliance_standards, license_requirements, docs_required
) VALUES (
    'BR', 'BR-ABNT-TO2', 'ABNT NBR + LGPD — ТО-2 (Полугодовое)',
    'TO-2', 6, 90, 20,
    ARRAY['ABNT NBR series', 'LGPD'],
    'ANPD registration',
    '["ABNT NBR intermediate checklist", "LGPD DSAR readiness report", "Data mapping update"]'::jsonb
);

WITH br_to2 AS (SELECT id FROM maintenance_regulations WHERE regulation_code = 'BR-ABNT-TO2')
INSERT INTO maintenance_checklists (regulation_id, item_order, description, category) VALUES
    ((SELECT id FROM br_to2), 1,  'Все пункты ТО-1 (12 позиций)', 'inspection'),
    ((SELECT id FROM br_to2), 2,  'Verificação de cabos e conectores', 'cabling'),
    ((SELECT id FROM br_to2), 3,  'Teste de UPS e fontes de alimentação', 'power'),
    ((SELECT id FROM br_to2), 4,  'Análise de desempenho do sistema', 'performance'),
    ((SELECT id FROM br_to2), 5,  'Verificação de logs de segurança', 'security'),
    ((SELECT id FROM br_to2), 6,  'LGPD — verificação de prontidão DSAR', 'compliance'),
    ((SELECT id FROM br_to2), 7,  'LGPD — verificação de registros de consentimento', 'documentation'),
    ((SELECT id FROM br_to2), 8,  'Atualização de data mapping (LGPD)', 'compliance'),
    ((SELECT id FROM br_to2), 9,  'Teste de restore de backup', 'backup');

-- BR: TO-3 (12 мес, 30 пунктов, 180 мин)
INSERT INTO maintenance_regulations (
    region_code, regulation_code, name, regulation_type,
    interval_months, estimated_minutes, total_items,
    compliance_standards, license_requirements, docs_required
) VALUES (
    'BR', 'BR-ABNT-TO3', 'ABNT NBR + LGPD — ТО-3 (Годовой аудит)',
    'TO-3', 12, 180, 30,
    ARRAY['ABNT NBR series', 'LGPD'],
    'ANPD registration',
    '["ABNT NBR full audit report", "LGPD DPIA document", "Data processor audit report", "ANPD compliance report"]'::jsonb
);

WITH br_to3 AS (SELECT id FROM maintenance_regulations WHERE regulation_code = 'BR-ABNT-TO3')
INSERT INTO maintenance_checklists (regulation_id, item_order, description, category) VALUES
    ((SELECT id FROM br_to3), 1,  'Все пункты ТО-1 (12 позиций)', 'inspection'),
    ((SELECT id FROM br_to3), 2,  'Все пункты ТО-2 (8 доп. позиций)', 'inspection'),
    ((SELECT id FROM br_to3), 3,  'ABNT NBR — полный аудит соответствия', 'compliance'),
    ((SELECT id FROM br_to3), 4,  'LGPD — обзор DPIA (Data Protection Impact Assessment)', 'compliance'),
    ((SELECT id FROM br_to3), 5,  'Аудит операторов данных (LGPD)', 'certification'),
    ((SELECT id FROM br_to3), 6,  'Тест реагирования на инциденты ИБ', 'security'),
    ((SELECT id FROM br_to3), 7,  'Тест непрерывности бизнеса (BCP/DR)', 'test'),
    ((SELECT id FROM br_to3), 8,  'Полный анализ ёмкости NVR', 'performance'),
    ((SELECT id FROM br_to3), 9,  'Калибровка камер и проверка фокусировки', 'calibration'),
    ((SELECT id FROM br_to3), 10, 'Обзор политики безопасности ИБ', 'security');

-- ═══════════════════════════════════════════════════════════════════════════
-- P2-REG.8.7: Pre-loaded данные — ZA (South Africa: SANS + POPIA)
-- ═══════════════════════════════════════════════════════════════════════════

-- ZA: TO-1 (3 мес, 10 пунктов, 60 мин)
INSERT INTO maintenance_regulations (
    region_code, regulation_code, name, regulation_type,
    interval_months, estimated_minutes, total_items,
    compliance_standards, license_requirements, docs_required
) VALUES (
    'ZA', 'ZA-SANS-TO1', 'SANS + POPIA — ТО-1 (Ежеквартальное)',
    'TO-1', 3, 60, 10,
    ARRAY['SANS 10160-4', 'POPIA'],
    'Information Regulator registration',
    '["SANS 10160-4 basic checklist", "POPIA compliance verification"]'::jsonb
);

WITH za_to1 AS (SELECT id FROM maintenance_regulations WHERE regulation_code = 'ZA-SANS-TO1')
INSERT INTO maintenance_checklists (regulation_id, item_order, description, category) VALUES
    ((SELECT id FROM za_to1), 1,  'Inspection of all camera functionality', 'inspection'),
    ((SELECT id FROM za_to1), 2,  'Cleaning of lenses and housings', 'cleaning'),
    ((SELECT id FROM za_to1), 3,  'Verification of recording quality', 'storage'),
    ((SELECT id FROM za_to1), 4,  'Network connectivity check', 'network'),
    ((SELECT id FROM za_to1), 5,  'Power and PoE supply verification', 'power'),
    ((SELECT id FROM za_to1), 6,  'Infrared illumination check (night mode)', 'inspection'),
    ((SELECT id FROM za_to1), 7,  'PTZ functionality test (if applicable)', 'test'),
    ((SELECT id FROM za_to1), 8,  'Mounting and bracket integrity check', 'mounting'),
    ((SELECT id FROM za_to1), 9,  'SANS 10160-4 compliance verification', 'compliance'),
    ((SELECT id FROM za_to1), 10, 'POPIA privacy safeguards check', 'compliance');

-- ZA: TO-2 (6 мес, 16 пунктов, 90 мин)
INSERT INTO maintenance_regulations (
    region_code, regulation_code, name, regulation_type,
    interval_months, estimated_minutes, total_items,
    compliance_standards, license_requirements, docs_required
) VALUES (
    'ZA', 'ZA-SANS-TO2', 'SANS + POPIA — ТО-2 (Полугодовое)',
    'TO-2', 6, 90, 16,
    ARRAY['SANS 10160-4', 'POPIA'],
    'Information Regulator registration',
    '["SANS 10160-4 audit checklist", "POPIA compliance report", "Data subject access request log"]'::jsonb
);

WITH za_to2 AS (SELECT id FROM maintenance_regulations WHERE regulation_code = 'ZA-SANS-TO2')
INSERT INTO maintenance_checklists (regulation_id, item_order, description, category) VALUES
    ((SELECT id FROM za_to2), 1,  'All TO-1 items (10 items)', 'inspection'),
    ((SELECT id FROM za_to2), 2,  'Cable routing and connector integrity', 'cabling'),
    ((SELECT id FROM za_to2), 3,  'UPS and backup power system test', 'power'),
    ((SELECT id FROM za_to2), 4,  'System performance analysis', 'performance'),
    ((SELECT id FROM za_to2), 5,  'Security logs and access audit', 'security'),
    ((SELECT id FROM za_to2), 6,  'POPIA — data subject access request (DSAR) test', 'compliance'),
    ((SELECT id FROM za_to2), 7,  'Retention policy compliance verification', 'compliance');

-- ═══════════════════════════════════════════════════════════════════════════
-- P2-REG.8.8: Функция для получения шаблонов по региону
-- ═══════════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION get_regional_templates(p_region_code VARCHAR(2))
RETURNS TABLE (
    regulation_id       TEXT,
    regulation_code     VARCHAR(20),
    regulation_name     TEXT,
    regulation_type     VARCHAR(4),
    interval_months     INT,
    estimated_minutes   INT,
    total_items         INT,
    compliance_standards TEXT[],
    license_requirements TEXT,
    docs_required       JSONB
)
LANGUAGE plpgsql
STABLE
AS $$
BEGIN
    RETURN QUERY
    SELECT
        mr.id,
        mr.regulation_code,
        mr.name,
        mr.regulation_type,
        mr.interval_months,
        mr.estimated_minutes,
        mr.total_items,
        mr.compliance_standards,
        mr.license_requirements,
        mr.docs_required
    FROM maintenance_regulations mr
    WHERE mr.region_code = p_region_code
      AND mr.is_active = true
    ORDER BY mr.regulation_type;
END;
$$;

COMMENT ON FUNCTION get_regional_templates(VARCHAR) IS
    'P2-REG.8: Возвращает активные шаблоны регламентов для указанного региона.';

-- ═══════════════════════════════════════════════════════════════════════════
-- P2-REG.8.9: Функция для получения чек-листа по регламенту
-- ═══════════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION get_regulation_checklist(p_regulation_id TEXT)
RETURNS TABLE (
    item_order      INT,
    description     TEXT,
    category        VARCHAR(50),
    is_required     BOOLEAN
)
LANGUAGE plpgsql
STABLE
AS $$
BEGIN
    RETURN QUERY
    SELECT
        mc.item_order,
        mc.description,
        mc.category,
        mc.is_required
    FROM maintenance_checklists mc
    WHERE mc.regulation_id = p_regulation_id
    ORDER BY mc.item_order;
END;
$$;

COMMENT ON FUNCTION get_regulation_checklist(TEXT) IS
    'P2-REG.8: Возвращает пункты чек-листа для указанного регламента.';

-- ═══════════════════════════════════════════════════════════════════════════
-- P2-REG.8.10: Триггер авто-обновления updated_at
-- ═══════════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION update_maint_reg_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_maint_reg_updated ON maintenance_regulations;
CREATE TRIGGER trg_maint_reg_updated
    BEFORE UPDATE ON maintenance_regulations
    FOR EACH ROW
    EXECUTE FUNCTION update_maint_reg_timestamp();
