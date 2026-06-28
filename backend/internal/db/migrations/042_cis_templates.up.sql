-- +migrate Up
-- P2-REG.8b: CIS Regional Maintenance Templates — BY, RU, KZ, UZ, KG
--
-- Pre-loaded шаблоны для стран СНГ на основе регулирующих документов.
--
-- Compliance:
--   - СН 3.02.19-2025 (BY) — CCTV стандарт РБ (вводится 24.09.2025)
--   - ТКП 472-2013 (BY) — ОПС системы РБ
--   - РД 25.964-90 (RU) — Техобслуживание АУПТ, ОПС
--   - РД 009-01-96 (RU) — Пожарная автоматика
--   - РД 009-02-96 (RU) — ТО и ППР
--   - РД 78.145-93 (RU) — ОПС монтаж
--   - ГОСТ Р 51558-2014 (RU) — CCTV стандарт РФ
--   - Приказ МЧС №55 (KZ) — Пожарная автоматика РК
--   - СТ РК ГОСТ Р 50776-2010 (KZ) — Тревожная сигнализация
--   - Закон РК «Об охранной деятельности» (KZ)
--   - ISO 27001 A.12.4 (Audit trail)

-- ═══════════════════════════════════════════════════════════════════════════
-- BY — Республика Беларусь
-- ═══════════════════════════════════════════════════════════════════════════

-- СН 3.02.19-2025 (CCTV)
INSERT INTO maintenance_regulations
    (region_code, regulation_code, name, regulation_type, interval_months, estimated_minutes, total_items,
     compliance_standards, license_requirements, docs_required)
VALUES
    ('BY', 'BY-SN-TO1', 'СН 3.02.19-2025 — ТО-1 CCTV (ежемесячное)', 'TO-1', 1, 30, 12,
     ARRAY['СН 3.02.19-2025', 'СТБ 34.101.27'],
     'ОАЦ лицензия (КИИ)',
     '["Журнал ТО CCTV (Приложение Б)", "Акт осмотра"]'),
    ('BY', 'BY-SN-TO2', 'СН 3.02.19-2025 — ТО-2 CCTV (квартальное)', 'TO-2', 3, 90, 25,
     ARRAY['СН 3.02.19-2025', 'СТБ IEC 62443', 'СТБ 34.101.27'],
     'ОАЦ лицензия (КИИ)',
     '["Журнал ТО CCTV", "Акт периодического ТО", "Проверка целостности архива"]'),
    ('BY', 'BY-SN-TO3', 'СН 3.02.19-2025 — ТО-3 CCTV (годовое)', 'TO-3', 12, 180, 40,
     ARRAY['СН 3.02.19-2025', 'СТБ IEC 62443', 'СТБ 34.101.27', 'Приказ ОАЦ №66'],
     'ОАЦ лицензия (КИИ)',
     '["Акт первичного обследования", "Акт периодического ТО", "Compliance report ОАЦ", "Audit log verification"]');

-- ТКП 472-2013 (ОПС)
INSERT INTO maintenance_regulations
    (region_code, regulation_code, name, regulation_type, interval_months, estimated_minutes, total_items,
     compliance_standards, license_requirements, docs_required)
VALUES
    ('BY', 'BY-TKP-TO1', 'ТКП 472-2013 — ТО-1 ОПС (ежемесячное)', 'TO-1', 1, 30, 10,
     ARRAY['ТКП 472-2013', 'СТБ 34.101.27'],
     'ОАЦ лицензия',
     '["Журнал ТО ОПС", "Акт проверки"]'),
    ('BY', 'BY-TKP-TO2', 'ТКП 472-2013 — ТО-2 ОПС (полугодовое)', 'TO-2', 6, 90, 20,
     ARRAY['ТКП 472-2013', 'СТБ 34.101.27'],
     'ОАЦ лицензия',
     '["Акт ТО-2", "Проверка резервирования"]'),
    ('BY', 'BY-TKP-TO3', 'ТКП 472-2013 — ТО-3 ОПС (годовое)', 'TO-3', 12, 180, 35,
     ARRAY['ТКП 472-2013', 'СТБ IEC 62443'],
     'ОАЦ лицензия',
     '["График ТО на год", "Акт ТО-3", "Дефектовка"]');

-- ═══════════════════════════════════════════════════════════════════════════
-- RU — Российская Федерация
-- ═══════════════════════════════════════════════════════════════════════════

-- РД 25.964-90 (АУПТ, ОПС)
INSERT INTO maintenance_regulations
    (region_code, regulation_code, name, regulation_type, interval_months, estimated_minutes, total_items,
     compliance_standards, license_requirements, docs_required)
