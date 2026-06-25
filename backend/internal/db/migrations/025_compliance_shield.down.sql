-- +migrate Down
-- KF-15.1.1: Compliance & Fines Shield — откат

DROP FUNCTION IF EXISTS refresh_compliance_risks();
DROP TABLE IF EXISTS compliance_audit_log;
DROP TABLE IF EXISTS compliance_risks;
