-- +migrate Down
DROP FUNCTION IF EXISTS log_compliance_audit(TEXT, TEXT, TEXT, VARCHAR(2), JSONB, TEXT);
DROP FUNCTION IF EXISTS get_due_regulations();
DROP TABLE IF EXISTS compliance_journal;
