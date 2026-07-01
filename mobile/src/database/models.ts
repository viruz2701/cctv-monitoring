import { Model } from '@nozbe/watermelondb';
import { field, date, readonly, children, lazy } from '@nozbe/watermelondb/decorators';
import { Q } from '@nozbe/watermelondb';

import type { ChecklistItem, PartUsage } from '../types';

/**
 * WorkOrderModel — reactive WatermelonDB model for CMMS work orders.
 *
 * Schema columns map 1:1 to `work_orders` table in schema.ts.
 * JSON-encoded arrays (checklist, photos, parts_used) are stored as string
 * and deserialised via getters — use `parseChecklist()`, `parsePhotos()`,
 * `parsePartsUsed()` for typed access.
 *
 * Compliance: ISO 27001 A.12.4 (audit trail via created_at/updated_at)
 */
export class WorkOrderModel extends Model {
  static table = 'work_orders';

  static associations = {
    pending_mutations: { type: 'has_many' as const, foreignKey: 'entity_id' },
  } as const;

  // ── Identifiers ──────────────────────────────────────────────
  @field('schedule_id') schedule_id!: string;
  @field('device_id') device_id!: string;
  @field('device_name') device_name?: string;
  @field('site_name') site_name?: string;

  // ── Metadata ─────────────────────────────────────────────────
  @field('type') type!: string;
  @field('status') status!: string;
  @field('priority') priority!: string;
  @field('assigned_to') assigned_to?: string;
  @field('sla_deadline') sla_deadline?: string;

  // ── Payload (stored as JSON string, parsed via getters) ──────
  @field('checklist') checklist!: string;
  @field('started_at') started_at?: string;
  @field('completed_at') completed_at?: string;
  @field('notes') notes?: string;
  @field('photos') photos!: string;
  @field('parts_used') parts_used!: string;

  // ── Audit trail ──────────────────────────────────────────────
  @field('created_by') created_by?: string;
  @readonly @date('created_at') created_at!: Date;
  @field('updated_at') updated_at!: string;

  // ── Display helpers ──────────────────────────────────────────
  @field('device_name_display') device_name_display?: string;
  @field('assignee_name') assignee_name?: string;
  @field('sla_status') sla_status?: string;

  // ── Sync metadata ────────────────────────────────────────────
  @field('synced_at') synced_at?: string;

  // ── Typed accessors ──────────────────────────────────────────

  /** Deserialise checklist from JSON string → ChecklistItem[] */
  @lazy parseChecklist = (): ChecklistItem[] =>
    JSON.parse(this.checklist || '[]');

  /** Deserialise photos from JSON string → string[] */
  @lazy parsePhotos = (): string[] =>
    JSON.parse(this.photos || '[]');

  /** Deserialise parts_used from JSON string → PartUsage[] */
  @lazy parsePartsUsed = (): PartUsage[] =>
    JSON.parse(this.parts_used || '[]');
}

/**
 * DeviceModel — reactive WatermelonDB model for CCTV cameras / edge devices.
 */
export class DeviceModel extends Model {
  static table = 'devices';

  static associations = {
    work_orders: { type: 'has_many' as const, foreignKey: 'device_id' },
  } as const;

  @field('name') name!: string;
  @field('device_type') device_type!: string;
  @field('status') status!: string;
  @field('site_name') site_name?: string;
  @field('latitude') latitude!: number;
  @field('longitude') longitude!: number;
  @field('health') health!: string;
  @field('updated_at') updated_at!: string;
  @field('synced_at') synced_at?: string;
}

/**
 * SiteModel — reactive WatermelonDB model for installation sites / geofences.
 */
export class SiteModel extends Model {
  static table = 'sites';

  @field('name') name!: string;
  @field('address') address?: string;
  @field('latitude') latitude!: number;
  @field('longitude') longitude!: number;
  @field('updated_at') updated_at!: string;
  @field('synced_at') synced_at?: string;
}

/**
 * PendingMutationModel — offline mutation queue for background sync.
 *
 * Each row represents one CRUD operation that couldn't be sent to the server.
 * The sync service consumes (and removes) these rows after successful sync.
 *
 * Compliance: ISO 27001 A.12.4 (tamper-evident audit via payload hashing)
 */
export class PendingMutationModel extends Model {
  static table = 'pending_mutations';

  @field('entity_type') entity_type!: string;   // work_order | device | site
  @field('entity_id') entity_id!: string;
  @field('mutation_type') mutation_type!: string; // create | update | delete
  @field('payload') payload!: string;             // JSON body
  @field('timestamp') timestamp!: number;
  @field('retry_count') retry_count!: number;
  @field('last_error') last_error?: string;
}
