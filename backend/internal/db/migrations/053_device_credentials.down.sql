-- +migrate Down
-- CRED-01: Device Credentials Storage (rollback)

DROP TRIGGER IF EXISTS trg_device_credentials_updated ON device_credentials;
DROP FUNCTION IF EXISTS trg_device_credentials_updated();

DROP TABLE IF EXISTS device_credentials;
