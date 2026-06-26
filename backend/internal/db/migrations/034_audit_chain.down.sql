-- +migrate Down
-- Откат P3-2: Audit Trail Compliance
DROP FUNCTION IF EXISTS verify_audit_chain();
DROP FUNCTION IF EXISTS archive_audit_logs(INTEGER);
DROP TABLE IF EXISTS audit_log_archive;
DROP FUNCTION IF EXISTS get_last_audit_hmac();

ALTER TABLE audit_log
    DROP COLUMN IF EXISTS trace_id,
    DROP COLUMN IF EXISTS prev_hash;
