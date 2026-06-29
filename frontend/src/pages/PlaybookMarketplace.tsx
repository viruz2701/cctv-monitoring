// ═══════════════════════════════════════════════════════════════════════
// PlaybookMarketplace — Публичный marketplace pre-built playbooks
//
// P1-MARKET: Card grid, filter by vendor/rating/verified,
// search by name/description, one-click install, rating modal.
//
// Features:
//   - Card grid of available playbooks
//   - Filter by vendor, rating, verified badge
//   - Search by name/description
//   - One-click install button
//   - Rating modal (1-5 stars + review)
//   - Version compatibility matrix
//   - Private sharing between tenants
// ═══════════════════════════════════════════════════════════════════════

import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Search,
  Star,
  Filter,
  Download,
  Store,
  ChevronDown,
  ChevronUp,
  MessageCircle,
  ShieldCheck,
  ExternalLink,
  Loader2,
  X,
} from '../components/ui/Icons';
import { Card, CardContent } from '../components/ui/Card';
import { Badge } from '../components/ui/Badge';
import { Button, IconButton } from '../components/ui/Button';
import { Input, SearchInput } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { playbookMarketplaceApi, type MarketplacePlaybook, type MarketplaceFilter } from '../services/api/playbookMarketplace';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

type VendorFilter = '' | 'hikvision' | 'dahua' | 'axis' | 'uniview' | 'generic';

