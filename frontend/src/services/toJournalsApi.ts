// ═══════════════════════════════════════════════════════════════════════
// toJournalsApi.ts — API для TO (Technical Operations) Journals
//
// Track 3: TO Compliance Automation
//   - UX-3.1: TO Journals with Regulatory Templates
//   - UX-3.4: AI Copilot for TO Journals
//
// Compliance:
//   - ISO 27001 A.12.4 (Audit trail — каждая мутация логируется)
//   - IEC 62443 SR 3.1 (Data integrity)
//   - OWASP ASVS V5.1 (Input validation через Zod)
// ═══════════════════════════════════════════════════════════════════════

import { request } from './api/client';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface TOJournal {
  id: string;
  work_order_id: string;
  region_code: string;
  status: TOJournalStatus;
  template_id: string;
  template_name: string;
  title: string;
  device_name: string;
  technician_name: string;
  generated_at: string | null;
  period_start: string;
  period_end: string;
  created_at: string;
  updated_at: string;
  meta: Record<string, unknown>;
}

export type TOJournalStatus = 'draft' | 'generated' | 'signed' | 'archived';

export interface TOJournalTemplate {
  id: string;
  region_code: string;
  name: string;
  description: string;
  regulatory_ref: string;
  version: string;
  sections: TemplateSection[];
  required_fields: string[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface TemplateSection {
  id: string;
  title: string;
  type: 'header' | 'checklist' | 'parts' | 'signatures' | 'qr' | 'notes' | 'custom';
  fields: TemplateField[];
  order: number;
  required: boolean;
}

export interface TemplateField {
  id: string;
  label: string;
  type: 'text' | 'textarea' | 'checkbox' | 'date' | 'signature' | 'photo' | 'select';
  required: boolean;
  placeholder?: string;
  options?: string[];
  default_value?: string;
}

export interface GenerateJournalRequest {
  work_order_id: string;
  template_id: string;
  period_start: string;
  period_end: string;
  region_code: string;
}

export interface GenerateJournalResponse {
  journal_id: string;
  pdf_url: string;
  preview_url: string;
  generated_at: string;
}

export interface JournalFilter {
  region_code?: string;
  status?: TOJournalStatus;
  work_order_id?: string;
  period_start?: string;
  period_end?: string;
  search?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// API Client
// ═══════════════════════════════════════════════════════════════════════

const BASE = '/to-journals';

export const toJournalsApi = {
  /** Получить список журналов с фильтрацией */
  getJournals: (filters?: JournalFilter) => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== '') {
          params.set(key, value);
        }
      });
    }
    const qs = params.toString();
    return request<TOJournal[]>(`${BASE}${qs ? '?' + qs : ''}`);
  },

  /** Получить один журнал по ID */
  getJournal: (id: string) => {
    return request<TOJournal>(`${BASE}/${id}`);
  },

  /** Сгенерировать журнал за период */
  generateJournal: (data: GenerateJournalRequest) => {
    return request<GenerateJournalResponse>(`${BASE}/generate`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  /** Получить шаблоны для региона */
  getTemplates: (regionCode: string) => {
    return request<TOJournalTemplate[]>(`${BASE}/templates/${regionCode}`);
  },

  /** Получить доступные регионы */
  getRegions: () => {
    return request<{ code: string; name: string; flag: string }[]>('/regions');
  },

  /** Предпросмотр PDF перед генерацией */
  previewJournal: (data: GenerateJournalRequest) => {
    return request<{ preview_url: string; pages: number }>(`${BASE}/preview`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  /** Скачать PDF журнала */
  downloadJournal: (id: string) => {
    return request<{ download_url: string }>(`${BASE}/${id}/download`);
  },

  /** Обновить статус журнала */
  updateJournalStatus: (id: string, status: TOJournalStatus) => {
    return request<TOJournal>(`${BASE}/${id}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    });
  },

  /** Удалить журнал */
  deleteJournal: (id: string) => {
    return request<{ status: string }>(`${BASE}/${id}`, {
      method: 'DELETE',
    });
  },

  // ═══════════════════════════════════════════════════════════════════
  // UX-3.4: AI Copilot for TO Journals
  // ═══════════════════════════════════════════════════════════════════

  /** AI: Получить suggested narrative */
  suggestNarrative: (journalId: string, context?: Record<string, unknown>) => {
    return request<{ suggestion: string; confidence: number }>(`${BASE}/${journalId}/ai/suggest`, {
      method: 'POST',
      body: JSON.stringify({ context }),
    });
  },

  /** AI: Отправить feedback (like/dislike) */
  sendAIFeedback: (journalId: string, messageId: string, score: 'like' | 'dislike') => {
    return request<{ status: string }>(`${BASE}/${journalId}/ai/feedback`, {
      method: 'POST',
      body: JSON.stringify({ message_id: messageId, score }),
    });
  },
};
