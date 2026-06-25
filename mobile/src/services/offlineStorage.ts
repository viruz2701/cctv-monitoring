import * as SQLite from 'expo-sqlite';
import { WorkOrder } from '../types';

// ──────────────────────────────────────────────────
// Типы
// ──────────────────────────────────────────────────

export interface DeviceRow {
  id: string;
  name: string;
  device_type: string;
  status: string;
  site_name: string | null;
  latitude: number;
  longitude: number;
  health: string;
  updated_at: string;
}

export interface SiteRow {
  id: string;
  name: string;
  address: string | null;
  latitude: number;
  longitude: number;
  updated_at: string;
}

export interface PendingMutation {
  id: string;
  entity_type: 'work_order' | 'device' | 'site';
  entity_id: string;
  mutation_type: 'create' | 'update' | 'delete';
  payload: string; // JSON
  timestamp: number;
  retry_count: number;
  last_error: string | null;
}

// ──────────────────────────────────────────────────
// Database singleton
// ──────────────────────────────────────────────────

const DB_NAME = 'cctv-offline.db';

let _db: SQLite.SQLiteDatabase | null = null;

async function getDb(): Promise<SQLite.SQLiteDatabase> {
  if (!_db) {
    _db = await SQLite.openDatabaseAsync(DB_NAME);
  }
  return _db;
}

// ──────────────────────────────────────────────────
// Инициализация
// ──────────────────────────────────────────────────

export async function initDatabase(): Promise<void> {
  const db = await getDb();

  await db.execAsync(`
    PRAGMA journal_mode = WAL;
    PRAGMA foreign_keys = ON;

    CREATE TABLE IF NOT EXISTS work_orders (
      id            TEXT PRIMARY KEY NOT NULL,
      schedule_id   TEXT,
      device_id     TEXT NOT NULL,
      device_name   TEXT,
      site_name     TEXT,
      type          TEXT NOT NULL CHECK(type IN ('preventive','corrective','emergency')),
      status        TEXT NOT NULL CHECK(status IN ('open','in_progress','completed','cancelled')),
      priority      TEXT NOT NULL CHECK(priority IN ('critical','high','medium','low')),
      assigned_to   TEXT,
      sla_deadline  TEXT,
      checklist     TEXT NOT NULL DEFAULT '[]',
      started_at    TEXT,
      completed_at  TEXT,
      notes         TEXT,
      photos        TEXT NOT NULL DEFAULT '[]',
      parts_used    TEXT NOT NULL DEFAULT '[]',
      created_by    TEXT,
      created_at    TEXT NOT NULL,
      updated_at    TEXT NOT NULL,
      device_name_display TEXT,
      assignee_name TEXT,
      sla_status    TEXT,
      synced_at     TEXT
    );

    CREATE INDEX IF NOT EXISTS idx_wo_status ON work_orders(status);
    CREATE INDEX IF NOT EXISTS idx_wo_updated ON work_orders(updated_at);
    CREATE INDEX IF NOT EXISTS idx_wo_device ON work_orders(device_id);

    CREATE TABLE IF NOT EXISTS devices (
      id          TEXT PRIMARY KEY NOT NULL,
      name        TEXT NOT NULL,
      device_type TEXT NOT NULL,
      status      TEXT NOT NULL DEFAULT 'OFFLINE',
      site_name   TEXT,
      latitude    REAL NOT NULL DEFAULT 0,
      longitude   REAL NOT NULL DEFAULT 0,
      health      TEXT NOT NULL DEFAULT 'healthy',
      updated_at  TEXT NOT NULL,
      synced_at   TEXT
    );

    CREATE INDEX IF NOT EXISTS idx_dev_status ON devices(status);
    CREATE INDEX IF NOT EXISTS idx_dev_updated ON devices(updated_at);

    CREATE TABLE IF NOT EXISTS sites (
      id         TEXT PRIMARY KEY NOT NULL,
      name       TEXT NOT NULL,
      address    TEXT,
      latitude   REAL NOT NULL DEFAULT 0,
      longitude  REAL NOT NULL DEFAULT 0,
      updated_at TEXT NOT NULL,
      synced_at  TEXT
    );

    CREATE TABLE IF NOT EXISTS pending_sync (
      id            TEXT PRIMARY KEY NOT NULL,
      entity_type   TEXT NOT NULL CHECK(entity_type IN ('work_order','device','site')),
      entity_id     TEXT NOT NULL,
      mutation_type TEXT NOT NULL CHECK(mutation_type IN ('create','update','delete')),
      payload       TEXT NOT NULL,
      timestamp     INTEGER NOT NULL,
      retry_count   INTEGER NOT NULL DEFAULT 0,
      last_error    TEXT
    );

    CREATE INDEX IF NOT EXISTS idx_pending_ts ON pending_sync(timestamp);
    CREATE INDEX IF NOT EXISTS idx_pending_entity ON pending_sync(entity_type, entity_id);
  `);
}

