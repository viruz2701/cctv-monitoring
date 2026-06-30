// ═══════════════════════════════════════════════════════════════════════
// Custom Fields API
// P2-FIELDS: Custom Fields Advanced (Shelf.nu-level)
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Field Types ──────────────────────────────────────────────────────

export type FieldType =
  | 'text'
  | 'number'
  | 'date'
  | 'dropdown'
  | 'multi_select'
  | 'url'
  | 'email'
  | 'barcode'
  | 'signature'
  | 'file_upload'
  | 'checkbox'
  | 'radio'
  | 'textarea'
  | 'time'
  | 'color'
  | 'user';

export type EntityType = 'device' | 'work_order' | 'site' | 'part';

// ─── Types ────────────────────────────────────────────────────────────

export interface ValidationRule {
  min?: number;
  max?: number;
  min_len?: number;
  max_len?: number;
  regex?: string;
  custom?: string;
}

export interface FieldCondition {
  field_id: string;
  operator: 'eq' | 'neq' | 'gt' | 'lt' | 'gte' | 'lte' | 'in' | 'contains';
  value: unknown;
}

export interface FieldDefinition {
  id: string;
  entity_type: EntityType;
  field_type: FieldType;
  name: string;
  label: string;
  description?: string;
  required: boolean;
  options?: string[];
  validation?: ValidationRule;
  visibility?: FieldCondition;
  group_id?: string;
  sort_order: number;
  default_value?: unknown;
  placeholder?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface FieldGroup {
  id: string;
  name: string;
  description: string;
  entity_type: EntityType;
  sort_order: number;
  is_collapsible: boolean;
  is_collapsed: boolean;
  created_at: string;
  updated_at: string;
}

export interface FieldDefinitionWithValue extends FieldDefinition {
  value?: unknown;
}

export interface CreateFieldDefinitionRequest {
  entity_type: EntityType;
  field_type: FieldType;
  name: string;
  label: string;
  description?: string;
  required?: boolean;
  options?: string[];
  validation?: ValidationRule;
  visibility?: FieldCondition;
  group_id?: string;
  sort_order?: number;
  default_value?: unknown;
  placeholder?: string;
}

export interface UpdateFieldDefinitionRequest {
  label?: string;
  description?: string;
  required?: boolean;
  options?: string[];
  validation?: ValidationRule;
  visibility?: FieldCondition;
  group_id?: string;
  sort_order?: number;
  default_value?: unknown;
  placeholder?: string;
  is_active?: boolean;
}

export interface CreateGroupRequest {
  name: string;
  description?: string;
  entity_type: EntityType;
  sort_order?: number;
  is_collapsible?: boolean;
  is_collapsed?: boolean;
}

export interface UpdateGroupRequest {
  name?: string;
  description?: string;
  sort_order?: number;
  is_collapsible?: boolean;
  is_collapsed?: boolean;
}

export interface BulkUpdateValuesRequest {
  values: Record<string, unknown>;
}

// ─── Field Type Metadata ──────────────────────────────────────────────

export const FIELD_TYPE_LABELS: Record<FieldType, string> = {
  text: 'Text',
  number: 'Number',
  date: 'Date',
  dropdown: 'Dropdown',
  multi_select: 'Multi-Select',
  url: 'URL',
  email: 'Email',
  barcode: 'Barcode',
  signature: 'Signature',
  file_upload: 'File Upload',
  checkbox: 'Checkbox',
  radio: 'Radio',
  textarea: 'Textarea',
  time: 'Time',
  color: 'Color',
  user: 'User',
};

export const FIELD_TYPE_CATEGORIES: Record<string, FieldType[]> = {
  text: ['text', 'textarea', 'url', 'email', 'barcode', 'color'],
  numeric: ['number'],
  temporal: ['date', 'time'],
  select: ['dropdown', 'multi_select', 'radio', 'checkbox'],
  media: ['signature', 'file_upload'],
  reference: ['user'],
};

// ─── API Methods ──────────────────────────────────────────────────────

export const customFieldsApi = {
  // ── Field Definitions ──

  getDefinitions(params?: {
    entity_type?: EntityType;
    group_id?: string;
    active_only?: boolean;
    limit?: number;
    offset?: number;
  }): Promise<FieldDefinition[]> {
    const searchParams = new URLSearchParams();
    if (params?.entity_type) searchParams.set('entity_type', params.entity_type);
    if (params?.group_id) searchParams.set('group_id', params.group_id);
    if (params?.active_only) searchParams.set('active_only', 'true');
    if (params?.limit) searchParams.set('limit', String(params.limit));
    if (params?.offset) searchParams.set('offset', String(params.offset));
    const qs = searchParams.toString();
    return request<FieldDefinition[]>(`/custom-fields/definitions${qs ? `?${qs}` : ''}`);
  },

  getDefinition(id: string): Promise<FieldDefinition> {
    return request<FieldDefinition>(`/custom-fields/definitions/${id}`);
  },

  createDefinition(data: CreateFieldDefinitionRequest): Promise<FieldDefinition> {
    return request<FieldDefinition>('/custom-fields/definitions', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  updateDefinition(id: string, data: UpdateFieldDefinitionRequest): Promise<FieldDefinition> {
    return request<FieldDefinition>(`/custom-fields/definitions/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  deleteDefinition(id: string): Promise<{ status: string }> {
    return request<{ status: string }>(`/custom-fields/definitions/${id}`, {
      method: 'DELETE',
    });
  },

  // ── Field Groups ──

  getGroups(params?: { entity_type?: EntityType }): Promise<FieldGroup[]> {
    const qs = params?.entity_type ? `?entity_type=${params.entity_type}` : '';
    return request<FieldGroup[]>(`/custom-fields/groups${qs}`);
  },

  getGroup(id: string): Promise<FieldGroup> {
    return request<FieldGroup>(`/custom-fields/groups/${id}`);
  },

  createGroup(data: CreateGroupRequest): Promise<FieldGroup> {
    return request<FieldGroup>('/custom-fields/groups', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  updateGroup(id: string, data: UpdateGroupRequest): Promise<FieldGroup> {
    return request<FieldGroup>(`/custom-fields/groups/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  deleteGroup(id: string): Promise<{ status: string }> {
    return request<{ status: string }>(`/custom-fields/groups/${id}`, {
      method: 'DELETE',
    });
  },

  // ── Field Values ──

  getFieldValues(
    entityType: EntityType,
    entityId: string,
  ): Promise<FieldDefinitionWithValue[]> {
    return request<FieldDefinitionWithValue[]>(
      `/custom-fields/values/${entityType}/${entityId}`,
    );
  },

  bulkUpdateValues(
    entityType: EntityType,
    entityId: string,
    data: BulkUpdateValuesRequest,
  ): Promise<{ status: string; fields_updated: number }> {
    return request<{ status: string; fields_updated: number }>(
      `/custom-fields/values/${entityType}/${entityId}`,
      {
        method: 'PUT',
        body: JSON.stringify(data),
      },
    );
  },
};
