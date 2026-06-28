-- +migrate Down
-- P2-REG.8b: Откат CIS Regional Templates

DROP FUNCTION IF EXISTS get_cis_regulations();
DROP FUNCTION IF EXISTS get_regulation_by_doc(TEXT);

DELETE FROM maintenance_regulations WHERE region_code IN ('BY', 'RU', 'KZ', 'UZ', 'KG');
