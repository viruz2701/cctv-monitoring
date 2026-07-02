// ═══════════════════════════════════════════════════════════════════════
// PropertiesPanel.tsx — Панель свойств для выбранного блока
//
// UX-3.5: Visual drag-n-drop редактор печатных форм
//   - Properties panel для каждого типа блока
//   - Редактирование в реальном времени
// ═══════════════════════════════════════════════════════════════════════

import React, { useCallback } from 'react';
import type {
  CanvasBlock,
  TextBlockProps,
  FieldBlockProps,
  TableBlockProps,
  ImageBlockProps,
  SignatureBlockProps,
  QRBlockProps,
} from './types';

// ── Color presets ─────────────────────────────────────────────────────

const COLOR_PRESETS = [
  '#1e293b', '#475569', '#64748b', '#94a3b8',
  '#dc2626', '#ea580c', '#d97706', '#65a30d',
  '#059669', '#0284c7', '#2563eb', '#7c3aed',
];

// ── Field presets ─────────────────────────────────────────────────────

const FIELD_PRESETS = [
  { value: 'device_name', label: 'Device Name' },
  { value: 'device_id', label: 'Device ID' },
  { value: 'technician_name', label: 'Technician Name' },
  { value: 'technician_id', label: 'Technician ID' },
  { value: 'site_name', label: 'Site Name' },
  { value: 'completed_at', label: 'Completed At' },
  { value: 'started_at', label: 'Started At' },
  { value: 'duration_minutes', label: 'Duration (min)' },
  { value: 'work_order_id', label: 'Work Order ID' },
  { value: 'schedule_id', label: 'Schedule ID' },
  { value: 'notes', label: 'Notes' },
  { value: 'checklist_notes', label: 'Checklist Notes' },
  { value: 'defects', label: 'Defects' },
];

// ── Sub-components ────────────────────────────────────────────────────

function InputField({
  label,
  value,
  onChange,
  type = 'text',
  min,
  max,
  step,
  placeholder,
}: {
  label: string;
  value: string | number;
  onChange: (val: string | number) => void;
  type?: string;
  min?: number;
  max?: number;
  step?: number;
  placeholder?: string;
}) {
  return (
    <div className="space-y-1">
      <label className="block text-xs font-medium text-slate-600 dark:text-slate-400">
        {label}
      </label>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(type === 'number' ? Number(e.target.value) : e.target.value)}
        min={min}
        max={max}
        step={step}
        placeholder={placeholder}
        className="w-full px-2.5 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
      />
    </div>
  );
}

function ColorPicker({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (val: string) => void;
}) {
  return (
    <div className="space-y-1">
      <label className="block text-xs font-medium text-slate-600 dark:text-slate-400">
        {label}
      </label>
      <div className="flex items-center gap-2">
        <input
          type="color"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-8 h-8 rounded cursor-pointer border border-slate-300"
          aria-label={`${label} color picker`}
        />
        <span className="text-xs text-slate-500 font-mono">{value}</span>
      </div>
      <div className="flex flex-wrap gap-1 mt-1">
        {COLOR_PRESETS.map((color) => (
          <button
            key={color}
            onClick={() => onChange(color)}
            className={`w-5 h-5 rounded-full border-2 ${
              value === color ? 'border-blue-500' : 'border-transparent'
            }`}
            style={{ backgroundColor: color }}
            aria-label={`Set color ${color}`}
          />
        ))}
      </div>
    </div>
  );
}

function SelectField({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: { value: string; label: string }[];
  onChange: (val: string) => void;
}) {
  return (
    <div className="space-y-1">
      <label className="block text-xs font-medium text-slate-600 dark:text-slate-400">
        {label}
      </label>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-2.5 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
    </div>
  );
}

// ── Block property editors ────────────────────────────────────────────

