import { Database } from '@nozbe/watermelondb';
import SQLiteAdapter from '@nozbe/watermelondb/adapters/sqlite';

import { schema } from './schema';
import {
  WorkOrderModel,
  DeviceModel,
  SiteModel,
  PendingMutationModel,
} from './models';

/**
 * Database provider — singleton WatermelonDB instance.
 *
 * Usage:
 * ```ts
 * import { getDatabase } from '../database';
 * const db = getDatabase();
 * const workOrders = await db.get('work_orders').query().fetch();
 * ```
 *
 * The adapter uses SQLite via `@nozbe/watermelondb/adapters/sqlite`.
 * JSI mode is enabled by default for synchronous C++ bindings.
 *
 * Compliance: СТБ IEC 62443 SL-3 (Zone 3 — Application)
 *             ISO 27001 A.12.4 (all mutations logged via audit_log)
 */

// ── Singleton ──────────────────────────────────────────────────

let databaseInstance: Database | null = null;

/**
 * Return the shared WatermelonDB Database singleton.
 * Creates the instance on first call with SQLite adapter and all model classes.
 */
export function getDatabase(): Database {
  if (!databaseInstance) {
    const adapter = new SQLiteAdapter({
      schema,
      dbName: 'cctv_watermelon',
      // JSI enables synchronous C++ SQLite access (faster, recommended)
      jsi: true,
      // Use database migrations if schema version changes
      migrations: undefined,
    });

    databaseInstance = new Database({
      adapter,
      modelClasses: [
        WorkOrderModel,
        DeviceModel,
        SiteModel,
        PendingMutationModel,
      ],
    });
  }

  return databaseInstance;
}

/**
 * Reset the database singleton (for testing or logout).
 * Clears the instance reference so the next getDatabase() call
 * creates a fresh connection.
 */
export function resetDatabase(): void {
  databaseInstance = null;
}

/**
 * Check if the database has been initialised.
 */
export function isDatabaseReady(): boolean {
  return databaseInstance !== null;
}

// ── Re-export models for convenience ───────────────────────────

export {
  WorkOrderModel,
  DeviceModel,
  SiteModel,
  PendingMutationModel,
} from './models';

export { schema } from './schema';
