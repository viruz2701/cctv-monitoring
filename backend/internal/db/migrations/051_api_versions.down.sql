-- +migrate Down

DROP TABLE IF EXISTS api_changelog CASCADE;
DROP TABLE IF EXISTS api_versions CASCADE;
