// ═══════════════════════════════════════════════════════════════════════
// toDocumentsApi.ts — API для TO Document Preview & Editing
//
// Track 3: TO Compliance Automation
//   - UX-3.3: TO Document Preview & Editing
//
// Compliance:
//   - ISO 27001 A.12.4 (Version history audit trail)
//   - IEC 62443 SR 3.1 (Data integrity)
//   - OWASP ASVS V6 (Cryptographic storage)
// ═══════════════════════════════════════════════════════════════════════

import { request } from './api/client';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface TODocument {
  id: string;
  journal_id: string;
  version: number;
  status: DocumentStatus;
  sections: DocumentSection[];
  created_by: string;
  created_at: string;
  updated_at: string;
  pdf_url: string | null;
  preview_url: string | null;
}

export type DocumentStatus = 'draft' | 'preview' | 'final' | 'signed';

export interface DocumentSection {
  id: string;
  type: 'logo' | 'header' | 'checklist' | 'parts' | 'signatures' | 'qr' | 'notes';
  title: string;
  content: Record<string, unknown>;
  order: number;
  locked: boolean;
}

export interface DocumentVersion {
  id: string;
  document_id: string;
  version: number;
  changes: VersionChange[];
  created_by: string;
  created_at: string;
}

export interface VersionChange {
  field: string;
  old_value: unknown;
  new_value: unknown;
  section_id: string;
}

export interface PhotoItem {
  id: string;
  url: string;
  caption: string;
  order: number;
  section_id: string;
}

export interface DocumentFieldUpdate {
  section_id: string;
  field_id: string;
  value: unknown;
}

// ═══════════════════════════════════════════════════════════════════════
// API Client
// ═══════════════════════════════════════════════════════════════════════

const BASE = '/to-documents';

export const toDocumentsApi = {
  /** Получить документ по ID */
  getDocument: (id: string) => {
    return request<TODocument>(`${BASE}/${id}`);
  },

  /** Получить версионную историю */
  getVersionHistory: (documentId: string) => {
    return request<DocumentVersion[]>(`${BASE}/${documentId}/versions`);
  },

  /** Создать новую версию */
  createVersion: (documentId: string) => {
    return request<DocumentVersion>(`${BASE}/${documentId}/versions`, {
      method: 'POST',
    });
  },

  /** Откатить к версии */
  restoreVersion: (documentId: string, versionId: string) => {
    return request<TODocument>(`${BASE}/${documentId}/versions/${versionId}/restore`, {
      method: 'POST',
    });
  },

  /** Обновить содержимое секции */
  updateSection: (documentId: string, sectionId: string, content: Record<string, unknown>) => {
    return request<TODocument>(`${BASE}/${documentId}/sections/${sectionId}`, {
      method: 'PATCH',
      body: JSON.stringify({ content }),
    });
  },

  /** Обновить одно поле в секции (с version tracking) */
  updateField: (documentId: string, data: DocumentFieldUpdate) => {
    return request<TODocument>(`${BASE}/${documentId}/field`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  /** Загрузить/переупорядочить фото */
  reorderPhotos: (documentId: string, sectionId: string, photoIds: string[]) => {
    return request<{ status: string }>(`${BASE}/${documentId}/sections/${sectionId}/photos/reorder`, {
      method: 'PUT',
      body: JSON.stringify({ photo_ids: photoIds }),
    });
  },

  /** Загрузить фото */
  uploadPhoto: (documentId: string, sectionId: string, file: File) => {
    const formData = new FormData();
    formData.append('photo', file);
    return request<PhotoItem>(`${BASE}/${documentId}/sections/${sectionId}/photos`, {
      method: 'POST',
      body: formData,
    });
  },

  /** Удалить фото */
  deletePhoto: (documentId: string, photoId: string) => {
    return request<{ status: string }>(`${BASE}/${documentId}/photos/${photoId}`, {
      method: 'DELETE',
    });
  },

  /** Получить PDF preview */
  getPreview: (documentId: string) => {
    return request<{ preview_url: string; expires_at: string }>(`${BASE}/${documentId}/preview`);
  },

  /** Экспортировать в PDF */
  exportPDF: (documentId: string) => {
    return request<{ download_url: string }>(`${BASE}/${documentId}/export`, {
      method: 'POST',
    });
  },

  /** Обновить логотип */
  updateLogo: (documentId: string, file: File) => {
    const formData = new FormData();
    formData.append('logo', file);
    return request<TODocument>(`${BASE}/${documentId}/logo`, {
      method: 'PUT',
      body: formData,
    });
  },

  /** Подписать документ (электронная подпись) */
  signDocument: (documentId: string, signature: string) => {
    return request<TODocument>(`${BASE}/${documentId}/sign`, {
      method: 'POST',
      body: JSON.stringify({ signature }),
    });
  },
};