// ──────────────────────────────────────────────────
// Work Orders CRUD
// ──────────────────────────────────────────────────

export async function upsertWorkOrders(orders: WorkOrder[]): Promise<void> {
  const db = await getDb();

  const stmt = `
    INSERT OR REPLACE INTO work_orders (
      id, schedule_id, device_id, device_name, site_name,
      type, status, priority, assigned_to, sla_deadline,
      checklist, started_at, completed_at, notes, photos,
      parts_used, created_by, created_at, updated_at,
      device_name_display, assignee_name, sla_status, synced_at
    ) VALUES (?,?,?,?,?, ?,?,?,?,?, ?,?,?,?,?, ?,?,?,?, ?,?,?, datetime('now'))
  `;

  for (const wo of orders) {
    await db.runAsync(stmt, [
      wo.id,
      wo.schedule_id ?? null,
      wo.device_id,
      wo.device_name ?? null,
      wo.site_name ?? null,
      wo.type,
      wo.status,
      wo.priority,
      wo.assigned_to ?? null,
      wo.sla_deadline ?? null,
      JSON.stringify(wo.checklist),
      wo.started_at ?? null,
      wo.completed_at ?? null,
      wo.notes ?? null,
      JSON.stringify(wo.photos),
      JSON.stringify(wo.parts_used),
      wo.created_by ?? null,
      wo.created_at,
      wo.updated_at,
      wo.device_name_display ?? null,
      wo.assignee_name ?? null,
      wo.sla_status ?? null,
    ]);
  }
}

export async function upsertWorkOrder(wo: WorkOrder): Promise<void> {
  await upsertWorkOrders([wo]);
}

export async function getWorkOrders(
  status?: WorkOrder['status'],
): Promise<WorkOrder[]> {
  const db = await getDb();

  let sql = 'SELECT * FROM work_orders';
  const params: (string | number)[] = [];

  if (status) {
    sql += ' WHERE status = ?';
    params.push(status);
  }

  sql += ' ORDER BY updated_at DESC';

  const rows = await db.getAllAsync(sql, params);
  return (rows as any[]).map(mapWorkOrderRow);
}

export async function getWorkOrder(id: string): Promise<WorkOrder | null> {
  const db = await getDb();
  const row = await db.getFirstAsync(
    'SELECT * FROM work_orders WHERE id = ?',
    [id],
  );
  return row ? mapWorkOrderRow(row as any) : null;
}

export async function deleteWorkOrder(id: string): Promise<void> {
  const db = await getDb();
  await db.runAsync('DELETE FROM work_orders WHERE id = ?', [id]);
}

export async function clearWorkOrders(): Promise<void> {
  const db = await getDb();
  await db.runAsync('DELETE FROM work_orders');
}

// ──────────────────────────────────────────────────
// Devices CRUD
// ──────────────────────────────────────────────────

export async function upsertDevices(devices: DeviceRow[]): Promise<void> {
  const db = await getDb();

  const stmt = `
    INSERT OR REPLACE INTO devices (
      id, name, device_type, status, site_name,
      latitude, longitude, health, updated_at, synced_at
    ) VALUES (?,?,?,?,?, ?,?,?,?, datetime('now'))
  `;

  for (const d of devices) {
    await db.runAsync(stmt, [
      d.id, d.name, d.device_type, d.status, d.site_name,
      d.latitude, d.longitude, d.health, d.updated_at,
    ]);
  }
}

export async function getDevices(): Promise<DeviceRow[]> {
  const db = await getDb();
  return db.getAllAsync(
    'SELECT * FROM devices ORDER BY updated_at DESC',
  ) as Promise<DeviceRow[]>;
}

export async function clearDevices(): Promise<void> {
  const db = await getDb();
  await db.runAsync('DELETE FROM devices');
}

// ──────────────────────────────────────────────────
// Sites CRUD
// ──────────────────────────────────────────────────

