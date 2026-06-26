-- +migrate Down
-- Migration 030: Dashboard Multi-Device Sync (Rollback)

DROP INDEX IF EXISTS idx_workspace_layouts_updated;
DROP INDEX IF EXISTS idx_workspace_layouts_user;
DROP TABLE IF EXISTS workspace_layouts;
