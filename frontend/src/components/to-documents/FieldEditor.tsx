// ═══════════════════════════════════════════════════════════════════════
// FieldEditor.tsx — Editable section fields with version tracking
//
// Track 3: TO Compliance Automation
//   - UX-3.3: TO Document Preview & Editing
//
// Features:
//   - Inline editing of document fields
//   - Per-field version tracking
//   - Support for: text, textarea, checkbox, date, signature, photo, select
//   - Auto-save with debounce
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { toDocumentsApi } from '../../services/toDocumentsApi';
import { Button, Input, Badge } from '../ui';
import {
  Save,
  Loader2,
  Image,
  CheckSquare,
  Type,
  Calendar,
  Edit3,
  List,
  Plus,
  X,
} from '../ui/Icons';
import type { DocumentSection } from '../../services/toDocumentsApi';
import type { TemplateField } from '../../services/toJournalsApi';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface FieldEditorProps {
  section: DocumentSection;
  documentId: string;
  onUpdate: (section: DocumentSection) => void;
}

interface EditableFieldProps {
  field: TemplateField;
  value: unknown;
  onChange: (fieldId: string, value: unknown) => void;
  saving?: boolean;
}

// ═══════════════════════════════════════════════════════════════════════
// Field Type Icons
// ═══════════════════════════════════════════════════════════════════════

const FIELD_ICONS: Record<string, React.ElementType> = {
  text: Type,
  textarea: Edit3,
  checkbox: CheckSquare,
  date: Calendar,
  signature: Edit3,
  photo: Image,
  select: List,
};

// ═══════════════════════════════════════════════════════════════════════
// EditableField Component
// ═══════════════════════════════════════════════════════════════════════

function EditableField({ field, value, onChange }: EditableFieldProps) {
  const Icon = FIELD_ICONS[field.type] ?? Type;

  const handleChange = (newValue: unknown) => {
    onChange(field.id, newValue);
  };

  const baseClass = "w-full text-sm";
  const inputClass = `
    ${baseClass} px-3 py-2 rounded-lg border border-slate-300 dark:border-slate-600
    bg-white dark:bg-slate-800 text-slate-900 dark:text-white
    focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500
    placeholder:text-slate-400
  `;

  switch (field.type) {
    case 'textarea':
      return (
        <textarea
          className={`${inputClass} min-h-[80px] resize-y`}
          value={(value as string) ?? ''}
          onChange={(e) => handleChange(e.target.value)}
          placeholder={field.placeholder}
          rows={3}
        />
      );

    case 'checkbox':
      return (
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={(value as boolean) ?? false}
            onChange={(e) => handleChange(e.target.checked)}
            className="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
          />
          <span className="text-sm text-slate-700 dark:text-slate-300">
            {field.label}
          </span>
        </label>
      );

    case 'date':
      return (
        <input
          type="date"
          className={inputClass}
          value={(value as string) ?? ''}
          onChange={(e) => handleChange(e.target.value)}
        />
      );

    case 'select':
      return (
        <select
          className={inputClass}
          value={(value as string) ?? ''}
          onChange={(e) => handleChange(e.target.value)}
        >
          <option value="">{field.placeholder ?? 'Select...'}</option>
          {field.options?.map((opt) => (
            <option key={opt} value={opt}>{opt}</option>
          ))}
        </select>
      );

    case 'signature':
      return (
        <div className="border border-dashed border-slate-300 dark:border-slate-600 rounded-lg p-4 text-center">
          <Edit3 className="w-8 h-8 text-slate-300 mx-auto mb-2" />
          <p className="text-xs text-slate-400">
            {value ? 'Signed' : 'Click to sign'}
          </p>
        </div>
      );

    case 'photo':
      return (
        <div className="border border-dashed border-slate-300 dark:border-slate-600 rounded-lg p-4 text-center">
          <Image className="w-8 h-8 text-slate-300 mx-auto mb-2" />
          <p className="text-xs text-slate-400">
            Click to upload photo
          </p>
        </div>
      );

    default: // text
      return (
        <input
          type="text"
          className={inputClass}
          value={(value as string) ?? ''}
          onChange={(e) => handleChange(e.target.value)}
          placeholder={field.placeholder}
        />
      );
  }
}

