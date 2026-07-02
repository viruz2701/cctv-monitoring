// ═══════════════════════════════════════════════════════════════════════
// DocumentPreview.tsx — Live PDF preview with edit capabilities
//
// Track 3: TO Compliance Automation
//   - UX-3.3: TO Document Preview & Editing
//
// Features:
//   - Live preview PDF in iframe
//   - Drag-n-drop photo reordering
//   - Version history sidebar
//   - Editable sections: logo, header, checklist, parts, signatures, QR
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { toDocumentsApi } from '../../services/toDocumentsApi';
import { FieldEditor } from './FieldEditor';
import { Button, Badge, Tabs, Card, Skeleton } from '../ui';
import {
  FileText,
  Eye,
  Download,
  History,
  RefreshCw,
  Upload,
  Maximize2,
  Loader2,
  RotateCcw,
  CheckCircle,
  GripVertical,
  X,
} from '../ui/Icons';
import type { TODocument, DocumentSection, DocumentVersion, PhotoItem } from '../../services/toDocumentsApi';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface DocumentPreviewProps {
  documentId: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Section Icons
// ═══════════════════════════════════════════════════════════════════════

const SECTION_ICONS: Record<string, React.ElementType> = {
  logo: Upload,
  header: FileText,
  checklist: CheckCircle,
  parts: FileText,
  signatures: FileText,
  qr: FileText,
  notes: FileText,
};

// ═══════════════════════════════════════════════════════════════════════
// DocumentPreview Component
// ═══════════════════════════════════════════════════════════════════════

export function DocumentPreview({ documentId }: DocumentPreviewProps) {
  const { t } = useTranslation();
  const [document, setDocument] = useState<TODocument | null>(null);
  const [versions, setVersions] = useState<DocumentVersion[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeSection, setActiveSection] = useState<string>('');
  const [showVersionHistory, setShowVersionHistory] = useState(false);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);

  // ── Load document ──────────────────────────────────────────────
  const loadDocument = useCallback(async () => {
    if (!documentId) return;
    setLoading(true);
    setError(null);
    try {
      const doc = await toDocumentsApi.getDocument(documentId);
      setDocument(doc);
      if (doc.sections.length > 0 && !activeSection) {
        setActiveSection(doc.sections[0].id);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load document');
    } finally {
      setLoading(false);
    }
  }, [documentId, activeSection]);

  // ── Load versions ──────────────────────────────────────────────
  const loadVersions = useCallback(async () => {
    if (!documentId) return;
    try {
      const v = await toDocumentsApi.getVersionHistory(documentId);
      setVersions(v);
    } catch {
      // silent fail for versions
    }
  }, [documentId]);

  // ── Generate preview ───────────────────────────────────────────
  const handlePreview = useCallback(async () => {
    if (!documentId) return;
    setPreviewLoading(true);
    try {
      const result = await toDocumentsApi.getPreview(documentId);
      setPreviewUrl(result.preview_url);
    } catch {
      // handled by UI
    } finally {
      setPreviewLoading(false);
    }
  }, [documentId]);

  // ── Export PDF ─────────────────────────────────────────────────
  const handleExport = useCallback(async () => {
    if (!documentId) return;
    try {
      const result = await toDocumentsApi.exportPDF(documentId);
      window.open(result.download_url, '_blank');
    } catch {
      // handled by UI
    }
  }, [documentId]);

  // ── Restore version ────────────────────────────────────────────
  const handleRestore = useCallback(async (versionId: string) => {
    if (!documentId) return;
    try {
      const restored = await toDocumentsApi.restoreVersion(documentId, versionId);
      setDocument(restored);
      setShowVersionHistory(false);
    } catch {
      // handled by UI
    }
  }, [documentId]);

  // ── Init ───────────────────────────────────────────────────────
  React.useEffect(() => {
    loadDocument();
  }, [loadDocument]);

  // ── Loading state ──────────────────────────────────────────────
  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
        <div className="grid grid-cols-3 gap-4">
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
        </div>
      </div>
    );
  }

  // ── Error state ────────────────────────────────────────────────
  if (error || !document) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <FileText className="w-12 h-12 text-slate-300 dark:text-slate-600 mb-4" />
        <p className="text-sm text-slate-500 dark:text-slate-400">
          {error || t('documents.not_found', 'Document not found')}
        </p>
        <Button
          variant="outline"
          size="sm"
          className="mt-4"
          icon={<RefreshCw className="w-4 h-4" />}
          onClick={loadDocument}
        >
          {t('common.retry', 'Retry')}
        </Button>
      </div>
    );
  }

  const currentSection = document.sections.find((s) => s.id === activeSection);

  return (
    <div className="flex gap-6">
      {/* ── Left: Section Navigation ──────────────────────────── */}
      <div className="w-56 flex-shrink-0 space-y-1">
        <h3 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-3">
          {t('documents.sections', 'Sections')}
        </h3>
        {document.sections.map((section) => {
          const Icon = SECTION_ICONS[section.type] ?? FileText;
          return (
            <button
              key={section.id}
              onClick={() => setActiveSection(section.id)}
              className={`
                w-full text-left px-3 py-2 rounded-lg text-sm transition-colors flex items-center gap-2
                ${activeSection === section.id
                  ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-400 font-medium'
                  : 'text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800/50'
                }
                ${section.locked ? 'opacity-60 cursor-not-allowed' : ''}
              `}
              disabled={section.locked}
            >
              <Icon className="w-4 h-4 flex-shrink-0" />
              <span className="truncate">{section.title}</span>
              {section.locked && (
                <span className="ml-auto text-xs text-slate-400">🔒</span>
              )}
            </button>
          );
        })}
      </div>

      {/* ── Center: Editor / Preview ───────────────────────────── */}
      <div className="flex-1 min-w-0">
        {/* Toolbar */}
        <div className="flex items-center justify-between mb-4 pb-3 border-b border-slate-200 dark:border-slate-700">
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
              {t('documents.document_title', 'Document')} v{document.version}
            </h2>
            <Badge variant={document.status === 'final' ? 'success' : document.status === 'draft' ? 'info' : 'neutral'}>
              {document.status}
            </Badge>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              icon={<History className="w-4 h-4" />}
              onClick={() => { loadVersions(); setShowVersionHistory(!showVersionHistory); }}
            >
              {t('documents.history', 'History')}
            </Button>
            <Button
              variant="outline"
              size="sm"
              icon={previewLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Eye className="w-4 h-4" />}
              onClick={handlePreview}
              disabled={previewLoading}
            >
              {t('documents.preview', 'Preview')}
            </Button>
            <Button
              variant="primary"
              size="sm"
              icon={<Download className="w-4 h-4" />}
              onClick={handleExport}
            >
              {t('documents.export_pdf', 'Export PDF')}
            </Button>
          </div>
        </div>

        {/* Preview iframe */}
        {previewUrl && (
          <div className="mb-4 border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden">
            <div className="flex items-center justify-between px-3 py-2 bg-slate-50 dark:bg-slate-800">
              <span className="text-xs font-medium text-slate-500">
                {t('documents.live_preview', 'Live Preview')}
              </span>
              <div className="flex items-center gap-1">
                <Button
                  variant="ghost"
                  size="sm"
                  icon={<RefreshCw className="w-3.5 h-3.5" />}
                  onClick={handlePreview}
                />
                <Button
                  variant="ghost"
                  size="sm"
                  icon={<Maximize2 className="w-3.5 h-3.5" />}
                  onClick={() => window.open(previewUrl, '_blank')}
                />
              </div>
            </div>
            <iframe
              src={previewUrl}
              className="w-full h-96 bg-white"
              title="Document Preview"
            />
          </div>
        )}

        {/* Section Editor */}
        {currentSection && (
          <FieldEditor
            section={currentSection}
            documentId={documentId}
            onUpdate={(updatedSection: DocumentSection) => {
              setDocument((prev) => {
                if (!prev) return prev;
                return {
                  ...prev,
                  sections: prev.sections.map((s) =>
                    s.id === updatedSection.id ? updatedSection : s,
                  ),
                };
              });
            }}
          />
        )}
      </div>

      {/* ── Right: Version History Sidebar ─────────────────────── */}
      {showVersionHistory && (
        <div className="w-72 flex-shrink-0 border-l border-slate-200 dark:border-slate-700 pl-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300">
              {t('documents.version_history', 'Version History')}
            </h3>
            <Button
              variant="ghost"
              size="sm"
              icon={<X className="w-4 h-4" />}
              onClick={() => setShowVersionHistory(false)}
            />
          </div>
          <div className="space-y-2 max-h-96 overflow-y-auto">
            {versions.length === 0 ? (
              <p className="text-xs text-slate-400 text-center py-4">
                {t('documents.no_versions', 'No version history')}
              </p>
            ) : (
              versions.map((version) => (
                <Card key={version.id} className="p-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <span className="text-sm font-medium text-slate-800 dark:text-slate-200">
                        v{version.version}
                      </span>
                      <p className="text-xs text-slate-400 mt-0.5">
                        {version.changes.length} change(s)
                      </p>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      icon={<RotateCcw className="w-3.5 h-3.5" />}
                      onClick={() => handleRestore(version.id)}
                      aria-label={t('documents.restore', 'Restore')}
                    />
                  </div>
                  <p className="text-xs text-slate-400 mt-1">
                    {new Date(version.created_at).toLocaleDateString()}
                  </p>
                </Card>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Drag-n-Drop Photo Reorder Component
// ═══════════════════════════════════════════════════════════════════════

interface PhotoReorderProps {
  photos: PhotoItem[];
  onReorder: (photoIds: string[]) => void;
}

export function PhotoReorder({ photos, onReorder }: PhotoReorderProps) {
  const [items, setItems] = useState(photos);
  const [dragIndex, setDragIndex] = useState<number | null>(null);

  const handleDragStart = (index: number) => {
    setDragIndex(index);
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    if (dragIndex === null || dragIndex === index) return;

    const newItems = [...items];
    const [removed] = newItems.splice(dragIndex, 1);
    newItems.splice(index, 0, removed);
    setItems(newItems);
    setDragIndex(index);
  };

  const handleDragEnd = () => {
    setDragIndex(null);
    onReorder(items.map((p) => p.id));
  };

  return (
    <div className="grid grid-cols-3 gap-2">
      {items.map((photo, index) => (
        <div
          key={photo.id}
          draggable
          onDragStart={() => handleDragStart(index)}
          onDragOver={(e) => handleDragOver(e, index)}
          onDragEnd={handleDragEnd}
          className={`
            relative group rounded-lg overflow-hidden border border-slate-200 dark:border-slate-700
            ${dragIndex === index ? 'opacity-50 border-blue-500' : ''}
            cursor-grab active:cursor-grabbing
          `}
        >
          <img
            src={photo.url}
            alt={photo.caption}
            className="w-full h-20 object-cover"
          />
          <div className="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-colors flex items-center justify-center">
            <GripVertical className="w-5 h-5 text-white opacity-0 group-hover:opacity-100 transition-opacity" />
          </div>
          <span className="absolute bottom-1 left-1 text-[10px] text-white bg-black/50 px-1 rounded">
            {photo.caption}
          </span>
        </div>
      ))}
    </div>
  );
}