VALUES
    ('RU', 'RU-RD25-TO1', 'РД 25.964-90 — ТО-1 (ежемесячное)', 'TO-1', 1, 45, 18,
     ARRAY['РД 25.964-90', '149-ФЗ', 'Приказ ФСТЭК №17'],
     'МЧС лицензия (обязательно)',
     '["Журнал регистрации работ (Приложение 6)", "Акт осмотра"]'),
    ('RU', 'RU-RD25-TO2', 'РД 25.964-90 — ТО-2 (полугодовое)', 'TO-2', 6, 120, 30,
     ARRAY['РД 25.964-90', '149-ФЗ', 'ГОСТ Р 34.10-2012'],
     'МЧС лицензия',
     '["График ТО (Приложение В)", "Акт дефектовки", "Протокол испытаний"]'),
    ('RU', 'RU-RD25-TO3', 'РД 25.964-90 — ТО-3 (годовое)', 'TO-3', 12, 240, 50,
     ARRAY['РД 25.964-90', '149-ФЗ', '152-ФЗ', 'ГОСТ Р 51558-2014'],
     'МЧС лицензия (КИИ)',
     '["Акт первичного обследования (Приложение 1)", "Акт периодического ТО", "Акт испытаний", "Deffect report"]');

-- ГОСТ Р 51558-2014 (CCTV)
INSERT INTO maintenance_regulations
    (region_code, regulation_code, name, regulation_type, interval_months, estimated_minutes, total_items,
     compliance_standards, license_requirements, docs_required)
VALUES
    ('RU', 'RU-GOST-TO1', 'ГОСТ Р 51558-2014 — ТО-1 CCTV (ежемесячное)', 'TO-1', 1, 30, 14,
     ARRAY['ГОСТ Р 51558-2014', '149-ФЗ', '152-ФЗ'],
     'МЧС лицензия',
     '["Журнал ТО CCTV", "Акт осмотра оборудования"]'),
    ('RU', 'RU-GOST-TO2', 'ГОСТ Р 51558-2014 — ТО-2 CCTV (квартальное)', 'TO-2', 3, 90, 25,
     ARRAY['ГОСТ Р 51558-2014', '149-ФЗ'],
     'МЧС лицензия',
     '["Акт ТО-2", "Проверка видеоархива", "Тест распознавания"]'),
    ('RU', 'RU-GOST-TO3', 'ГОСТ Р 51558-2014 — ТО-3 CCTV (годовое)', 'TO-3', 12, 180, 40,
     ARRAY['ГОСТ Р 51558-2014', '149-ФЗ', '152-ФЗ', 'ФСТЭК'],
     'МЧС лицензия (КИИ)',
     '["Акт годового ТО", "Аудит безопасности (149-ФЗ)", "Compliance report ФСТЭК"]');

-- РД 009-01-96 (Пожарная автоматика)
INSERT INTO maintenance_regulations
    (region_code, regulation_code, name, regulation_type, interval_months, estimated_minutes, total_items,
     compliance_standards, license_requirements, docs_required)
VALUES
    ('RU', 'RU-RD009-TO1', 'РД 009-01-96 — Пожарная автоматика (ежемесячное)', 'TO-1', 1, 40, 16,
     ARRAY['РД 009-01-96', '149-ФЗ'],
     'МЧС лицензия',
     '["Журнал учёта ТО пожарной автоматики", "Акт проверки"]'),
    ('RU', 'RU-RD009-TO2', 'РД 009-01-96 — Пожарная автоматика (полугодовое)', 'TO-2', 6, 120, 28,
     ARRAY['РД 009-01-96'],
     'МЧС лицензия',
     '["Акт ТО-2", "Протокол испытаний"]'),
    ('RU', 'RU-RD009-TO3', 'РД 009-01-96 — Пожарная автоматика (годовое)', 'TO-3', 12, 240, 45,
     ARRAY['РД 009-01-96', '149-ФЗ'],
     'МЧС лицензия',
     '["Акт годового ТО", "Дефектовка", "Предписание"]');

-- ═══════════════════════════════════════════════════════════════════════════
-- KZ — Республика Казахстан
-- ═══════════════════════════════════════════════════════════════════════════

-- Приказ МЧС №55 (Пожарная автоматика)
INSERT INTO maintenance_regulations
    (region_code, regulation_code, name, regulation_type, interval_months, estimated_minutes, total_items,
     compliance_standards, license_requirements, docs_required)
