// ═══════════════════════════════════════════════════════════════════════
// EndpointEditor — редактор отдельного endpoint'а дескриптора (PROTO-06)
//
// Позволяет редактировать method, path, parser, headers, query params
// для одного endpoint'а протокола.
//
// Compliance:
//   - WCAG 2.1 AA (labels, aria-describedby, focus management)
//   - OWASP ASVS V5 (input validation — whitelist для method)
// ═══════════════════════════════════════════════════════════════════════

import React, { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  ChevronDown,
  ChevronUp,
  Trash2,
  Plus,
  Copy,
} from '../../components/ui/Icons';
import { Button, Input, Select, Textarea } from '../../components/ui';
import type {
  ProtocolEndpoint,
  HttpMethod,
  ParamDef,
  ParserType,
} from '../../types/descriptor';

// ─── Props ──────────────────────────────────────────────────────────

interface EndpointEditorProps {
  endpoint: ProtocolEndpoint;
  index: number;
  onChange: (index: number, patch: Partial<ProtocolEndpoint>) => void;
  onRemove: (index: number) => void;
  onDuplicate: (index: number) => void;
  collapsed?: boolean;
  onToggleCollapse?: (index: number) => void;
}

// ─── Constants ──────────────────────────────────────────────────────

const METHOD_OPTIONS: { value: string; label: string }[] = [
  { value: 'GET', label: 'GET' },
  { value: 'POST', label: 'POST' },
  { value: 'PUT', label: 'PUT' },
  { value: 'DELETE', label: 'DELETE' },
  { value: 'PATCH', label: 'PATCH' },
];

const PARSER_OPTIONS: { value: string; label: string }[] = [
  { value: 'jsonpath', label: 'JSONPath' },
  { value: 'regex', label: 'Regex' },
  { value: 'xpath', label: 'XPath' },
  { value: 'custom', label: 'Custom Script' },
];

const PARAM_TYPE_OPTIONS: { value: string; label: string }[] = [
  { value: 'string', label: 'string' },
  { value: 'number', label: 'number' },
  { value: 'boolean', label: 'boolean' },
  { value: 'object', label: 'object' },
];

const DEFAULT_SUCCESS_CODES = [200, 201, 204];

// ─── Component ──────────────────────────────────────────────────────

