import { appSchema, tableSchema } from '@nozbe/watermelondb';

/**
 * WatermelonDB schema for CCTV Health Monitor
 *
 * Maps the existing SQLite schema from offlineStorage.ts to WatermelonDB's
 * reactive, sync-ready ORM layer.
 *
 * Tables:
 *   work_orders       — CMMS work orders with full metadata
 *   devices           — CCTV camera / edge device catalog
 *   sites             — Installation sites / geofences
 *   pending_mutations — Offline mutation queue for background sync
 *
 * Compliance: СТБ IEC 62443 SL-3, ISO 27001 A.12.4 (audit trail)
 */

export const schema = appSchema({
  version: 1,
  tables: [
    // ── Work Orders ──────────────────────────────────────────────
    tableSchema({
      name: 'work_orders',
      columns: [
        // Core identifiers
        { name: 'schedule_id', type: 'string', isOptional: true },
        { name: 'device_id', type: 'string' },
        { name: 'device_name', type: 'string', isOptional: true },
        { name: 'site_name', type: 'string', isOptional: true },

        // Work order metadata
        { name: 'type', type: 'string' },               // preventive | corrective | emergency
        { name: 'status', type: 'string' },              // open | in_progress | completed | cancelled
        { name: 'priority', type: 'string' },            // critical | high | medium | low
        { name: 'assigned_to', type: 'string', isOptional: true },
        { name: 'sla_deadline', type: 'string', isOptional: true },

        // Payload (JSON-encoded arrays)
        { name: 'checklist', type: 'string' },           // ChecklistItem[]
        { name: 'started_at', type: 'string', isOptional: true },
        { name: 'completed_at', type: 'string', isOptional: true },
        { name: 'notes', type: 'string', isOptional: true },
        { name: 'photos', type: 'string' },              // string[]  (JSON)
        { name: 'parts_used', type: 'string' },          // PartUsage[] (JSON)

        // Audit trail (ISO 27001 A.12.4)
        { name: 'created_by', type: 'string', isOptional: true },
        { name: 'created_at', type: 'string' },
        { name: 'updated_at', type: 'string' },

        // Display / UX helpers
        { name: 'device_name_display', type: 'string', isOptional: true },
        { name: 'assignee_name', type: 'string', isOptional: true },
        { name: 'sla_status', type: 'string', isOptional: true },

        // Sync metadata
        { name: 'synced_at', type: 'string', isOptional: true },
      ],
    }),

    // ── Devices ─────────────────────────────────────────────────
    tableSchema({
      name: 'devices',
      columns: [
        { name: 'name', type: 'string' },
        { name: 'device_type', type: 'string' },
        { name: 'status', type: 'string' },
        { name: 'site_name', type: 'string', isOptional: true },
        { name: 'latitude', type: 'number' },
        { name: 'longitude', type: 'number' },
        { name: 'health', type: 'string' },
        { name: 'updated_at', type: 'string' },
        { name: 'synced_at', type: 'string', isOptional: true },
      ],
    }),

    // ── Sites ───────────────────────────────────────────────────
    tableSchema({
      name: 'sites',
      columns: [
        { name: 'name', type: 'string' },
        { name: 'address', type: 'string', isOptional: true },
        { name: 'latitude', type: 'number' },
        { name: 'longitude', type: 'number' },
        { name: 'updated_at', type: 'string' },
        { name: 'synced_at', type: 'string', isOptional: true },
      ],
    }),

    // ── Pending Mutations (offline sync queue) ──────────────────
    tableSchema({
      name: 'pending_mutations',
      columns: [
        { name: 'entity_type', type: 'string' },    // work_order | device | site
        { name: 'entity_id', type: 'string' },
        { name: 'mutation_type', type: 'string' },  // create | update | delete
        { name: 'payload', type: 'string' },         // JSON-encoded diff / body
        { name: 'timestamp', type: 'number' },
        { name: 'retry_count', type: 'number' },
        { name: 'last_error', type: 'string', isOptional: true },
      ],
    }),
  ],
});
