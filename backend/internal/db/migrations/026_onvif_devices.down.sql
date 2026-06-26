-- +migrate Down
-- Migration 026: ONVIF Devices (Rollback)

DROP TABLE IF EXISTS onvif_devices;
