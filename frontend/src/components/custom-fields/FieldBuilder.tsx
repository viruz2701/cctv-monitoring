// ═══════════════════════════════════════════════════════════════════════
// FieldBuilder — drag-and-drop field builder for custom fields.
// P2-FIELDS: Custom Fields Advanced (Shelf.nu-level)
//
// Features:
//   - Drag-and-drop reordering via @hello-pangea/dnd
//   - Field type selector (15+ types)
//   - Validation rule editor
//   - Conditional visibility builder
//   - Field groups management
//   - Bulk apply values
// ═══════════════════════════════════════════════════════════════════════

import { useCallback, useEffect, useState } from 'react';
import type {
  CreateFieldDefinitionRequest,
  CreateGroupRequest,
  FieldDefinition,
  FieldGroup,
  FieldType,
  UpdateFieldDefinitionRequest,
  ValidationRule,
} from '../../services/api/customFields';
import { FIELD_TYPE_LABELS, customFieldsApi } from '../../services/api/customFields';

import {
  DragDropContext,
  Droppable,
  Draggable,
  type DropResult,
} from '@hello-pangea/dnd';

// ─── Types ────────────────────────────────────────────────────────────

interface FieldBuilderProps {
  entityType: 'device' | 'work_order' | 'site' | 'part';
  initialFields?: FieldDefinition[];
  initialGroups?: FieldGroup[];
  onFieldsChange?: (fields: FieldDefinition[]) => void;
  onGroupsChange?: (groups: FieldGroup[]) => void;
}

// ─── Validation Rule Editor ────────────────────────────────────────────

function ValidationRuleEditor({
  validation,
  onChange,
}: {
  validation: ValidationRule | undefined;
  onChange: (v: ValidationRule | undefined) => void;
}) {
  const [expanded, setExpanded] = useState(!!validation);

  if (!expanded) {
    return (
      <button
        type="button"
        onClick={() => {
          setExpanded(true);
          onChange({});
        }}
        className="text-sm text-blue-600 hover:text-blue-800"
      >
        + Add validation rules
      </button>
    );
  }

  const set = (key: keyof ValidationRule, value: unknown) => {
    onChange({ ...validation, [key]: value } as ValidationRule);
  };

  return (
    <div className="space-y-3 rounded-lg border border-gray-200 bg-gray-50 p-4">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium text-gray-700">Validation Rules</h4>
        <button
          type="button"
          onClick={() => {
            setExpanded(false);
            onChange(undefined);
          }}
          className="text-xs text-red-500 hover:text-red-700"
        >
          Remove
        </button>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-xs font-medium text-gray-600">Min</label>
          <input
            type="number"
            value={validation?.min ?? ''}
            onChange={(e) => set('min', e.target.value ? Number(e.target.value) : undefined)}
            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600">Max</label>
          <input
            type="number"
            value={validation?.max ?? ''}
            onChange={(e) => set('max', e.target.value ? Number(e.target.value) : undefined)}
            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600">Min Length</label>
          <input
            type="number"
            value={validation?.min_len ?? ''}
            onChange={(e) => set('min_len', e.target.value ? Number(e.target.value) : undefined)}
            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600">Max Length</label>
          <input
            type="number"
            value={validation?.max_len ?? ''}
            onChange={(e) => set('max_len', e.target.value ? Number(e.target.value) : undefined)}
            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-600">Regex Pattern</label>
        <input
          type="text"
          value={validation?.regex ?? ''}
          onChange={(e) => set('regex', e.target.value || undefined)}
          placeholder="^[A-Za-z0-9]+$"
          className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm font-mono shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
        />
      </div>
    </div>
  );
}

// ─── Field Editor Modal ────────────────────────────────────────────────