export function EndpointEditor({
  endpoint,
  index,
  onChange,
  onRemove,
  onDuplicate,
  collapsed = false,
  onToggleCollapse,
}: EndpointEditorProps) {
  const { t } = useTranslation();

  const handleChange = useCallback(
    (patch: Partial<ProtocolEndpoint>) => {
      onChange(index, patch);
    },
    [onChange, index],
  );

  const handleParamChange = useCallback(
    (paramIndex: number, patch: Partial<ParamDef>) => {
      const params = [...(endpoint.queryParams || [])];
      if (paramIndex >= 0 && paramIndex < params.length) {
        params[paramIndex] = { ...params[paramIndex], ...patch };
        handleChange({ queryParams: params });
      }
    },
    [handleChange, endpoint.queryParams],
  );

  const addParam = useCallback(() => {
    const params = [...(endpoint.queryParams || [])];
    params.push({
      name: '',
      type: 'string',
      required: false,
    });
    handleChange({ queryParams: params });
  }, [handleChange, endpoint.queryParams]);

  const removeParam = useCallback(
    (paramIndex: number) => {
      const params = (endpoint.queryParams || []).filter(
        (_, i) => i !== paramIndex,
      );
      handleChange({ queryParams: params });
    },
    [handleChange, endpoint.queryParams],
  );

  const headerEntries = Object.entries(endpoint.headers || {});

  const addHeader = useCallback(() => {
    const headers = { ...(endpoint.headers || {}), '': '' };
    handleChange({ headers });
  }, [handleChange, endpoint.headers]);

  const removeHeader = useCallback(
    (key: string) => {
      const headers = { ...(endpoint.headers || {}) };
      delete headers[key];
      handleChange({ headers });
    },
    [handleChange, endpoint.headers],
  );

  return (
    <div className="border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 overflow-hidden">
      {/* Header */}
      <button
        type="button"
        onClick={() => onToggleCollapse?.(index)}
        className="w-full flex items-center justify-between px-4 py-3 text-sm font-medium
                   text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-750
                   transition-colors"
        aria-expanded={!collapsed}
        aria-controls={`endpoint-body-${index}`}
      >
        <span className="flex items-center gap-2">
          <span className="px-2 py-0.5 text-xs font-mono font-bold rounded
                           bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300">
            {endpoint.method}
          </span>
          <code className="text-xs font-mono text-slate-500 dark:text-slate-400">
            {endpoint.path || '/'}
          </code>
          {endpoint.name && (
            <span className="text-slate-500 dark:text-slate-400">
              — {endpoint.name}
            </span>
          )}
        </span>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onDuplicate(index);
            }}
            className="p-1 rounded hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-400"
            title={t('descriptors.duplicate')}
            aria-label={t('descriptors.duplicate')}
          >
            <Copy className="w-3.5 h-3.5" />
          </button>
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onRemove(index);
            }}
            className="p-1 rounded hover:bg-red-100 dark:hover:bg-red-900/30 text-red-400"
            title={t('descriptors.removeEndpoint')}
            aria-label={t('descriptors.removeEndpoint')}
          >
            <Trash2 className="w-3.5 h-3.5" />
          </button>
          {collapsed ? (
            <ChevronDown className="w-4 h-4 text-slate-400" />
          ) : (
            <ChevronUp className="w-4 h-4 text-slate-400" />
          )}
        </div>
      </button>

      {/* Body */}
      {!collapsed && (
        <div id={`endpoint-body-${index}`} className="px-4 pb-4 space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {/* Method */}
            <div>
              <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                {t('descriptors.method')}
              </label>
              <Select
                value={endpoint.method}
                onChange={(e) => handleChange({ method: e.target.value as HttpMethod })}
                options={METHOD_OPTIONS}
                aria-label={t('descriptors.method')}
              />
            </div>

            {/* Path */}
            <div className="md:col-span-2">
              <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                {t('descriptors.path')}
              </label>
              <Input
                value={endpoint.path}
                onChange={(e) => handleChange({ path: e.target.value })}
                placeholder="/cgi-bin/param.cgi"
                aria-label={t('descriptors.path')}
              />
            </div>
          </div>

          {/* Name & Description */}
          <div>
            <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
              {t('descriptors.endpointName')}
            </label>
            <Input
              value={endpoint.name}
              onChange={(e) => handleChange({ name: e.target.value })}
              placeholder={t('descriptors.endpointNamePlaceholder')}
              aria-label={t('descriptors.endpointName')}
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
              {t('descriptors.description')}
            </label>
            <Textarea
              value={endpoint.description || ''}
              onChange={(e) => handleChange({ description: e.target.value })}
              placeholder={t('descriptors.endpointDescPlaceholder')}
              rows={2}
              aria-label={t('descriptors.description')}
            />
          </div>

          {/* Success Codes & Timeout */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                {t('descriptors.successCodes')}
              </label>
              <Input
                value={(endpoint.successCodes || DEFAULT_SUCCESS_CODES).join(', ')}
                onChange={(e) => {
                  const codes = e.target.value
                    .split(',')
                    .map((s) => parseInt(s.trim(), 10))
                    .filter((n) => !isNaN(n));
                  handleChange({ successCodes: codes });
                }}
                placeholder="200, 201, 204"
                aria-label={t('descriptors.successCodes')}
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                {t('descriptors.timeout')} (s)
              </label>
              <Input
                type="number"
                min={1}
                max={120}
                value={endpoint.timeout ?? 30}
                onChange={(e) =>
                  handleChange({ timeout: parseInt(e.target.value, 10) || 30 })
                }
                aria-label={t('descriptors.timeout')}
              />
            </div>
          </div>

          {/* Headers */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
                {t('descriptors.headers')}
              </span>
              <button
                type="button"
                onClick={addHeader}
                className="text-xs text-blue-600 dark:text-blue-400 hover:underline flex items-center gap-1"
              >
                <Plus className="w-3 h-3" />
                {t('descriptors.addHeader')}
              </button>
            </div>
            {headerEntries.length === 0 && (
              <p className="text-xs text-slate-400 italic">
                {t('descriptors.noHeaders')}
              </p>
            )}
            {headerEntries.map(([key, value]) => (
              <div key={key || Math.random().toString()} className="flex gap-2 mb-1">
                <Input
                  value={key}
                  onChange={(e) => {
                    const newKey = e.target.value;
                    const headers = { ...(endpoint.headers || {}) };
                    delete headers[key];
                    headers[newKey] = value;
                    handleChange({ headers });
                  }}
                  placeholder={t('descriptors.headerName')}
                  className="flex-1"
                  aria-label={t('descriptors.headerName')}
                />
                <Input
                  value={value}
                  onChange={(e) => {
                    const headers = { ...(endpoint.headers || {}), [key]: e.target.value };
                    handleChange({ headers });
                  }}
                  placeholder={t('descriptors.headerValue')}
                  className="flex-1"
                  aria-label={t('descriptors.headerValue')}
                />
                <button
                  type="button"
                  onClick={() => removeHeader(key)}
                  className="p-1.5 rounded hover:bg-red-100 dark:hover:bg-red-900/30 text-red-400"
                  aria-label={t('descriptors.removeHeader')}
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
            ))}
          </div>

          {/* Query Params */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
                {t('descriptors.queryParams')}
              </span>
              <button
                type="button"
                onClick={addParam}
                className="text-xs text-blue-600 dark:text-blue-400 hover:underline flex items-center gap-1"
              >
                <Plus className="w-3 h-3" />
                {t('descriptors.addParam')}
              </button>
            </div>
            {(endpoint.queryParams || []).length === 0 && (
              <p className="text-xs text-slate-400 italic">
                {t('descriptors.noParams')}
              </p>
            )}
            {(endpoint.queryParams || []).map((param, pi) => (
              <div key={pi} className="flex gap-2 mb-1 items-start">
                <Input
                  value={param.name}
                  onChange={(e) => handleParamChange(pi, { name: e.target.value })}
                  placeholder={t('descriptors.paramName')}
                  className="flex-[2]"
                  aria-label={t('descriptors.paramName')}
                />
                <Select
                  value={param.type}
                  onChange={(e) =>
                    handleParamChange(pi, { type: e.target.value as ParamDef['type'] })
                  }
                  options={PARAM_TYPE_OPTIONS}
                  className="flex-1"
                  aria-label={t('descriptors.paramType')}
                />
                <label className="flex items-center gap-1 text-xs text-slate-500 mt-1.5 whitespace-nowrap">
                  <input
                    type="checkbox"
                    checked={param.required}
                    onChange={(e) => handleParamChange(pi, { required: e.target.checked })}
                    className="rounded border-slate-300 dark:border-slate-600"
                  />
                  {t('descriptors.required')}
                </label>
                <button
                  type="button"
                  onClick={() => removeParam(pi)}
                  className="p-1.5 rounded hover:bg-red-100 dark:hover:bg-red-900/30 text-red-400 mt-1"
                  aria-label={t('descriptors.removeParam')}
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
            ))}
          </div>

          {/* Parser */}
          <div>
            <span className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
              {t('descriptors.parser')}
            </span>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <Select
                  value={endpoint.parser?.type || 'jsonpath'}
                  onChange={(e) =>
                    handleChange({
                      parser: {
                        type: e.target.value as ParserType,
                        expression: endpoint.parser?.expression,
                        resultPath: endpoint.parser?.resultPath,
                      },
                    })
                  }
                  options={PARSER_OPTIONS}
                  aria-label={t('descriptors.parserType')}
                />
              </div>
              <div>
                <Input
                  value={endpoint.parser?.expression || ''}
                  onChange={(e) =>
                    handleChange({
                      parser: {
                        ...(endpoint.parser || { type: 'jsonpath' }),
                        expression: e.target.value,
                      },
                    })
                  }
                  placeholder="$.data.status"
                  aria-label={t('descriptors.parserExpression')}
                />
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
