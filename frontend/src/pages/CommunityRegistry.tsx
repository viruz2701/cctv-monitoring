// ═══════════════════════════════════════════════════════════════════════
// CommunityRegistry — Публичный реестр Protocol Descriptor'ов (PROTO-07)
//
// Docker Hub для CCTV протоколов — community публикует и обменивается
// дескрипторами для различных вендоров (Hikvision, Dahua, ONVIF и т.д.).
//
// Features:
//   - Card grid community descriptors
//   - Search by vendor/keywords
//   - Filter: verified only, min rating, sort options
//   - Detail view: JSON preview, rating, download
//   - Publish modal (auth required)
//   - Rating modal (1-5 stars)
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation)
//   - Rate limiting для публикации
// ═══════════════════════════════════════════════════════════════════════

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Search,
  Star,
  Filter,
  Download,
  Upload,
  ChevronDown,
  ChevronUp,
  ShieldCheck,
  Loader2,
  X,
  Eye,
  Package,
} from '../components/ui/Icons';
import { Card, CardContent } from '../components/ui/Card';
import { Badge } from '../components/ui/Badge';
import { Button, IconButton } from '../components/ui/Button';
import { Input, SearchInput } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import {
  communityRegistryApi,
  type CommunityDescriptorSummary,
  type CommunityDescriptor,
  type CommunityDescriptorFilter,
  type PublishDescriptorRequest,
} from '../services/api/communityRegistry';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

type SortField = 'rating' | 'downloads' | 'created_at' | 'vendor';

interface RateForm {
  score: number;
}

interface PublishForm {
  vendor: string;
  version: string;
  descriptor: string;
}

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
// DescriptorCard Component
// ═══════════════════════════════════════════════════════════════════════