VALUES
    ('KZ', 'KZ-MCHS-TO1', 'Приказ МЧС №55 — ТО-1 (ежемесячное)', 'TO-1', 1, 30, 16,
     ARRAY['Приказ МЧС №55', 'Закон РК «Об охранной деятельности»'],
     'МЧС РК (обязательно с 01.02.2026, уголовная ответственность)',
     '["Журнал учёта ТО систем пожарной автоматики", "Акт осмотра"]'),
    ('KZ', 'KZ-MCHS-TO2', 'Приказ МЧС №55 — ТО-2 (квартальное)', 'TO-2', 3, 90, 28,
     ARRAY['Приказ МЧС №55', 'СТ РК ГОСТ Р 50776-2010'],
     'МЧС РК лицензия',
     '["Акт технического освидетельствования", "Проверка резервирования"]'),
    ('KZ', 'KZ-MCHS-TO3', 'Приказ МЧС №55 — ТО-3 (годовое)', 'TO-3', 12, 180, 40,
     ARRAY['Приказ МЧС №55', 'СТ РК ГОСТ Р 50776-2010', 'Закон РК «Об охранной деятельности»'],
     'МЧС РК лицензия',
     '["Акт годового ТО", "Compliance report", "Лицензионный контроль"]');

-- ═══════════════════════════════════════════════════════════════════════════
-- UZ — Республика Узбекистан
-- ═══════════════════════════════════════════════════════════════════════════

-- Закон «О персональных данных» (процедурный)
INSERT INTO maintenance_regulations
    (region_code, regulation_code, name, regulation_type, interval_months, estimated_minutes, total_items,
     compliance_standards, license_requirements, docs_required)
VALUES
    ('UZ', 'UZ-PD-TO1', 'Закон о ПД — ТО-1 CCTV (квартальное)', 'TO-1', 3, 45, 10,
     ARRAY['Закон РУз «О персональных данных»'],
     'Лицензия ID.UZ',
     '["Журнал ТО CCTV", "Акт осмотра"]'),
    ('UZ', 'UZ-PD-TO2', 'Закон о ПД — ТО-2 CCTV (годовое)', 'TO-2', 12, 120, 20,
     ARRAY['Закон РУз «О персональных данных»'],
     'Лицензия ID.UZ',
     '["Акт годового ТО", "Compliance report ПД", "Аудит безопасности"]');

-- ═══════════════════════════════════════════════════════════════════════════
-- KG — Кыргызская Республика
-- ═══════════════════════════════════════════════════════════════════════════

-- Закон КР «О персональных данных»
INSERT INTO maintenance_regulations
    (region_code, regulation_code, name, regulation_type, interval_months, estimated_minutes, total_items,
     compliance_standards, license_requirements, docs_required)
VALUES
    ('KG', 'KG-PD-TO1', 'Закон КР о ПД — ТО-1 CCTV (квартальное)', 'TO-1', 3, 45, 10,
     ARRAY['Закон КР «О персональных данных»'],
     'Лицензия МЧС КР',
     '["Журнал ТО", "Акт осмотра"]'),
    ('KG', 'KG-PD-TO2', 'Закон КР о ПД — ТО-2 CCTV (годовое)', 'TO-2', 12, 120, 20,
     ARRAY['Закон КР «О персональных данных»'],
     'Лицензия МЧС КР',
     '["Акт годового ТО", "Compliance report"]');

-- ═══════════════════════════════════════════════════════════════════════════
-- Вспомогательные функции (регион-специфичные)
-- ═══════════════════════════════════════════════════════════════════════════

-- get_cis_regulations — получение всех регламентов для стран СНГ
CREATE OR REPLACE FUNCTION get_cis_regulations()
RETURNS TABLE (
    region_code VARCHAR(2),
    regulation_code VARCHAR(20),
    name TEXT,
    regulation_type VARCHAR(4),
    interval_months INT,
    estimated_minutes INT,
    total_items INT
) AS $$
BEGIN
    RETURN QUERY
    SELECT mr.region_code, mr.regulation_code, mr.name,
           mr.regulation_type, mr.interval_months,
           mr.estimated_minutes, mr.total_items
    FROM maintenance_regulations mr
    WHERE mr.region_code IN ('BY', 'RU', 'KZ', 'UZ', 'KG')
      AND mr.is_active = true
    ORDER BY mr.region_code, mr.regulation_type;
END;
$$ LANGUAGE plpgsql;

-- get_regulation_by_doc — поиск по нормативному документу
CREATE OR REPLACE FUNCTION get_regulation_by_doc(doc_name TEXT)
RETURNS TABLE (
    region_code VARCHAR(2),
    regulation_code VARCHAR(20),
    name TEXT,
    doc_type VARCHAR(4)
) AS $$
BEGIN
    RETURN QUERY
    SELECT mr.region_code, mr.regulation_code, mr.name,
           mr.regulation_type
    FROM maintenance_regulations mr
    WHERE doc_name = ANY(mr.compliance_standards)
      AND mr.is_active = true
    ORDER BY mr.region_code;
END;
$$ LANGUAGE plpgsql;