export async function upsertSites(sites: SiteRow[]): Promise<void> {
  const db = await getDb();

  const stmt = `
    INSERT OR REPLACE INTO sites (
      id, name, address, latitude, longitude, updated_at, synced_at
    ) VALUES (?,?,?,?,?, ?, datetime('now'))
  `;

  for (const s of sites) {
    await db.runAsync(stmt, [
      s.id, s.name, s.address, s.latitude, s.longitude, s.updated_at,
    ]);
  }
}

export async function getSites(): Promise<SiteRow[]> {
  const db = await getDb();
  return db.getAllAsync(
    'SELECT * FROM sites ORDER BY updated_at DESC',
  ) as Promise<SiteRow[]>;
}

export async function clearSites(): Promise<void> {
  const db = await getDb();
  await db.runAsync('DELETE FROM sites');
}

// ──────────────────────────────────────────────────
// Pending Mutations
// ──────────────────────────────────────────────────

export async function savePendingMutation(
  mutation: Omit<PendingMutation, 'id' | 'timestamp' | 'retry_count' | 'last_error'>,
): Promise<string> {
  const db = await getDb();
  const id = `pm_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;

  await db.runAsync(
    `INSERT INTO pending_sync (id, entity_type, entity_id, mutation_type, payload, timestamp)
     VALUES (?, ?, ?, ?, ?, ?)`,
    [id, mutation.entity_type, mutation.entity_id, mutation.mutation_type, mutation.payload, Date.now()],
  );

  return id;
}

export async function getPendingMutations(): Promise<PendingMutation[]> {
  const db = await getDb();
  return db.getAllAsync(
    'SELECT * FROM pending_sync ORDER BY timestamp ASC',
  ) as Promise<PendingMutation[]>;
}

export async function removePendingMutation(id: string): Promise<void> {
  const db = await getDb();
  await db.runAsync('DELETE FROM pending_sync WHERE id = ?', [id]);
}

export async function incrementPendingRetry(
  id: string,
  error: string,
): Promise<void> {
  const db = await getDb();
  await db.runAsync(
    'UPDATE pending_sync SET retry_count = retry_count + 1, last_error = ? WHERE id = ?',
    [error, id],
  );
}

export async function clearPendingMutations(): Promise<void> {
  const db = await getDb();
  await db.runAsync('DELETE FROM pending_sync');
}

export async function getPendingMutationCount(): Promise<number> {
  const db = await getDb();
  const row = await db.getFirstAsync(
    'SELECT COUNT(*) as count FROM pending_sync',
  );
  return (row as any)?.count ?? 0;
}

// ──────────────────────────────────────────────────
// Database maintenance
// ──────────────────────────────────────────────────

export async function getDatabaseSize(): Promise<number> {
  const db = await getDb();
  const row = await db.getFirstAsync(
    "SELECT page_count * page_size AS size_bytes FROM pragma_page_count, pragma_page_size",
  );
  return (row as any)?.size_bytes ?? 0;
}

export async function clearAllData(): Promise<void> {
  const db = await getDb();
  await db.execAsync(`
    DELETE FROM work_orders;
    DELETE FROM devices;
    DELETE FROM sites;
    DELETE FROM pending_sync;
  `);
}

export async function closeDatabase(): Promise<void> {
  // Database будет закрыт при завершении приложения
  // expo-sqlite SDK 52 управляет соединением автоматически
  _db = null;
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

function mapWorkOrderRow(row: any): WorkOrder {
  return {
    id: row.id,
    schedule_id: row.schedule_id ?? undefined,
    device_id: row.device_id,
    device_name: row.device_name ?? undefined,
    site_name: row.site_name ?? undefined,
    type: row.type,
    status: row.status,
    priority: row.priority,
    assigned_to: row.assigned_to ?? undefined,
    sla_deadline: row.sla_deadline ?? undefined,
    checklist: JSON.parse(row.checklist || '[]'),
    started_at: row.started_at ?? undefined,
    completed_at: row.completed_at ?? undefined,
    notes: row.notes ?? undefined,
    photos: JSON.parse(row.photos || '[]'),
    parts_used: JSON.parse(row.parts_used || '[]'),
    created_by: row.created_by ?? undefined,
    created_at: row.created_at,
    updated_at: row.updated_at,
    device_name_display: row.device_name_display ?? undefined,
    assignee_name: row.assignee_name ?? undefined,
    sla_status: row.sla_status ?? undefined,
  };
}