function TextProperties({
  props,
  onChange,
}: {
  props: TextBlockProps;
  onChange: (props: Record<string, unknown>) => void;
}) {
  const set = (key: string, val: unknown) => onChange({ ...props, [key]: val });

  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <label className="block text-xs font-medium text-slate-600 dark:text-slate-400">
          Content
        </label>
        <textarea
          value={props.content || ''}
          onChange={(e) => set('content', e.target.value)}
          rows={3}
          className="w-full px-2.5 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono"
          placeholder="Use {{variable}} syntax for dynamic data"
        />
        <p className="text-[10px] text-slate-400">
          Use {'{{variable_name}}'} for dynamic fields
        </p>
      </div>
      <InputField label="Font Size" value={props.fontSize || 14} onChange={(v) => set('fontSize', v)} type="number" min={8} max={72} />
      <SelectField
        label="Font Weight"
        value={props.fontWeight || 'normal'}
        options={[
          { value: 'light', label: 'Light' },
          { value: 'normal', label: 'Normal' },
          { value: 'bold', label: 'Bold' },
        ]}
        onChange={(v) => set('fontWeight', v)}
      />
      <SelectField
        label="Alignment"
        value={props.alignment || 'left'}
        options={[
          { value: 'left', label: 'Left' },
          { value: 'center', label: 'Center' },
          { value: 'right', label: 'Right' },
        ]}
        onChange={(v) => set('alignment', v)}
      />
      <ColorPicker label="Text Color" value={props.color || '#1e293b'} onChange={(v) => set('color', v)} />
    </div>
  );
}

function FieldProperties({
  props,
  onChange,
}: {
  props: FieldBlockProps;
  onChange: (props: Record<string, unknown>) => void;
}) {
  const set = (key: string, val: unknown) => onChange({ ...props, [key]: val });

  return (
    <div className="space-y-3">
      <SelectField
        label="Data Field"
        value={props.field || ''}
        options={FIELD_PRESETS}
        onChange={(v) => set('field', v)}
      />
      <InputField label="Label" value={props.label || ''} onChange={(v) => set('label', v)} placeholder="Field label" />
      <InputField label="Font Size" value={props.fontSize || 11} onChange={(v) => set('fontSize', v)} type="number" min={8} max={24} />
      <InputField label="Format" value={props.format || ''} onChange={(v) => set('format', v)} placeholder="e.g. dd.MM.yyyy" />
      <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400">
        <input
          type="checkbox"
          checked={props.showLabel !== false}
          onChange={(e) => set('showLabel', e.target.checked)}
          className="rounded border-slate-300"
        />
        Show Label
      </label>
    </div>
  );
}

