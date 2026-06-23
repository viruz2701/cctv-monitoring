-- +migrate Down

DROP INDEX IF EXISTS idx_csrf_tokens_expires;
DROP INDEX IF EXISTS idx_csrf_tokens_token;
DROP INDEX IF EXISTS idx_csrf_tokens_user;
DROP TABLE IF EXISTS csrf_tokens;

DROP INDEX IF EXISTS idx_vulnerability_scans_status;
DROP INDEX IF EXISTS idx_vulnerability_scans_type;
DROP TABLE IF EXISTS vulnerability_scans;

DROP INDEX IF EXISTS idx_audit_log_entity;
DROP INDEX IF EXISTS idx_audit_log_trace_id;
ALTER TABLE audit_log DROP COLUMN IF EXISTS source_service;
ALTER TABLE audit_log DROP COLUMN IF EXISTS trace_id;
ALTER TABLE audit_log DROP COLUMN IF EXISTS prev_hash;

DROP INDEX IF EXISTS idx_user_approval_queue_user;
DROP INDEX IF EXISTS idx_user_approval_queue_status;
DROP TABLE IF EXISTS user_approval_queue;

ALTER TABLE users DROP COLUMN IF EXISTS allowed_ips;
ALTER TABLE users DROP COLUMN IF EXISTS max_concurrent_sessions;
ALTER TABLE users DROP COLUMN IF EXISTS session_timeout_minutes;
ALTER TABLE users DROP COLUMN IF EXISTS locked_until;
ALTER TABLE users DROP COLUMN IF EXISTS failed_login_attempts;
ALTER TABLE users DROP COLUMN IF EXISTS last_activity_at;

DROP INDEX IF EXISTS idx_devices_asset_status;
DROP INDEX IF EXISTS idx_devices_asset_criticality;
DROP INDEX IF EXISTS idx_devices_asset_tag;

ALTER TABLE devices DROP COLUMN IF EXISTS vendor_contact;
ALTER TABLE devices DROP COLUMN IF EXISTS purchase_cost;
ALTER TABLE devices DROP COLUMN IF EXISTS purchase_date;
ALTER TABLE devices DROP COLUMN IF EXISTS warranty_expires_at;
ALTER TABLE devices DROP COLUMN IF EXISTS asset_status;
ALTER TABLE devices DROP COLUMN IF EXISTS asset_tag;
ALTER TABLE devices DROP COLUMN IF EXISTS asset_location_physical;
ALTER TABLE devices DROP COLUMN IF EXISTS asset_owner;
ALTER TABLE devices DROP COLUMN IF EXISTS asset_criticality;
ALTER TABLE devices DROP COLUMN IF EXISTS asset_category;