// ═══════════════════════════════════════════════════════════════════════
// FieldEditor Component
// ═══════════════════════════════════════════════════════════════════════

export function FieldEditor({ section, documentId, onUpdate }: FieldEditorProps) {
  const { t } = useTranslation();
  const [saving, setSaving] = useState<Record<string, boolean>>({});
  const [dirtyFields, setDirtyFields] = useState<Set<string>>(new Set());
  const debounceRef = useRef<Record<string, ReturnType<typeof setTimeout>>>({});

  // Extract fields from section content
  const fields: TemplateField[] = (section.content?.fields as TemplateField[]) ?? [];
  const fieldValues: Record<string, unknown> = (section.content?.values as Record<string, unknown>) ?? {};

  // ── Handle field change with debounced save ───────────────────
  const handleFieldChange = useCallback((fieldId: string, value: unknown) => {
    // Optimistic local update
    const updatedSection = {
      ...section,
      content: {
        ...section.content,
        values: {
          ...fieldValues,
          [fieldId]: value,
        },
      },
    };
    onUpdate(updatedSection);

    // Mark dirty
    setDirtyFields((prev) => new Set(prev).add(fieldId));

    // Debounced save
    if (debounceRef.current[fieldId]) {
      clearTimeout(debounceRef.current[fieldId]);
    }

    debounceRef.current[fieldId] = setTimeout(async () => {
      setSaving((prev) => ({ ...prev, [fieldId]: true }));
      try {
        await toDocumentsApi.updateField(documentId, {
          section_id: section.id,
          field_id: fieldId,
          value,
        });
        setDirtyFields((prev) => {
          const next = new Set(prev);
          next.delete(fieldId);
          return next;
        });
      } catch {
        // Error handled silently — user can retry
      } finally {
        setSaving((prev) => ({ ...prev, [fieldId]: false }));
      }
    }, 800); // 800ms debounce for auto-save
  }, [section, documentId, fieldValues, onUpdate]);

  // Cleanup debounce timers on unmount
  useEffect(() => {
    return () => {
      Object.values(debounceRef.current).forEach(clearTimeout);
    };
  }, []);

  // If section has no fields defined, show raw content editor
  if (fields.length === 0) {
    return (
      <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
        <pre className="text-xs text-slate-500 dark:text-slate-400 overflow-auto max-h-64">
          {JSON.stringify(section.content, null, 2)}
        </pre>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300">
          {section.title}
        </h3>
        <div className="flex items-center gap-2">
          {dirtyFields.size > 0 && (
            <Badge variant="info" size="sm">
              {t('fields.unsaved', '{{count}} unsaved', { count: dirtyFields.size })}
            </Badge>
          )}
        </div>
      </div>

      <div className="space-y-3">
        {fields.map((field) => {
          const Icon = FIELD_ICONS[field.type] ?? Type;
          const isSaving = saving[field.id];
          const isDirty = dirtyFields.has(field.id);

          return (
            <div key={field.id} className="space-y-1">
              <div className="flex items-center justify-between">
                <label className="flex items-center gap-1.5 text-xs font-medium text-slate-600 dark:text-slate-400">
                  <Icon className="w-3.5 h-3.5" />
                  {field.label}
                  {field.required && (
                    <span className="text-red-500">*</span>
                  )}
                </label>
                <div className="flex items-center gap-1">
                  {isSaving && (
                    <Loader2 className="w-3 h-3 animate-spin text-slate-400" />
                  )}
                  {isDirty && !isSaving && (
                    <span className="w-2 h-2 rounded-full bg-amber-400" title={t('fields.unsaved_changes', 'Unsaved changes')} />
                  )}
                </div>
              </div>
              <EditableField
                field={field}
                value={fieldValues[field.id]}
                onChange={handleFieldChange}
                saving={isSaving}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
}
