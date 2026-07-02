// ═══════════════════════════════════════════════════════════════════════
// TemplateSelector.tsx — Select regulatory template and generate journal
//
// Track 3: TO Compliance Automation
//   - UX-3.1: TO Journals with Regulatory Templates
//
// Features:
//   - Multi-region support (10+ regulatory regions)
//   - Auto-fill applicable templates by region
//   - Preview before generation
//   - Date range picker for generation period
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback } from 'react';
import { useTOTemplates, usePreviewJournal, useGenerateJournal } from '../../hooks/useApiQuery/toJournals';
import { Button, Badge, Input } from '../ui';
import {
  FileText,
  Eye,
  Loader2,
  CheckCircle,
  AlertTriangle,
  ChevronRight,
  FileDown,
  Calendar,
} from '../ui/Icons';
import type { TOJournalTemplate, GenerateJournalRequest } from '../../services/toJournalsApi';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface TemplateSelectorProps {
  regions: { code: string; name: string; flag: string }[];
  selectedRegion: string;
  onRegionChange: (region: string) => void;
  onGenerate: (data: GenerateJournalRequest) => void;
  onClose: () => void;
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const REGULATORY_REGIONS = [
  { code: 'BY', name: 'Belarus', flag: '🇧🇾', standards: 'СТБ IEC 62443, СТБ 34.101.27/30, ОАЦ №66' },
  { code: 'RU', name: 'Russia', flag: '🇷🇺', standards: 'ГОСТ/ФСТЭК, 152-ФЗ, 149-ФЗ, КИИ' },
  { code: 'KZ', name: 'Kazakhstan', flag: '🇰🇿', standards: 'Закон РБ о КИИ, ISO 27001' },
  { code: 'EU', name: 'European Union', flag: '🇪🇺', standards: 'GDPR Art.35, EN 62676, NIS2' },
  { code: 'TR', name: 'Turkey', flag: '🇹🇷', standards: 'KVKK №6698, TS EN 62676' },
  { code: 'VN', name: 'Vietnam', flag: '🇻🇳', standards: 'TCVN 11930:2017, Decree 13/2023' },
  { code: 'ID', name: 'Indonesia', flag: '🇮🇩', standards: 'UU PDP, SNI 27001' },
  { code: 'ZA', name: 'South Africa', flag: '🇿🇦', standards: 'POPIA, SANS 10160-4' },
  { code: 'BR', name: 'Brazil', flag: '🇧🇷', standards: 'LGPD' },
  { code: 'MX', name: 'Mexico', flag: '🇲🇽', standards: 'LFPDPPP' },
  { code: 'US', name: 'United States', flag: '🇺🇸', standards: 'NIST SP 800-82, IEC 62443' },
  { code: 'INTL', name: 'International', flag: '🌐', standards: 'ISO 27001:2022, IEC 62443-3-3' },
];

// ═══════════════════════════════════════════════════════════════════════
// TemplateSelector Component
// ═══════════════════════════════════════════════════════════════════════

export function TemplateSelector({
  regions,
  selectedRegion,
  onRegionChange,
  onGenerate,
  onClose,
}: TemplateSelectorProps) {
  // Local state
  const [selectedTemplate, setSelectedTemplate] = useState<string>('');
  const [workOrderId, setWorkOrderId] = useState<string>('');
  const [periodStart, setPeriodStart] = useState<string>(
    new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
  );
  const [periodEnd, setPeriodEnd] = useState<string>(
    new Date().toISOString().split('T')[0],
  );
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);

  // Hooks
  const { data: templates, isLoading: templatesLoading } = useTOTemplates(selectedRegion);
  const previewMutation = usePreviewJournal();
  const generateMutation = useGenerateJournal();

  // Callbacks
  const handlePreview = useCallback(async () => {
    if (!selectedTemplate || !periodStart || !periodEnd) return;
    try {
      const result = await previewMutation.mutateAsync({
        template_id: selectedTemplate,
        work_order_id: workOrderId || 'preview',
        period_start: periodStart,
        period_end: periodEnd,
        region_code: selectedRegion,
      });
      setPreviewUrl(result.preview_url);
    } catch {
      // Error handled by mutation
    }
  }, [selectedTemplate, periodStart, periodEnd, workOrderId, selectedRegion, previewMutation]);

  const handleGenerate = useCallback(async () => {
    if (!selectedTemplate || !periodStart || !periodEnd) return;
    try {
      await generateMutation.mutateAsync({
        template_id: selectedTemplate,
        work_order_id: workOrderId,
        period_start: periodStart,
        period_end: periodEnd,
        region_code: selectedRegion,
      });
      onClose();
    } catch {
      // Error handled by mutation
    }
  }, [selectedTemplate, periodStart, periodEnd, workOrderId, selectedRegion, generateMutation, onClose]);

  const handleRegionSelect = useCallback((code: string) => {
    onRegionChange(code);
    setSelectedTemplate('');
    setPreviewUrl(null);
  }, [onRegionChange]);

  // Derive active standards info
  const activeRegionInfo = REGULATORY_REGIONS.find((r) => r.code === selectedRegion);

  return (
    <div className="space-y-6">
      {/* ── Step 1: Select Region ──────────────────────────────── */}
      <div>
        <h3 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">
          Step 1: Select Regulatory Region
        </h3>
        <div className="grid grid-cols-3 sm:grid-cols-4 gap-2">
          {REGULATORY_REGIONS.map((region) => {
            const isSelected = selectedRegion === region.code;
            return (
              <button
                key={region.code}
                onClick={() => handleRegionSelect(region.code)}
                className={`
                  flex flex-col items-center gap-1 p-3 rounded-lg border text-sm transition-all
                  ${isSelected
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 dark:border-blue-400'
                    : 'border-slate-200 dark:border-slate-700 hover:border-slate-300 dark:hover:border-slate-600'
                  }
                `}
              >
                <span className="text-xl">{region.flag}</span>
                <span className="font-medium text-slate-700 dark:text-slate-300">
                  {region.code}
                </span>
                <span className="text-xs text-slate-400 dark:text-slate-500">
                  {region.name}
                </span>
              </button>
            );
          })}
        </div>

        {/* Active standards info */}
        {activeRegionInfo && (
          <div className="mt-3 p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
            <p className="text-xs text-slate-500 dark:text-slate-400">
              <span className="font-medium">Applicable standards: </span>
              {activeRegionInfo.standards}
            </p>
          </div>
        )}
      </div>

      {/* ── Step 2: Select Template ────────────────────────────── */}
      {selectedRegion && (
        <div>
          <h3 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">
            Step 2: Select Template
          </h3>
          {templatesLoading ? (
            <div className="flex items-center gap-2 text-sm text-slate-500">
              <Loader2 className="w-4 h-4 animate-spin" />
              Loading templates...
            </div>
          ) : templates && templates.length > 0 ? (
            <div className="space-y-2 max-h-48 overflow-y-auto">
              {templates.map((tmpl) => {
                const isSelected = selectedTemplate === tmpl.id;
                return (
                  <button
                    key={tmpl.id}
                    onClick={() => setSelectedTemplate(tmpl.id)}
                    className={`
                      w-full text-left p-3 rounded-lg border transition-all
                      ${isSelected
                        ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 dark:border-blue-400'
                        : 'border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800/50'
                      }
                    `}
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <span className="font-medium text-slate-800 dark:text-slate-200 text-sm">
                          {tmpl.name}
                        </span>
                        <span className="ml-2 text-xs text-slate-400">
                          v{tmpl.version}
                        </span>
                      </div>
                      <ChevronRight className={`w-4 h-4 transition-colors ${
                        isSelected ? 'text-blue-500' : 'text-slate-300'
                      }`} />
                    </div>
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                      {tmpl.description}
                    </p>
                    <div className="flex items-center gap-2 mt-1.5">
                      <Badge variant="neutral" size="sm">
                        {tmpl.regulatory_ref}
                      </Badge>
                      <span className="text-xs text-slate-400">
                        {tmpl.sections.length} sections
                      </span>
                    </div>
                  </button>
                );
              })}
            </div>
          ) : (
            <div className="flex items-center gap-2 p-4 text-sm text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20 rounded-lg">
              <AlertTriangle className="w-4 h-4" />
              No templates available for this region
            </div>
          )}
        </div>
      )}

      {/* ── Step 3: Configure Period ───────────────────────────── */}
      {selectedTemplate && (
        <div>
          <h3 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">
            Step 3: Configure Period
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                Period Start
              </label>
              <Input
                type="date"
                value={periodStart}
                onChange={(e) => setPeriodStart(e.target.value)}
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                Period End
              </label>
              <Input
                type="date"
                value={periodEnd}
                onChange={(e) => setPeriodEnd(e.target.value)}
              />
            </div>
          </div>
          <div className="mt-3">
            <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
              Work Order ID (optional)
            </label>
            <Input
              type="text"
              placeholder="WO-2025-XXXX"
              value={workOrderId}
              onChange={(e) => setWorkOrderId(e.target.value)}
            />
          </div>
        </div>
      )}

      {/* ── Preview ────────────────────────────────────────────── */}
      {previewUrl && (
        <div className="border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden">
          <div className="p-3 bg-slate-50 dark:bg-slate-800 flex items-center justify-between">
            <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
              <Eye className="w-4 h-4 inline mr-1" />
              Preview
            </span>
            <Button
              variant="outline"
              size="sm"
              icon={<FileDown className="w-3.5 h-3.5" />}
              onClick={() => window.open(previewUrl, '_blank')}
            >
              Open PDF
            </Button>
          </div>
          <iframe
            src={previewUrl}
            className="w-full h-64 bg-white"
            title="Journal Preview"
          />
        </div>
      )}

      {/* ── Actions ────────────────────────────────────────────── */}
      <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
        <Button variant="outline" onClick={onClose}>
          Cancel
        </Button>
        {selectedTemplate && (
          <Button
            variant="outline"
            icon={<Eye className="w-4 h-4" />}
            onClick={handlePreview}
            loading={previewMutation.isPending}
            disabled={!selectedTemplate || !periodStart || !periodEnd}
          >
            Preview
          </Button>
        )}
        <Button
          variant="primary"
          icon={<FileText className="w-4 h-4" />}
          onClick={handleGenerate}
          loading={generateMutation.isPending}
          disabled={!selectedTemplate || !periodStart || !periodEnd}
        >
          Generate Journal
        </Button>
      </div>
    </div>
  );
}
