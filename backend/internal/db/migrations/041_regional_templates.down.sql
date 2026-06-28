-- +migrate Down
-- P2-REG.8: Откат — удаление таблиц региональных шаблонов ТО

DROP TRIGGER IF EXISTS trg_maint_reg_updated ON maintenance_regulations;
DROP FUNCTION IF EXISTS update_maint_reg_timestamp();

DROP FUNCTION IF EXISTS get_regulation_checklist(TEXT);
DROP FUNCTION IF EXISTS get_regional_templates(VARCHAR);

DROP TABLE IF EXISTS maintenance_checklists CASCADE;
DROP TABLE IF EXISTS maintenance_regulations CASCADE;