interface RateForm {
  score: number;
  review: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Vendor colors mapping
// ═══════════════════════════════════════════════════════════════════════

const vendorColors: Record<string, string> = {
  hikvision: 'text-blue-600 dark:text-blue-400',
  dahua: 'text-red-600 dark:text-red-400',
  axis: 'text-emerald-600 dark:text-emerald-400',
  uniview: 'text-purple-600 dark:text-purple-400',
  generic: 'text-slate-600 dark:text-slate-400',
};

const vendorLabels: Record<string, string> = {
  hikvision: 'Hikvision',
  dahua: 'Dahua',
  axis: 'Axis',
  uniview: 'Uniview',
  generic: 'Generic',
};

// ═══════════════════════════════════════════════════════════════════════
// StarRating Component
// ═══════════════════════════════════════════════════════════════════════

function StarRating({
  value,
  onChange,
  readonly = false,
}: {
  value: number;
  onChange?: (v: number) => void;
  readonly?: boolean;
}) {
  return (
    <div className="flex gap-1" role={readonly ? 'img' : 'radiogroup'} aria-label={`Rating: ${value} out of 5`}>
      {[1, 2, 3, 4, 5].map((star: number) => (
        <button
          key={star}
          type="button"
          disabled={readonly}
          onClick={() => onChange?.(star)}
          className={`${readonly ? 'cursor-default' : 'cursor-pointer hover:scale-110'} transition-transform`}
          aria-label={`${star} star${star > 1 ? 's' : ''}`}
        >
          <Star
            className={`w-5 h-5 ${
              star <= value
                ? 'fill-amber-400 text-amber-400'
                : 'fill-none text-slate-300 dark:text-slate-600'
            }`}
          />
        </button>
      ))}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// PlaybookCard Component
// ═══════════════════════════════════════════════════════════════════════

function PlaybookCard({
  playbook,
  onInstall,
  onRate,
  installing,
}: {
  playbook: MarketplacePlaybook;
  onInstall: (id: string) => void;
  onRate: (id: string) => void;
  installing: boolean;
}) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);

  return (
    <Card variant="interactive" className="flex flex-col h-full" onClick={() => setExpanded(!expanded)}>
      <CardContent>
        {/* Header */}
        <div className="flex items-start justify-between mb-3">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <h3 className="text-base font-semibold text-slate-900 dark:text-white truncate">
                {playbook.name}
              </h3>
              {playbook.verified && (
                <span title="Vendor Verified">
                  <ShieldCheck className="w-4 h-4 text-emerald-500 shrink-0" />
                </span>
              )}
            </div>
            <span className={`text-xs font-medium ${vendorColors[playbook.vendor] || vendorColors.generic}`}>
              {vendorLabels[playbook.vendor] || playbook.vendor}
            </span>
            <span className="text-xs text-slate-400 dark:text-slate-500 ml-2">v{playbook.version}</span>
          </div>
          <Badge variant={playbook.verified ? 'success' : 'neutral'} size="sm">
            {playbook.verified ? t('Verified') : t('Community')}
          </Badge>
        </div>

        {/* Description */}
        {playbook.description && (
          <p className="text-sm text-slate-600 dark:text-slate-400 mb-3 line-clamp-2">
            {playbook.description}
          </p>
        )}

        {/* Stats row */}
        <div className="flex items-center gap-4 text-xs text-slate-500 dark:text-slate-400 mb-3">
          <div className="flex items-center gap-1">
            <Star className="w-3.5 h-3.5 text-amber-400 fill-amber-400" />
            <span>{playbook.avg_rating.toFixed(1)}</span>
            <span className="text-slate-400">({playbook.review_count})</span>
          </div>
          <div className="flex items-center gap-1">
            <Download className="w-3.5 h-3.5" />
            <span>{playbook.install_count}</span>
          </div>
        </div>

        {/* Compat matrix (collapsible) */}
        {playbook.compat_matrix.length > 0 && (
          <>
            <button
              type="button"
              onClick={(e) => { e.stopPropagation(); setExpanded(!expanded); }}
              className="flex items-center gap-1 text-xs text-blue-600 dark:text-blue-400 hover:underline mb-2"
            >
              {expanded ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
              {t('Compatible Devices')} ({playbook.compat_matrix.length})
            </button>
            {expanded && (
              <div className="flex flex-wrap gap-1 mb-3">
                {playbook.compat_matrix.map((model: string) => (
                  <Badge key={model} variant="info" size="sm">
                    {model}
                  </Badge>
                ))}
              </div>
            )}
          </>
        )}

        {/* Actions */}
        <div className="flex items-center gap-2 mt-auto pt-2 border-t border-slate-100 dark:border-slate-700">
          <Button
            size="sm"
            variant="primary"
            onClick={(e) => { e.stopPropagation(); onInstall(playbook.id); }}
            disabled={installing}
          >
            {installing ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Download className="w-4 h-4" />
            )}
            {t('Install')}
          </Button>
          <IconButton
            icon={<MessageCircle className="w-4 h-4" />}
            variant="ghost"
            size="sm"
            label={t('Rate this playbook')}
            onClick={(e) => { e.stopPropagation(); onRate(playbook.id); }}
          />
        </div>
      </CardContent>
    </Card>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Main Page Component
// ═══════════════════════════════════════════════════════════════════════

export function PlaybookMarketplace() {
  const { t } = useTranslation();

  // ── State ────────────────────────────────────────────────────────
  const [playbooks, setPlaybooks] = useState<MarketplacePlaybook[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [search, setSearch] = useState('');
  const [vendorFilter, setVendorFilter] = useState<VendorFilter>('');
  const [minRating, setMinRating] = useState(0);
  const [verifiedOnly, setVerifiedOnly] = useState(false);

  // Pagination
  const [limit] = useState(20);
  const [offset, setOffset] = useState(0);

  // Install state
  const [installingId, setInstallingId] = useState<string | null>(null);

  // Rating modal
  const [rateModalOpen, setRateModalOpen] = useState(false);
  const [ratePlaybookId, setRatePlaybookId] = useState<string | null>(null);
  const [rateForm, setRateForm] = useState<RateForm>({ score: 0, review: '' });
  const [rateSubmitting, setRateSubmitting] = useState(false);

  // Share modal
  const [shareModalOpen, setShareModalOpen] = useState(false);
  const [sharePlaybookId, setSharePlaybookId] = useState<string | null>(null);
  const [shareTarget, setShareTarget] = useState('');

  // ── Data fetching ────────────────────────────────────────────────
  const fetchPlaybooks = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const filter: MarketplaceFilter = {
        limit,
        offset,
        search: search || undefined,
      };
      if (vendorFilter) filter.vendor = vendorFilter;
      if (minRating > 0) filter.min_rating = minRating;
      if (verifiedOnly) filter.verified = true;

      const data = await playbookMarketplaceApi.list(filter);
      setPlaybooks(data.playbooks);
      setTotal(data.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load playbooks');
    } finally {
      setLoading(false);
    }
  }, [search, vendorFilter, minRating, verifiedOnly, limit, offset]);

  useEffect(() => {
    fetchPlaybooks();
  }, [fetchPlaybooks]);

  // ── Actions ─────────────────────────────────────────────────────
  const handleInstall = useCallback(async (id: string) => {
    setInstallingId(id);
    try {
      await playbookMarketplaceApi.install(id);
      fetchPlaybooks();
    } catch (err) {
      console.error('Install failed:', err);
      setError(err instanceof Error ? err.message : 'Install failed');
    } finally {
      setInstallingId(null);
    }
  }, [fetchPlaybooks]);

  const openRateModal = useCallback((id: string) => {
    setRatePlaybookId(id);
    setRateForm({ score: 0, review: '' });
    setRateModalOpen(true);
  }, []);

  const handleRateSubmit = useCallback(async () => {
    if (!ratePlaybookId || rateForm.score === 0) return;
    setRateSubmitting(true);
    try {
      await playbookMarketplaceApi.rate(ratePlaybookId, rateForm.score, rateForm.review || undefined);
      setRateModalOpen(false);
      fetchPlaybooks();
    } catch (err) {
      console.error('Rate failed:', err);
    } finally {
      setRateSubmitting(false);
    }
  }, [ratePlaybookId, rateForm, fetchPlaybooks]);

  const handleShare = useCallback(async () => {
    if (!sharePlaybookId || !shareTarget) return;
    try {
      await playbookMarketplaceApi.share(sharePlaybookId, shareTarget);
      setShareModalOpen(false);
      setShareTarget('');
    } catch (err) {
      console.error('Share failed:', err);
    }
  }, [sharePlaybookId, shareTarget]);

  // ── Pagination ──────────────────────────────────────────────────
  const totalPages = Math.ceil(total / limit);
  const currentPage = Math.floor(offset / limit) + 1;

  // ── Render ──────────────────────────────────────────────────────
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-2">
            <Store className="w-6 h-6" />
            {t('Playbook Marketplace')}
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            {t('Pre-built playbooks for Hikvision, Dahua, Axis, Uniview and more')}
          </p>
        </div>
        <div className="flex items-center gap-2 text-sm text-slate-500 dark:text-slate-400">
          <span>{total} {t('playbooks')}</span>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3 p-4 bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
        {/* Search */}
        <div className="flex-1 min-w-[200px]">
          <SearchInput
            placeholder={t('Search playbooks...')}
            value={search}
            onChange={(e) => { setSearch(e.target.value); setOffset(0); }}
          />
        </div>

        {/* Vendor filter */}
        <select
          className="h-10 px-3 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-sm text-slate-700 dark:text-slate-300"
          value={vendorFilter}
          onChange={(e) => { setVendorFilter(e.target.value as VendorFilter); setOffset(0); }}
          aria-label={t('Filter by vendor')}
        >
          <option value="">{t('All Vendors')}</option>
          <option value="hikvision">Hikvision</option>
          <option value="dahua">Dahua</option>
          <option value="axis">Axis</option>
          <option value="uniview">Uniview</option>
          <option value="generic">{t('Generic')}</option>
        </select>

        {/* Min rating filter */}
        <select
          className="h-10 px-3 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-sm text-slate-700 dark:text-slate-300"
          value={minRating}
          onChange={(e) => { setMinRating(Number(e.target.value)); setOffset(0); }}
          aria-label={t('Minimum rating')}
        >
          <option value="0">{t('Any Rating')}</option>
          <option value="4">4+ ★</option>
          <option value="3">3+ ★</option>
          <option value="2">2+ ★</option>
        </select>

        {/* Verified toggle */}
        <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300 cursor-pointer">
          <input
            type="checkbox"
            checked={verifiedOnly}
            onChange={(e) => { setVerifiedOnly(e.target.checked); setOffset(0); }}
            className="rounded border-slate-300 dark:border-slate-600"
          />
          <ShieldCheck className="w-4 h-4 text-emerald-500" />
          {t('Verified only')}
        </label>
      </div>

      {/* Error state */}
      {error && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-700 dark:text-red-400">
          {error}
        </div>
      )}

      {/* Loading state */}
      {loading && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardContent>
                <div className="h-4 bg-slate-200 dark:bg-slate-700 rounded w-3/4 mb-3" />
                <div className="h-3 bg-slate-200 dark:bg-slate-700 rounded w-1/2 mb-2" />
                <div className="h-3 bg-slate-200 dark:bg-slate-700 rounded w-full mb-1" />
                <div className="h-3 bg-slate-200 dark:bg-slate-700 rounded w-2/3 mb-4" />
                <div className="h-8 bg-slate-200 dark:bg-slate-700 rounded w-20" />
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Playbook grid */}
      {!loading && !error && (
        <>
          {playbooks.length === 0 ? (
            <div className="text-center py-12 text-slate-500 dark:text-slate-400">
              <Store className="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p className="text-lg font-medium mb-1">{t('No playbooks found')}</p>
              <p className="text-sm">{t('Try adjusting your search or filters')}</p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {playbooks.map((playbook) => (
                <PlaybookCard
                  key={playbook.id}
                  playbook={playbook}
                  onInstall={handleInstall}
                  onRate={openRateModal}
                  installing={installingId === playbook.id}
                />
              ))}
            </div>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 pt-4">
              <Button
                variant="outline"
                size="sm"
                disabled={offset === 0}
                onClick={() => setOffset(Math.max(0, offset - limit))}
              >
                {t('Previous')}
              </Button>
              <span className="text-sm text-slate-500 dark:text-slate-400 px-3">
                {t('Page')} {currentPage} {t('of')} {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={offset + limit >= total}
                onClick={() => setOffset(offset + limit)}
              >
                {t('Next')}
              </Button>
            </div>
          )}
        </>
      )}

      {/* ── Rating Modal ──────────────────────────────────────────── */}
      <Modal
        isOpen={rateModalOpen}
        onClose={() => setRateModalOpen(false)}
        title={t('Rate this playbook')}
        size="sm"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setRateModalOpen(false)}>
              {t('Cancel')}
            </Button>
            <Button
              variant="primary"
              onClick={handleRateSubmit}
              disabled={rateForm.score === 0 || rateSubmitting}
            >
              {rateSubmitting ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Star className="w-4 h-4" />
              )}
              {t('Submit Rating')}
            </Button>
          </div>
        }
      >
        <div className="space-y-4">
          <div className="text-center">
            <p className="text-sm text-slate-600 dark:text-slate-400 mb-2">
              {t('How many stars would you give this playbook?')}
            </p>
            <div className="flex justify-center">
              <StarRating
                value={rateForm.score}
                onChange={(score) => setRateForm((prev) => ({ ...prev, score }))}
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t('Review (optional)')}
            </label>
            <textarea
              className="w-full h-24 px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-sm text-slate-700 dark:text-slate-300 resize-none"
              placeholder={t('Share your experience with this playbook...')}
              value={rateForm.review}
              onChange={(e) => setRateForm((prev) => ({ ...prev, review: e.target.value }))}
              maxLength={2000}
            />
            <p className="text-xs text-slate-400 text-right mt-1">
              {rateForm.review.length}/2000
            </p>
          </div>
        </div>
      </Modal>

      {/* ── Share Modal ───────────────────────────────────────────── */}
      <Modal
        isOpen={shareModalOpen}
        onClose={() => setShareModalOpen(false)}
        title={t('Share Playbook')}
        size="sm"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setShareModalOpen(false)}>
              {t('Cancel')}
            </Button>
            <Button variant="primary" onClick={handleShare} disabled={!shareTarget}>
              <ExternalLink className="w-4 h-4" />
              {t('Share')}
            </Button>
          </div>
        }
      >
        <div className="space-y-3">
          <p className="text-sm text-slate-600 dark:text-slate-400">
            {t('Share this playbook with another tenant. The playbook will appear in their marketplace.')}
          </p>
          <Input
            label={t('Target Tenant ID')}
            placeholder="tenant-xyz"
            value={shareTarget}
            onChange={(e) => setShareTarget(e.target.value)}
          />
        </div>
      </Modal>
    </div>
  );
}
