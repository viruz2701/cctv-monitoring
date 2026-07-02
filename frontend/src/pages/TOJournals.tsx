// ═══════════════════════════════════════════════════════════════════════
// TOJournals.tsx — TO Compliance Automation Journals Page
//
// Track 3: TO Compliance Automation
//   - UX-3.1: TO Journals with Regulatory Templates
//   - UX-3.4: AI Copilot for TO Journals (behind feature flag)
//
// Compliance:
//   - ISO 27001 A.12.4 (Audit trail)
//   - IEC 62443 SR 3.1 (Data integrity)
//   - OWASP ASVS V1.8 (Feature flags)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { isFeatureEnabled } from '../config/featureFlags';
import { JournalList } from '../components/to-journals/JournalList';
import { TemplateSelector } from '../components/to-journals/TemplateSelector';
import { AICopilot } from '../components/to-journals/AICopilot';
import { useTOJournals, useRegions } from '../hooks/useApiQuery/toJournals';
import { Card, Button, Tabs, Modal, EmptyState } from '../components/ui';
import {
  FileText,
  Plus,
  Download,
  Filter,
  Sparkles,
  Globe,
  RefreshCw,
} from '../components/ui/Icons';
import type { JournalFilter, GenerateJournalRequest } from '../services/toJournalsApi';

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const STATUS_LABELS: Record<string, { label: string; variant: 'success' | 'warning' | 'danger' | 'info' | 'neutral' | 'primary' | 'outline' }> = {
  draft: { label: 'Draft', variant: 'info' },
  generated: { label: 'Generated', variant: 'success' },
  signed: { label: 'Signed', variant: 'warning' },
  archived: { label: 'Archived', variant: 'neutral' },
};

// ═══════════════════════════════════════════════════════════════════════
// TOJournals Page
// ═══════════════════════════════════════════════════════════════════════

export function TOJournals() {
  const { t } = useTranslation();
  const aiEnabled = isFeatureEnabled('ai_copilot_to_journals');

  const [activeTab, setActiveTab] = useState('all');
  const [selectedRegion, setSelectedRegion] = useState<string>('');
  const [showTemplateModal, setShowTemplateModal] = useState(false);
  const [showAICopilot, setShowAICopilot] = useState(false);
  const [selectedJournalId, setSelectedJournalId] = useState<string | null>(null);

  // Filters
  const filters: JournalFilter = {
    ...(selectedRegion ? { region_code: selectedRegion } : {}),
    ...(activeTab !== 'all' ? { status: activeTab as JournalFilter['status'] } : {}),
  };

  const { data: journals, isLoading, error, refetch } = useTOJournals(filters);
  const { data: regions } = useRegions();

  // Callbacks
  const handleGenerate = useCallback((data: GenerateJournalRequest) => {
    setShowTemplateModal(false);
    // Генерация происходит через TemplateSelector
  }, []);

  const handleAIOpen = useCallback((journalId: string) => {
    setSelectedJournalId(journalId);
    setShowAICopilot(true);
  }, []);

  // Tabs configuration
  const tabs = [
    { id: 'all', label: t('to_journals.tabs.all', 'All') },
    { id: 'draft', label: t('to_journals.tabs.draft', 'Draft') },
    { id: 'generated', label: t('to_journals.tabs.generated', 'Generated') },
    { id: 'signed', label: t('to_journals.tabs.signed', 'Signed') },
    { id: 'archived', label: t('to_journals.tabs.archived', 'Archived') },
  ];

  return (
    <div className="space-y-6 p-6">
      {/* ── Header ────────────────────────────────────────────── */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
            {t('to_journals.title', 'TO Journals')}
          </h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            {t('to_journals.subtitle', 'Technical Operations compliance journals with regulatory templates')}
          </p>
        </div>
        <div className="flex items-center gap-3">
          {/* Region Filter */}
          <div className="relative">
            <Globe className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <select
              value={selectedRegion}
              onChange={(e) => setSelectedRegion(e.target.value)}
              className="pl-9 pr-4 py-2 text-sm rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 focus:ring-2 focus:ring-blue-500"
            >
              <option value="">{t('to_journals.all_regions', 'All Regions')}</option>
              {regions?.map((r) => (
                <option key={r.code} value={r.code}>
                  {r.flag} {r.name} ({r.code})
                </option>
              ))}
            </select>
          </div>

          {/* AI Copilot Button */}
          {aiEnabled && selectedJournalId && (
            <Button
              variant="outline"
              icon={<Sparkles className="w-4 h-4" />}
              onClick={() => handleAIOpen(selectedJournalId)}
            >
              {t('to_journals.ai_copilot', 'AI Copilot')}
            </Button>
          )}

          {/* Generate Button */}
          <Button
            variant="primary"
            icon={<Plus className="w-4 h-4" />}
            onClick={() => setShowTemplateModal(true)}
          >
            {t('to_journals.generate', 'Generate Journal')}
          </Button>

          {/* Refresh */}
          <Button
            variant="ghost"
            icon={<RefreshCw className="w-4 h-4" />}
            onClick={() => refetch()}
            aria-label={t('common.refresh', 'Refresh')}
          />
        </div>
      </div>

      {/* ── Filters & Tabs ─────────────────────────────────────── */}
      <Card className="p-4">
        <Tabs tabs={tabs} activeTab={activeTab} onChange={setActiveTab}>
          <div className="mt-4">
            {isLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 5 }).map((_, i) => (
                  <div
                    key={i}
                    className="h-16 bg-slate-100 dark:bg-slate-800 rounded-lg animate-pulse"
                  />
                ))}
              </div>
            ) : error ? (
              <EmptyState
                icon={<FileText className="w-12 h-12" />}
                title={t('to_journals.error_title', 'Failed to load journals')}
                description={error instanceof Error ? error.message : t('to_journals.error_desc', 'Please try again')}
                action={{ label: t('common.retry', 'Retry'), onClick: () => refetch() }}
              />
            ) : journals && journals.length > 0 ? (
              <JournalList
                journals={journals}
                onGenerate={handleAIOpen}
                statusConfig={STATUS_LABELS}
                regions={regions ?? []}
              />
            ) : (
              <EmptyState
                icon={<FileText className="w-12 h-12" />}
                title={t('to_journals.empty_title', 'No journals found')}
                description={t('to_journals.empty_desc', 'Generate your first TO journal using a regulatory template')}
                action={{ label: t('to_journals.generate', 'Generate Journal'), onClick: () => setShowTemplateModal(true) }}
              />
            )}
          </div>
        </Tabs>
      </Card>

      {/* ── Template Selector Modal ────────────────────────────── */}
      <Modal
        isOpen={showTemplateModal}
        onClose={() => setShowTemplateModal(false)}
        title={t('to_journals.select_template', 'Select Regulatory Template')}
        size="xl"
      >
        <TemplateSelector
          regions={regions ?? []}
          selectedRegion={selectedRegion}
          onRegionChange={setSelectedRegion}
          onGenerate={handleGenerate}
          onClose={() => setShowTemplateModal(false)}
        />
      </Modal>

      {/* ── AI Copilot Modal ───────────────────────────────────── */}
      {aiEnabled && (
        <Modal
          isOpen={showAICopilot}
          onClose={() => { setShowAICopilot(false); setSelectedJournalId(null); }}
          title={t('to_journals.ai_copilot_title', 'AI Journal Assistant')}
          size="lg"
        >
          {selectedJournalId && (
            <AICopilot
              journalId={selectedJournalId}
              onClose={() => { setShowAICopilot(false); setSelectedJournalId(null); }}
            />
          )}
        </Modal>
      )}
    </div>
  );
}