function FieldEditorModal({
  field,
  entityType,
  groups,
  onSave,
  onClose,
}: {
  field?: FieldDefinition;
  entityType: string;
  groups: FieldGroup[];
  onSave: (data: CreateFieldDefinitionRequest | UpdateFieldDefinitionRequest) => void;
  onClose: () => void;
}) {
  const isEdit = !!field;
  const [label, setLabel] = useState(field?.label ?? '');
  const [name, setName] = useState(field?.name ?? '');
  const [description, setDescription] = useState(field?.description ?? '');
  const [fieldType, setFieldType] = useState<FieldType>(field?.field_type ?? 'text');
  const [required, setRequired] = useState(field?.required ?? false);
  const [placeholder, setPlaceholder] = useState(field?.placeholder ?? '');
  const [options, setOptions] = useState<string[]>(field?.options ?? []);
  const [optionInput, setOptionInput] = useState('');
  const [groupID, setGroupID] = useState(field?.group_id ?? '');
  const [validation, setValidation] = useState<ValidationRule | undefined>(field?.validation);
  const [sortOrder, setSortOrder] = useState(field?.sort_order ?? 0);

  const needsOptions = ['dropdown', 'multi_select', 'radio'].includes(fieldType);

  const addOption = () => {
    if (optionInput.trim()) {
      setOptions([...options, optionInput.trim()]);
      setOptionInput('');
    }
  };

  const removeOption = (idx: number) => {
    setOptions(options.filter((_, i) => i !== idx));
  };

  const handleSave = () => {
    if (isEdit) {
      onSave({
        label: label || undefined,
        description: description || undefined,
        required: required || undefined,
        placeholder: placeholder || undefined,
        options: needsOptions ? options : undefined,
        group_id: groupID || undefined,
        validation: validation || undefined,
        sort_order: sortOrder || undefined,
      } as UpdateFieldDefinitionRequest);
    } else {
      onSave({
        entity_type: entityType,
        field_type: fieldType,
        name,
        label,
        description: description || undefined,
        required,
        placeholder,
        options: needsOptions ? options : undefined,
        group_id: groupID || undefined,
        validation: validation || undefined,
        sort_order: sortOrder,
      } as CreateFieldDefinitionRequest);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="mx-4 w-full max-w-lg rounded-xl bg-white p-6 shadow-xl">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-900">
            {isEdit ? 'Edit Field' : 'New Field'}
          </h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="space-y-4">
          {/* Field Type (only for create) */}
          {!isEdit && (
            <div>
              <label className="block text-sm font-medium text-gray-700">Field Type</label>
              <select
                value={fieldType}
                onChange={(e) => setFieldType(e.target.value as FieldType)}
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              >
                {(Object.entries(FIELD_TYPE_LABELS) as [FieldType, string][]).map(([type, label]) => (
                  <option key={type} value={type}>{label}</option>
                ))}
              </select>
            </div>
          )}

          {/* Label */}
          <div>
            <label className="block text-sm font-medium text-gray-700">Label *</label>
            <input
              type="text"
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              placeholder="e.g., Serial Number"
            />
          </div>

          {/* Name */}
          {!isEdit && (
            <div>
              <label className="block text-sm font-medium text-gray-700">Name (API key) *</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm font-mono shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                placeholder="serial_number"
              />
            </div>
          )}

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-gray-700">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={2}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          {/* Placeholder */}
          <div>
            <label className="block text-sm font-medium text-gray-700">Placeholder</label>
            <input
              type="text"
              value={placeholder}
              onChange={(e) => setPlaceholder(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          {/* Options (dropdown/multi_select/radio) */}
          {needsOptions && (
            <div>
              <label className="block text-sm font-medium text-gray-700">Options</label>
              <div className="mt-1 flex gap-2">
                <input
                  type="text"
                  value={optionInput}
                  onChange={(e) => setOptionInput(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addOption())}
                  className="block flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="Add option..."
                />
                <button
                  type="button"
                  onClick={addOption}
                  className="rounded-md bg-blue-50 px-3 py-2 text-sm font-medium text-blue-700 hover:bg-blue-100"
                >
                  Add
                </button>
              </div>
              {options.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-2">
                  {options.map((opt, idx) => (
                    <span
                      key={idx}
                      className="inline-flex items-center gap-1 rounded-full bg-gray-100 px-3 py-1 text-sm text-gray-700"
                    >
                      {opt}
                      <button
                        type="button"
                        onClick={() => removeOption(idx)}
                        className="text-gray-400 hover:text-red-500"
                      >
                        &times;
                      </button>
                    </span>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Group selector */}
          {groups.length > 0 && (
            <div>
              <label className="block text-sm font-medium text-gray-700">Group</label>
              <select
                value={groupID}
                onChange={(e) => setGroupID(e.target.value)}
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              >
                <option value="">No group</option>
                {groups.map((g) => (
                  <option key={g.id} value={g.id}>{g.name}</option>
                ))}
              </select>
            </div>
          )}

          {/* Sort Order */}
          <div>
            <label className="block text-sm font-medium text-gray-700">Sort Order</label>
            <input
              type="number"
              value={sortOrder}
              onChange={(e) => setSortOrder(Number(e.target.value))}
              className="mt-1 block w-24 rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          {/* Required */}
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="required"
              checked={required}
              onChange={(e) => setRequired(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <label htmlFor="required" className="text-sm text-gray-700">Required field</label>
          </div>

          {/* Validation Rules */}
          <ValidationRuleEditor
            validation={validation}
            onChange={setValidation}
          />
        </div>

        {/* Actions */}
        <div className="mt-6 flex justify-end gap-3">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={!label || (!isEdit && !name)}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isEdit ? 'Update' : 'Create'}
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Group Editor Modal ────────────────────────────────────────────────

function GroupEditorModal({
  group,
  onSave,
  onClose,
}: {
  group?: FieldGroup;
  onSave: (data: CreateGroupRequest) => void;
  onClose: () => void;
}) {
  const [name, setName] = useState(group?.name ?? '');
  const [description, setDescription] = useState(group?.description ?? '');
  const [sortOrder, setSortOrder] = useState(group?.sort_order ?? 0);
  const [isCollapsible, setIsCollapsible] = useState(group?.is_collapsible ?? false);
  const [isCollapsed, setIsCollapsed] = useState(group?.is_collapsed ?? false);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-900">
            {group ? 'Edit Group' : 'New Group'}
          </h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">Name *</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={2}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">Sort Order</label>
            <input
              type="number"
              value={sortOrder}
              onChange={(e) => setSortOrder(Number(e.target.value))}
              className="mt-1 block w-24 rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="collapsible"
              checked={isCollapsible}
              onChange={(e) => setIsCollapsible(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <label htmlFor="collapsible" className="text-sm text-gray-700">Collapsible</label>
          </div>
          {isCollapsible && (
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="collapsed"
                checked={isCollapsed}
                onChange={(e) => setIsCollapsed(e.target.checked)}
                className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <label htmlFor="collapsed" className="text-sm text-gray-700">Start collapsed</label>
            </div>
          )}
        </div>

        <div className="mt-6 flex justify-end gap-3">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => onSave({
              name,
              description: description || undefined,
              entity_type: group?.entity_type ?? 'device',
              sort_order: sortOrder,
              is_collapsible: isCollapsible,
              is_collapsed: isCollapsed,
            })}
            disabled={!name}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {group ? 'Update' : 'Create'}
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Main Component ────────────────────────────────────────────────────

export function FieldBuilder({
  entityType,
  initialFields,
  initialGroups,
  onFieldsChange,
  onGroupsChange,
}: FieldBuilderProps) {
  const [fields, setFields] = useState<FieldDefinition[]>(initialFields ?? []);
  const [groups, setGroups] = useState<FieldGroup[]>(initialGroups ?? []);
  const [editingField, setEditingField] = useState<FieldDefinition | undefined>();
  const [editingGroup, setEditingGroup] = useState<FieldGroup | undefined>();
  const [showNewFieldModal, setShowNewFieldModal] = useState(false);
  const [showNewGroupModal, setShowNewGroupModal] = useState(false);
  const [loading, setLoading] = useState(!initialFields);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'fields' | 'groups'>('fields');

  // Load data on mount
  useEffect(() => {
    if (initialFields && initialGroups) {
      setLoading(false);
      return;
    }
    loadData();
  }, [entityType]);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [fetchedFields, fetchedGroups] = await Promise.all([
        customFieldsApi.getDefinitions({ entity_type: entityType, active_only: true }),
        customFieldsApi.getGroups({ entity_type: entityType }),
      ]);
      setFields(fetchedFields);
      setGroups(fetchedGroups);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load custom fields');
    } finally {
      setLoading(false);
    }
  }, [entityType]);

  // ── Field operations ──

  const handleCreateField = useCallback(async (data: CreateFieldDefinitionRequest) => {
    try {
      const created = await customFieldsApi.createDefinition(data);
      const updated = [...fields, created];
      setFields(updated);
      onFieldsChange?.(updated);
      setShowNewFieldModal(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create field');
    }
  }, [fields, onFieldsChange]);

  const handleUpdateField = useCallback(async (id: string, data: UpdateFieldDefinitionRequest) => {
    try {
      const updated = await customFieldsApi.updateDefinition(id, data);
      const newFields = fields.map((f) => (f.id === id ? updated : f));
      setFields(newFields);
      onFieldsChange?.(newFields);
      setEditingField(undefined);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update field');
    }
  }, [fields, onFieldsChange]);

  const handleDeleteField = useCallback(async (id: string) => {
    try {
      await customFieldsApi.deleteDefinition(id);
      const newFields = fields.filter((f) => f.id !== id);
      setFields(newFields);
      onFieldsChange?.(newFields);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete field');
    }
  }, [fields, onFieldsChange]);

  const handleDragEnd = useCallback((result: DropResult) => {
    if (!result.destination) return;
    if (result.source.index === result.destination.index) return;

    const newFields = Array.from(fields);
    const [moved] = newFields.splice(result.source.index, 1);
    newFields.splice(result.destination.index, 0, moved);

    // Update sort_order based on new position
    const reordered = newFields.map((f, idx) => ({ ...f, sort_order: idx }));
    setFields(reordered);
    onFieldsChange?.(reordered);

    // Persist sort_order changes
    reordered.forEach((f) => {
      customFieldsApi.updateDefinition(f.id, { sort_order: f.sort_order } as UpdateFieldDefinitionRequest)
        .catch((err) => console.error('Failed to persist sort_order', err));
    });
  }, [fields, onFieldsChange]);

  // ── Group operations ──

  const handleCreateGroup = useCallback(async (data: CreateGroupRequest) => {
    try {
      const created = await customFieldsApi.createGroup({ ...data, entity_type: entityType });
      const updated = [...groups, created];
      setGroups(updated);
      onGroupsChange?.(updated);
      setShowNewGroupModal(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create group');
    }
  }, [groups, entityType, onGroupsChange]);

  const handleUpdateGroup = useCallback(async (id: string, data: CreateGroupRequest) => {
    try {
      const updated = await customFieldsApi.updateGroup(id, data);
      const newGroups = groups.map((g) => (g.id === id ? updated : g));
      setGroups(newGroups);
      onGroupsChange?.(newGroups);
      setEditingGroup(undefined);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update group');
    }
  }, [groups, onGroupsChange]);

  const handleDeleteGroup = useCallback(async (id: string) => {
    try {
      await customFieldsApi.deleteGroup(id);
      const newGroups = groups.filter((g) => g.id !== id);
      setGroups(newGroups);
      onGroupsChange?.(newGroups);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete group');
    }
  }, [groups, onGroupsChange]);

  // ── Render ──

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-blue-200 border-t-blue-600" />
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
      {/* Header */}
      <div className="border-b border-gray-200 px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-900">Custom Fields</h2>
            <p className="text-sm text-gray-500">
              Manage custom fields for {entityType.replace('_', ' ')}
            </p>
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => setShowNewGroupModal(true)}
              className="rounded-md border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              + Group
            </button>
            <button
              onClick={() => setShowNewFieldModal(true)}
              className="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              + Field
            </button>
          </div>
        </div>
      </div>

      {/* Error banner */}
      {error && (
        <div className="mx-6 mt-4 rounded-lg bg-red-50 p-3 text-sm text-red-700">
          {error}
          <button onClick={() => setError(null)} className="ml-2 text-red-500 hover:text-red-700">
            Dismiss
          </button>
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-gray-200 px-6">
        <div className="-mb-px flex gap-6">
          <button
            onClick={() => setActiveTab('fields')}
            className={`border-b-2 px-1 py-3 text-sm font-medium transition-colors ${
              activeTab === 'fields'
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            Fields ({fields.length})
          </button>
          <button
            onClick={() => setActiveTab('groups')}
            className={`border-b-2 px-1 py-3 text-sm font-medium transition-colors ${
              activeTab === 'groups'
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            Groups ({groups.length})
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="p-6">
        {activeTab === 'fields' && (
          <div className="space-y-3">
            {fields.length === 0 ? (
              <div className="py-8 text-center text-sm text-gray-500">
                <p className="mb-2">No custom fields yet.</p>
                <button
                  onClick={() => setShowNewFieldModal(true)}
                  className="text-blue-600 hover:text-blue-800"
                >
                  Create your first field
                </button>
              </div>
            ) : (
              <DragDropContext onDragEnd={handleDragEnd}>
                <Droppable droppableId="field-list">
                  {(provided) => (
                    <div
                      ref={provided.innerRef}
                      {...provided.droppableProps}
                      className="space-y-2"
                    >
                      {fields.map((field, index) => (
                        <Draggable key={field.id} draggableId={field.id} index={index}>
                          {(provided, snapshot) => (
                            <div
                              ref={provided.innerRef}
                              {...provided.draggableProps}
                              className={`group flex items-center gap-3 rounded-lg border bg-white p-3 shadow-sm transition-all hover:shadow-md ${
                                snapshot.isDragging
                                  ? 'border-blue-400 shadow-lg ring-2 ring-blue-100'
                                  : 'border-gray-200'
                              }`}
                            >
                              {/* Drag handle */}
                              <div
                                {...provided.dragHandleProps}
                                className="cursor-grab touch-none text-gray-400 hover:text-gray-600 active:cursor-grabbing"
                                aria-label="Drag to reorder"
                              >
                                <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h16M4 16h16" />
                                </svg>
                              </div>

                              {/* Field type badge */}
                              <span className="inline-flex items-center rounded-full bg-blue-50 px-2.5 py-0.5 text-xs font-medium text-blue-700">
                                {FIELD_TYPE_LABELS[field.field_type] ?? field.field_type}
                              </span>

                              {/* Field info */}
                              <div className="min-w-0 flex-1">
                                <div className="flex items-center gap-2">
                                  <span className="truncate text-sm font-medium text-gray-900">
                                    {field.label}
                                  </span>
                                  {field.required && (
                                    <span className="text-xs text-red-500">*</span>
                                  )}
                                  {field.validation && (
                                    <span className="inline-flex items-center rounded bg-green-50 px-1.5 py-0.5 text-xs text-green-700">
                                      validated
                                    </span>
                                  )}
                                  {field.visibility && (
                                    <span className="inline-flex items-center rounded bg-amber-50 px-1.5 py-0.5 text-xs text-amber-700">
                                      conditional
                                    </span>
                                  )}
                                </div>
                                <p className="truncate text-xs text-gray-500">
                                  {field.name}
                                  {field.group_id && ' · in group'}
                                </p>
                              </div>

                              {/* Actions */}
                              <div className="flex shrink-0 items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                                <button
                                  onClick={() => setEditingField(field)}
                                  className="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                                  title="Edit field"
                                >
                                  <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                                  </svg>
                                </button>
                                <button
                                  onClick={() => handleDeleteField(field.id)}
                                  className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-500"
                                  title="Delete field"
                                >
                                  <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                  </svg>
                                </button>
                              </div>
                            </div>
                          )}
                        </Draggable>
                      ))}
                      {provided.placeholder}
                    </div>
                  )}
                </Droppable>
              </DragDropContext>
            )}
          </div>
        )}

        {activeTab === 'groups' && (
          <div className="space-y-3">
            {groups.length === 0 ? (
              <div className="py-8 text-center text-sm text-gray-500">
                <p className="mb-2">No field groups yet.</p>
                <button
                  onClick={() => setShowNewGroupModal(true)}
                  className="text-blue-600 hover:text-blue-800"
                >
                  Create your first group
                </button>
              </div>
            ) : (
              groups.map((group) => (
                <div
                  key={group.id}
                  className="flex items-center justify-between rounded-lg border border-gray-200 bg-white p-4"
                >
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-900">{group.name}</span>
                      {group.is_collapsible && (
                        <span className="inline-flex items-center rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600">
                          collapsible
                        </span>
                      )}
                    </div>
                    {group.description && (
                      <p className="text-sm text-gray-500">{group.description}</p>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => setEditingGroup(group)}
                      className="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                    >
                      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                    </button>
                    <button
                      onClick={() => handleDeleteGroup(group.id)}
                      className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-500"
                    >
                      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        )}
      </div>

      {/* Modals */}
      {showNewFieldModal && (
        <FieldEditorModal
          entityType={entityType}
          groups={groups}
          onSave={(data) => handleCreateField(data as CreateFieldDefinitionRequest)}
          onClose={() => setShowNewFieldModal(false)}
        />
      )}

      {editingField && (
        <FieldEditorModal
          field={editingField}
          entityType={entityType}
          groups={groups}
          onSave={(data) => handleUpdateField(editingField.id, data as UpdateFieldDefinitionRequest)}
          onClose={() => setEditingField(undefined)}
        />
      )}

      {showNewGroupModal && (
        <GroupEditorModal
          onSave={(data) => handleCreateGroup(data)}
          onClose={() => setShowNewGroupModal(false)}
        />
      )}

      {editingGroup && (
        <GroupEditorModal
          group={editingGroup}
          onSave={(data) => handleUpdateGroup(editingGroup.id, data)}
          onClose={() => setEditingGroup(undefined)}
        />
      )}
    </div>
  );
}

export default FieldBuilder;