function DescriptorCard({
  descriptor,
  onView,
  onRate,
  onDownload,
  downloading,
}: {
  descriptor: CommunityDescriptorSummary;
  onView: (vendor: string) => void;
  onRate: (vendor: string) => void;
  onDownload: (vendor: string) => void;
  downloading: boolean;
}) {
  return (
    <Card variant="interactive" className="flex flex-col h-full">
      <CardContent>
        {/* Header */}
        <div className="flex items-start justify-between mb-3">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <h3 className="text-base font-semibold text-slate-900 dark:text-white truncate">
                {descriptor.vendor}
              </h3>
              {descriptor.verified && (
                <span title="Verified by team">
                  <ShieldCheck className="w-4 h-4 text-emerald-500 shrink-0" />
                </span>
              )}
            </div>
            <span className="text-xs text-slate-400 dark:text-slate-500">
              v{descriptor.version}
            </span>
          </div>
          <Badge variant={descriptor.verified ? 'success' : 'neutral'} size="sm">
            {descriptor.verified ? 'Verified' : 'Community'}
          </Badge>
        </div>

        {/* Stats row */}
        <div className="flex items-center gap-4 text-xs text-slate-500 dark:text-slate-400 mb-4">
          <div className="flex items-center gap-1">
            <Star className="w-3.5 h-3.5 text-amber-400 fill-amber-400" />
            <span>{descriptor.rating.toFixed(1)}</span>
          </div>
          <div className="flex items-center gap-1">
            <Download className="w-3.5 h-3.5" />
            <span>{descriptor.downloads}</span>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2 mt-auto pt-2 border-t border-slate-100 dark:border-slate-700">
          <Button
            size="sm"
            variant="primary"
            onClick={(e) => { e.stopPropagation(); onView(descriptor.vendor); }}
          >
            <Eye className="w-4 h-4" />
            View
          </Button>
          <Button
            size="sm"
            variant="secondary"
            onClick={(e) => { e.stopPropagation(); onDownload(descriptor.vendor); }}
            disabled={downloading}
          >
            {downloading ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Download className="w-4 h-4" />
            )}
            Download
          </Button>
          <IconButton
            icon={<Star className="w-4 h-4" />}
            variant="ghost"
            size="sm"
            label="Rate this descriptor"
            onClick={(e) => { e.stopPropagation(); onRate(descriptor.vendor); }}
          />
        </div>
      </CardContent>
    </Card>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// DetailModal Component
// ═══════════════════════════════════════════════════════════════════════

function DetailModal({
  descriptor,
  onClose,
  onRate,
}: {
  descriptor: CommunityDescriptor | null;
  onClose: () => void;
  onRate: (vendor: string) => void;
}) {
  if (!descriptor) return null;

  return (
    <Modal isOpen onClose={onClose} title={descriptor.vendor}>
      <div className="space-y-4">
        {/* Meta info */}
        <div className="flex items-center gap-4 text-sm text-slate-600 dark:text-slate-400">
          <span>Version: <strong>{descriptor.version}</strong></span>
          {descriptor.verified && (
            <Badge variant="success" size="sm">Verified</Badge>
          )}
        </div>

        {/* Stats */}
        <div className="flex items-center gap-6 text-sm">
          <div className="flex items-center gap-1">
            <Star className="w-4 h-4 text-amber-400 fill-amber-400" />
            <span className="font-medium">{descriptor.rating.toFixed(1)}</span>
          </div>
          <div className="flex items-center gap-1">
            <Download className="w-4 h-4 text-slate-500" />
            <span>{descriptor.downloads} downloads</span>
          </div>
        </div>

        {/* JSON Preview */}
        <div>
          <h4 className="text-sm font-semibold text-slate-800 dark:text-slate-200 mb-2">
            Descriptor JSON
          </h4>
          <pre className="bg-slate-50 dark:bg-slate-800 rounded-lg p-4 text-xs overflow-auto max-h-96 border border-slate-200 dark:border-slate-700">
            {JSON.stringify(descriptor.descriptor, null, 2)}
          </pre>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2 pt-2">
          <Button
            size="sm"
            variant="secondary"
            onClick={() => onRate(descriptor.vendor)}
          >
            <Star className="w-4 h-4" />
            Rate
          </Button>
        </div>
      </div>
    </Modal>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// PublishModal Component
// ═══════════════════════════════════════════════════════════════════════

function PublishModal({
  open,
  onClose,
  onPublish,
}: {
  open: boolean;
  onClose: () => void;
  onPublish: (req: PublishDescriptorRequest) => Promise<void>;
}) {
  const [form, setForm] = useState<PublishForm>({
    vendor: '',
    version: '1.0.0',
    descriptor: '',
  });
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    // OWASP ASVS V5.1: input validation
    if (!form.vendor.trim()) {
      setError('Vendor name is required');
      return;
    }
    if (form.vendor.trim().length > 200) {
      setError('Vendor name must be <= 200 characters');
      return;
    }
    if (!form.version.trim()) {
      setError('Version is required');
      return;
    }
    if (form.version.trim().length > 50) {
      setError('Version must be <= 50 characters');
      return;
    }

    let parsedDescriptor: unknown;
    try {
      parsedDescriptor = JSON.parse(form.descriptor);
    } catch {
      setError('Invalid JSON in descriptor field');
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      await onPublish({
        vendor: form.vendor.trim(),
        version: form.version.trim(),
        descriptor: parsedDescriptor,
      });
      setForm({ vendor: '', version: '1.0.0', descriptor: '' });
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to publish');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal isOpen={open} onClose={onClose} title="Publish Community Descriptor">
      <div className="space-y-4">
        {/* Validation error */}
        {error && (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3 text-sm text-red-700 dark:text-red-400">
            {error}
          </div>
        )}

        {/* Vendor */}
        <div>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
            Vendor Name
          </label>
          <Input
            value={form.vendor}
            onChange={(e) => setForm({ ...form, vendor: e.target.value })}
            placeholder="e.g., hikvision, dahua, axis"
            maxLength={200}
          />
        </div>

        {/* Version */}
        <div>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
            Version
          </label>
          <Input
            value={form.version}
            onChange={(e) => setForm({ ...form, version: e.target.value })}
            placeholder="1.0.0"
            maxLength={50}
          />
        </div>

        {/* Descriptor JSON */}
        <div>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
            Descriptor JSON
          </label>
          <textarea
            value={form.descriptor}
            onChange={(e) => setForm({ ...form, descriptor: e.target.value })}
            placeholder='{
  "endpoints": [...],
  "auth": { "type": "basic" },
  ...
}'
            className="w-full h-48 px-3 py-2 text-sm font-mono border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-y"
            spellCheck={false}
          />
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 pt-2">
          <Button size="sm" variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            size="sm"
            variant="primary"
            onClick={handleSubmit}
            disabled={submitting}
          >
            {submitting ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Upload className="w-4 h-4" />
            )}
            Publish
          </Button>
        </div>
      </div>
    </Modal>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// RateModal Component
// ═══════════════════════════════════════════════════════════════════════

function RateModal({
  vendor,
  open,
  onClose,
  onSubmit,
}: {
  vendor: string | null;
  open: boolean;
  onClose: () => void;
  onSubmit: (vendor: string, score: number) => Promise<void>;
}) {
  const [score, setScore] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setScore(0);
      setError(null);
    }
  }, [open]);

  const handleSubmit = async () => {
    if (score < 1 || score > 5) {
      setError('Please select a rating (1-5)');
      return;
    }

    if (!vendor) return;

    setSubmitting(true);
    setError(null);

    try {
      await onSubmit(vendor, score);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit rating');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal isOpen={open} onClose={onClose} title={`Rate ${vendor || 'Descriptor'}`}>
      <div className="space-y-4">
        {error && (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3 text-sm text-red-700 dark:text-red-400">
            {error}
          </div>
        )}

        <div className="flex justify-center py-4">
          <StarRating value={score} onChange={setScore} />
        </div>

        <div className="text-center text-sm text-slate-500 dark:text-slate-400">
          {score > 0 ? `You rated ${score} out of 5` : 'Click a star to rate'}
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <Button size="sm" variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            size="sm"
            variant="primary"
            onClick={handleSubmit}
            disabled={submitting || score === 0}
          >
            {submitting ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Star className="w-4 h-4" />
            )}
            Submit Rating
          </Button>
        </div>
      </div>
    </Modal>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Main Page Component
// ═══════════════════════════════════════════════════════════════════════

export function CommunityRegistry() {
  // ── State ────────────────────────────────────────────────────────
  const [descriptors, setDescriptors] = useState<CommunityDescriptorSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [search, setSearch] = useState('');
  const [minRating, setMinRating] = useState(0);
  const [verifiedOnly, setVerifiedOnly] = useState(false);
  const [sortBy, setSortBy] = useState<SortField>('rating');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');

  // Pagination
  const [page, setPage] = useState(1);
  const pageSize = 20;

  // Detail view
  const [detailDescriptor, setDetailDescriptor] = useState<CommunityDescriptor | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  // Download state
  const [downloadingVendor, setDownloadingVendor] = useState<string | null>(null);

  // Modals
  const [publishModalOpen, setPublishModalOpen] = useState(false);
  const [rateModalOpen, setRateModalOpen] = useState(false);
  const [rateVendor, setRateVendor] = useState<string | null>(null);

  // ── Data fetching ────────────────────────────────────────────────
  const fetchDescriptors = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const filter: CommunityDescriptorFilter = {
        page,
        page_size: pageSize,
        sort_by: sortBy,
        sort_dir: sortDir,
      };
      if (search) filter.search = search;
      if (minRating > 0) filter.min_rating = minRating;
      if (verifiedOnly) filter.verified = true;

      const data = await communityRegistryApi.list(filter);
      setDescriptors(data.descriptors);
      setTotal(data.total);
      setTotalPages(data.total_pages);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load descriptors');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, search, minRating, verifiedOnly, sortBy, sortDir]);

  useEffect(() => {
    fetchDescriptors();
  }, [fetchDescriptors]);

  // ── Handlers ─────────────────────────────────────────────────────
  const handleSearchSubmit = useCallback((e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    fetchDescriptors();
  }, [fetchDescriptors]);

  const handleViewDetail = useCallback(async (vendor: string) => {
    setDetailLoading(true);
    try {
      const descriptor = await communityRegistryApi.get(vendor);
      setDetailDescriptor(descriptor);
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to load ${vendor}`);
    } finally {
      setDetailLoading(false);
    }
  }, []);

  const handleDownload = useCallback(async (vendor: string) => {
    setDownloadingVendor(vendor);
    try {
      const descriptor = await communityRegistryApi.download(vendor);
      setDetailDescriptor(descriptor);
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to download ${vendor}`);
    } finally {
      setDownloadingVendor(null);
    }
  }, []);

  const handleRate = useCallback(async (vendor: string, score: number) => {
    await communityRegistryApi.rate(vendor, score);
    // Refresh list to update average rating
    fetchDescriptors();
  }, [fetchDescriptors]);

  const handlePublish = useCallback(async (req: PublishDescriptorRequest) => {
    await communityRegistryApi.publish(req);
    // Refresh list
    fetchDescriptors();
  }, [fetchDescriptors]);

  // ── Pagination helpers ───────────────────────────────────────────
  const pageNumbers = useMemo(() => {
    const pages: number[] = [];
    const maxVisible = 5;
    let start = Math.max(1, page - Math.floor(maxVisible / 2));
    const end = Math.min(totalPages, start + maxVisible - 1);

    if (end - start + 1 < maxVisible) {
      start = Math.max(1, end - maxVisible + 1);
    }

    for (let i = start; i <= end; i++) {
      pages.push(i);
    }
    return pages;
  }, [page, totalPages]);

  // ── Filter controls ──────────────────────────────────────────────
  const hasActiveFilters = search || minRating > 0 || verifiedOnly;

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
            Community Protocol Registry
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            {total} shared descriptors from the community
          </p>
        </div>
        <Button
          variant="primary"
          onClick={() => setPublishModalOpen(true)}
        >
          <Upload className="w-4 h-4" />
          Publish Descriptor
        </Button>
      </div>

      {/* Search & Filters */}
      <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 p-4 mb-6">
        <div className="flex flex-wrap items-center gap-3">
          {/* Search */}
          <form onSubmit={handleSearchSubmit} className="flex-1 min-w-[200px]">
            <SearchInput
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search by vendor..."
            />
          </form>

          {/* Min Rating */}
          <div className="flex items-center gap-2">
            <label className="text-xs text-slate-500 dark:text-slate-400 whitespace-nowrap">
              Min Rating:
            </label>
            <select
              value={minRating}
              onChange={(e) => { setMinRating(Number(e.target.value)); setPage(1); }}
              className="text-sm border border-slate-300 dark:border-slate-600 rounded-lg px-2 py-1.5 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
            >
              <option value={0}>Any</option>
              <option value={3}>3+</option>
              <option value={4}>4+</option>
              <option value={4.5}>4.5+</option>
            </select>
          </div>

          {/* Verified filter */}
          <label className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400 cursor-pointer">
            <input
              type="checkbox"
              checked={verifiedOnly}
              onChange={(e) => { setVerifiedOnly(e.target.checked); setPage(1); }}
              className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500"
            />
            Verified only
          </label>

          {/* Sort */}
          <div className="flex items-center gap-2">
            <label className="text-xs text-slate-500 dark:text-slate-400 whitespace-nowrap">
              Sort:
            </label>
            <select
              value={sortBy}
              onChange={(e) => { setSortBy(e.target.value as SortField); setPage(1); }}
              className="text-sm border border-slate-300 dark:border-slate-600 rounded-lg px-2 py-1.5 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
            >
              <option value="rating">Rating</option>
              <option value="downloads">Downloads</option>
              <option value="created_at">Newest</option>
              <option value="vendor">Vendor</option>
            </select>
            <button
              type="button"
              onClick={() => { setSortDir(sortDir === 'desc' ? 'asc' : 'desc'); setPage(1); }}
              className="p-1.5 rounded-lg border border-slate-300 dark:border-slate-600 hover:bg-slate-50 dark:hover:bg-slate-700"
              title={sortDir === 'desc' ? 'Descending' : 'Ascending'}
            >
              {sortDir === 'desc' ? <ChevronDown className="w-4 h-4" /> : <ChevronUp className="w-4 h-4" />}
            </button>
          </div>

          {/* Reset filters */}
          {hasActiveFilters && (
            <button
              type="button"
              onClick={() => {
                setSearch('');
                setMinRating(0);
                setVerifiedOnly(false);
                setSortBy('rating');
                setSortDir('desc');
                setPage(1);
              }}
              className="text-sm text-blue-600 dark:text-blue-400 hover:underline"
            >
              Reset filters
            </button>
          )}
        </div>
      </div>

      {/* Error state */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-xl p-4 mb-6">
          <div className="flex items-center gap-2 text-red-700 dark:text-red-400">
            <X className="w-5 h-5 shrink-0" />
            <p className="text-sm">{error}</p>
          </div>
        </div>
      )}

      {/* Loading state */}
      {loading && (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
        </div>
      )}

      {/* Empty state */}
      {!loading && !error && descriptors.length === 0 && (
        <div className="text-center py-12">
          <Package className="w-12 h-12 mx-auto text-slate-300 dark:text-slate-600 mb-4" />
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-1">
            No descriptors found
          </h3>
          <p className="text-sm text-slate-500 dark:text-slate-400">
            {hasActiveFilters
              ? 'Try adjusting your search or filters'
              : 'Be the first to publish a community descriptor'}
          </p>
        </div>
      )}

      {/* Grid */}
      {!loading && descriptors.length > 0 && (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 mb-6">
            {descriptors.map((d) => (
              <DescriptorCard
                key={d.id}
                descriptor={d}
                onView={handleViewDetail}
                onRate={(vendor) => { setRateVendor(vendor); setRateModalOpen(true); }}
                onDownload={handleDownload}
                downloading={downloadingVendor === d.vendor}
              />
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2">
              <Button
                size="sm"
                variant="ghost"
                onClick={() => setPage(Math.max(1, page - 1))}
                disabled={page === 1}
              >
                Previous
              </Button>
              {pageNumbers.map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => setPage(p)}
                  className={`w-8 h-8 text-sm rounded-lg ${
                    p === page
                      ? 'bg-blue-600 text-white'
                      : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'
                  }`}
                >
                  {p}
                </button>
              ))}
              <Button
                size="sm"
                variant="ghost"
                onClick={() => setPage(Math.min(totalPages, page + 1))}
                disabled={page === totalPages}
              >
                Next
              </Button>
            </div>
          )}
        </>
      )}

      {/* Detail Modal */}
      {detailLoading && (
        <Modal isOpen onClose={() => setDetailDescriptor(null)} title="Loading...">
          <div className="flex justify-center py-8">
            <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
          </div>
        </Modal>
      )}
      <DetailModal
        descriptor={detailDescriptor}
        onClose={() => setDetailDescriptor(null)}
        onRate={(vendor) => { setRateVendor(vendor); setRateModalOpen(true); }}
      />

      {/* Publish Modal */}
      <PublishModal
        open={publishModalOpen}
        onClose={() => setPublishModalOpen(false)}
        onPublish={handlePublish}
      />

      {/* Rate Modal */}
      <RateModal
        vendor={rateVendor}
        open={rateModalOpen}
        onClose={() => { setRateModalOpen(false); setRateVendor(null); }}
        onSubmit={handleRate}
      />
    </div>
  );
}