function TableProperties({
  props,
  onChange,
}: {
  props: TableBlockProps;
  onChange: (props: Record<string, unknown>) => void;
}) {
  const set = (key: string, val: unknown) => onChange({ ...props, [key]: val });

  return (
    <div className="space-y-3">
      <SelectField
        label="Data Source"
        value={props.dataSource || 'checklist'}
        options={[
          { value: 'checklist', label: 'Checklist' },
          { value: 'parts_used', label: 'Parts Used' },
          { value: 'labor', label: 'Labor' },
        ]}
        onChange={(v) => set('dataSource', v)}
      />
      <InputField label="Font Size" value={props.fontSize || 10} onChange={(v) => set('fontSize', v)} type="number" min={6} max={18} />
      <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400">
        <input
          type="checkbox"
          checked={props.showHeader !== false}
          onChange={(e) => set('showHeader', e.target.checked)}
          className="rounded border-slate-300"
        />
        Show Header
      </label>
      <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400">
        <input
          type="checkbox"
          checked={props.bordered !== false}
          onChange={(e) => set('bordered', e.target.checked)}
          className="rounded border-slate-300"
        />
        Bordered
      </label>
      {props.columns && (
        <div className="space-y-1">
          <span className="text-xs font-medium text-slate-600 dark:text-slate-400">
            Columns ({props.columns.length})
          </span>
          <div className="text-xs text-slate-500 space-y-0.5">
            {props.columns.map((col: { key: string; label: string }, i: number) => (
              <div key={i} className="flex items-center gap-1">
                <span className="text-slate-400">{i + 1}.</span>
                <span className="font-medium">{col.label}</span>
                <span className="text-slate-400">({col.key})</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function ImageProperties({
  props,
  onChange,
}: {
  props: ImageBlockProps;
  onChange: (props: Record<string, unknown>) => void;
}) {
  const set = (key: string, val: unknown) => onChange({ ...props, [key]: val });

  return (
    <div className="space-y-3">
      <InputField label="Image URL" value={props.src || ''} onChange={(v) => set('src', v)} placeholder="https://..." />
      <InputField label="Alt Text" value={props.alt || ''} onChange={(v) => set('alt', v)} placeholder="Description" />
      <div className="grid grid-cols-2 gap-2">
        <InputField label="Width" value={props.width || 200} onChange={(v) => set('width', v)} type="number" min={20} max={595} />
        <InputField label="Height" value={props.height || 150} onChange={(v) => set('height', v)} type="number" min={20} max={842} />
      </div>
      <SelectField
        label="Fit"
        value={props.fit || 'contain'}
        options={[
          { value: 'contain', label: 'Contain' },
          { value: 'cover', label: 'Cover' },
          { value: 'fill', label: 'Fill' },
        ]}
        onChange={(v) => set('fit', v)}
      />
      {props.dataSource && (
        <>
          <SelectField
            label="Data Source"
            value={props.dataSource || ''}
            options={[
              { value: 'photos', label: 'Photos' },
              { value: '', label: 'Static Image' },
            ]}
            onChange={(v) => set('dataSource', v)}
          />
          {props.dataSource === 'photos' && (
            <>
              <InputField label="Columns" value={props.columns || 2} onChange={(v) => set('columns', v)} type="number" min={1} max={4} />
              <InputField label="Max Photos" value={props.maxPhotos || 4} onChange={(v) => set('maxPhotos', v)} type="number" min={1} max={20} />
            </>
          )}
        </>
      )}
    </div>
  );
}

function SignatureProperties({
  props,
  onChange,
}: {
  props: SignatureBlockProps;
  onChange: (props: Record<string, unknown>) => void;
}) {
  const set = (key: string, val: unknown) => onChange({ ...props, [key]: val });

  return (
    <div className="space-y-3">
      <InputField label="Label" value={props.label || ''} onChange={(v) => set('label', v)} placeholder="Signature label" />
      <InputField label="Line Width" value={props.lineWidth || 200} onChange={(v) => set('lineWidth', v)} type="number" min={80} max={595} />
      <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400">
        <input
          type="checkbox"
          checked={props.showDate !== false}
          onChange={(e) => set('showDate', e.target.checked)}
          className="rounded border-slate-300"
        />
        Show Date
      </label>
      {props.showHash && (
        <>
          <SelectField
            label="Algorithm"
            value={props.algorithm || 'bash-256'}
            options={[
              { value: 'bash-256', label: 'СТБ bash-256' },
              { value: 'bash-512', label: 'СТБ bash-512' },
            ]}
            onChange={(v) => set('algorithm', v)}
          />
          <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400">
            <input
              type="checkbox"
              checked={props.showHash !== false}
              onChange={(e) => set('showHash', e.target.checked)}
              className="rounded border-slate-300"
            />
            Show Hash Value
          </label>
        </>
      )}
    </div>
  );
}

function QRProperties({
  props,
  onChange,
}: {
  props: QRBlockProps;
  onChange: (props: Record<string, unknown>) => void;
}) {
  const set = (key: string, val: unknown) => onChange({ ...props, [key]: val });

  return (
    <div className="space-y-3">
      <SelectField
        label="Data Source"
        value={props.dataSource || 'work_order_id'}
        options={[
          { value: 'work_order_id', label: 'Work Order ID' },
          { value: 'verification_url', label: 'Verification URL' },
          { value: 'custom', label: 'Custom Text' },
        ]}
        onChange={(v) => set('dataSource', v)}
      />
      <InputField label="QR Size" value={props.size || 80} onChange={(v) => set('size', v)} type="number" min={40} max={300} />
      <InputField label="Label" value={props.label || ''} onChange={(v) => set('label', v)} placeholder="QR label text" />
      <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400">
        <input
          type="checkbox"
          checked={props.includeHash !== false}
          onChange={(e) => set('includeHash', e.target.checked)}
          className="rounded border-slate-300"
        />
        Include Hash
      </label>
    </div>
  );
}

// ── Main PropertiesPanel ──────────────────────────────────────────────

interface PropertiesPanelProps {
  selectedBlock: CanvasBlock | null;
  onUpdateProps: (instanceId: string, props: Record<string, unknown>) => void;
  blocksCount: number;
}

export function PropertiesPanel({
  selectedBlock,
  onUpdateProps,
  blocksCount,
}: PropertiesPanelProps) {
  if (!selectedBlock) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6 text-center">
        <div className="w-10 h-10 rounded-full bg-slate-100 dark:bg-slate-800 flex items-center justify-center mb-3">
          <span className="text-lg text-slate-400">⚙</span>
        </div>
        <p className="text-sm text-slate-500 dark:text-slate-400">
          Select a block to edit its properties
        </p>
        <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">
          {blocksCount} block{blocksCount !== 1 ? 's' : ''} on canvas
        </p>
      </div>
    );
  }

  const renderProperties = () => {
    switch (selectedBlock.type) {
      case 'text':
        return (
          <TextProperties
            props={selectedBlock.props as unknown as TextBlockProps}
            onChange={(p) => onUpdateProps(selectedBlock.instanceId, p)}
          />
        );
      case 'field':
        return (
          <FieldProperties
            props={selectedBlock.props as unknown as FieldBlockProps}
            onChange={(p) => onUpdateProps(selectedBlock.instanceId, p)}
          />
        );
      case 'table':
        return (
          <TableProperties
            props={selectedBlock.props as unknown as TableBlockProps}
            onChange={(p) => onUpdateProps(selectedBlock.instanceId, p)}
          />
        );
      case 'image':
        return (
          <ImageProperties
            props={selectedBlock.props as unknown as ImageBlockProps}
            onChange={(p) => onUpdateProps(selectedBlock.instanceId, p)}
          />
        );
      case 'signature':
        return (
          <SignatureProperties
            props={selectedBlock.props as unknown as SignatureBlockProps}
            onChange={(p) => onUpdateProps(selectedBlock.instanceId, p)}
          />
        );
      case 'qr':
        return (
          <QRProperties
            props={selectedBlock.props as unknown as QRBlockProps}
            onChange={(p) => onUpdateProps(selectedBlock.instanceId, p)}
          />
        );
      default:
        return (
          <p className="text-sm text-slate-500">
            No properties available for this block type
          </p>
        );
    }
  };

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="px-4 py-3 border-b border-slate-200 dark:border-slate-700">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100 truncate">
          {selectedBlock.label}
        </h3>
        <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
          {selectedBlock.type} · {selectedBlock.size.width}×{selectedBlock.size.height}
        </p>
      </div>

      {/* Properties */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        <div className="grid grid-cols-2 gap-2">
          <InputField
            label="Width"
            value={selectedBlock.size.width}
            onChange={(v) => {
              const w = Number(v);
              if (w >= 20) {
                onUpdateProps(selectedBlock.instanceId, {
                  ...selectedBlock.props,
                  _resize: { width: w, height: selectedBlock.size.height },
                });
              }
            }}
            type="number"
            min={20}
            max={595}
          />
          <InputField
            label="Height"
            value={selectedBlock.size.height}
            onChange={(v) => {
              const h = Number(v);
              if (h >= 20) {
                onUpdateProps(selectedBlock.instanceId, {
                  ...selectedBlock.props,
                  _resize: { width: selectedBlock.size.width, height: h },
                });
              }
            }}
            type="number"
            min={20}
            max={842}
          />
        </div>
        <div className="border-t border-slate-200 dark:border-slate-700 pt-3" />
        {renderProperties()}
      </div>

      {/* Block Info */}
      <div className="px-4 py-2 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
        <div className="flex items-center justify-between text-[10px] text-slate-400">
          <span>ID: {selectedBlock.instanceId.slice(0, 8)}</span>
          <span>Pos: {selectedBlock.position.x},{selectedBlock.position.y}</span>
        </div>
      </div>
    </div>
  );
}

export default PropertiesPanel;
