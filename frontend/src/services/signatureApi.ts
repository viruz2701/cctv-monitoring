// ═══════════════════════════════════════════════════════════════════════
// signatureApi.ts — Hash-Chain Digital Signatures API
//
// UX-3.6: Hash-Chain Digital Signatures (СТБ 34.101.27)
//   - Подпись включает hash предыдущего ТО
//   - Visual chain: "Previous TO: TO-200 · Hash: a3f4...2b1c ✓"
//   - Verifier tool для аудиторов
//   - Fallback на простую подпись (с warning)
//
// Compliance:
//   - СТБ 34.101.27 п. 6.3 (Hash chain integrity)
//   - СТБ 34.101.30 (bash-256/belt-gcm криптография)
//   - ISO 27001 A.12.4 (Audit trail)
//   - OWASP ASVS V6 (Cryptographic storage)
// ═══════════════════════════════════════════════════════════════════════

import { request } from './api/client';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

/** Хеш-цепочка подписей */
export interface SignatureChain {
  /** ID цепочки */
  id: string;
  /** ID WorkOrder (текущий) */
  work_order_id: string;
  /** Тип документа */
  document_type: 'work_order' | 'to_journal' | 'inspection_report';
  /** Хеш текущего документа (bash-256) */
  current_hash: string;
  /** Хеш предыдущего документа в цепочке */
  previous_hash: string | null;
  /** ID предыдущего документа */
  previous_document_id: string | null;
  /** Номер предыдущего документа (TO-200) */
  previous_document_label: string | null;
  /** Цифровая подпись (bign) */
  signature: string;
  /** Алгоритм подписи */
  algorithm: 'bash-256' | 'bash-512' | 'belt-gcm';
  /** Сертификат (bign public key) */
  certificate_fingerprint: string;
  /** Кем подписано */
  signed_by: string;
  /** Когда подписано */
  signed_at: string;
  /** Статус верификации */
  verified: boolean | null;
  /** Статус целостности цепочки */
  chain_valid: boolean | null;
  /** Если верифицировано — кем */
  verified_by?: string;
  /** Если верифицировано — когда */
  verified_at?: string;
  /** Является ли fallback-подписью (без хеш-цепочек) */
  is_fallback: boolean;
  /** Сообщение об ошибке fallback */
  fallback_reason?: string;
}

/** Результат верификации */
export interface VerificationResult {
  chain_id: string;
  /** Хеш текущего документа корректен */
  hash_valid: boolean;
  /** Подпись корректна */
  signature_valid: boolean;
  /** Хеш цепочки корректен (prev_hash совпадает) */
  chain_valid: boolean | null;
  /** Верификация пройдена */
  overall_valid: boolean;
  /** Ссылка на предыдущий документ */
  previous_document: {
    id: string;
    label: string;
    hash: string;
  } | null;
  /** Время последней верификации */
  verified_at: string;
  /** Детали ошибок */
  errors: string[];
}

/** Статистика цепочки */
export interface ChainStatistics {
  total_signatures: number;
  first_document: string;
  last_document: string;
  broken_links: number;
  verified_count: number;
  pending_verification: number;
  average_verification_time_ms: number;
}

// ═══════════════════════════════════════════════════════════════════════
// API Client
// ═══════════════════════════════════════════════════════════════════════

const BASE = '/signatures';

export const signatureApi = {
  /** Получить цепочку подписей для WorkOrder */
  getChain: (workOrderId: string) =>
    request<SignatureChain>(`${BASE}/chains/${workOrderId}`),

  /** Получить все цепочки (для аудитора) */
  getAllChains: (params?: {
    limit?: number;
    offset?: number;
    status?: 'verified' | 'pending' | 'broken';
  }) => {
    const query = new URLSearchParams();
    if (params?.limit) query.set('limit', String(params.limit));
    if (params?.offset) query.set('offset', String(params.offset));
    if (params?.status) query.set('status', params.status);
    return request<SignatureChain[]>(`${BASE}/chains?${query.toString()}`);
  },

  /** Подписать документ (создать запись в цепочке) */
  signDocument: (data: {
    work_order_id: string;
    document_type: SignatureChain['document_type'];
    previous_document_id?: string;
    previous_hash?: string;
    signature: string;
    algorithm?: SignatureChain['algorithm'];
    certificate_fingerprint: string;
    signed_by: string;
  }) =>
    request<SignatureChain>(`${BASE}/chains`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  /** Верифицировать подпись */
  verifySignature: (chainId: string) =>
    request<VerificationResult>(`${BASE}/chains/${chainId}/verify`, {
      method: 'POST',
    }),

  /** Верифицировать всю цепочку */
  verifyChain: (workOrderId: string) =>
    request<VerificationResult>(`${BASE}/chains/${workOrderId}/verify-chain`, {
      method: 'POST',
    }),

  /** Получить статистику цепочки */
  getStatistics: (workOrderId: string) =>
    request<ChainStatistics>(`${BASE}/chains/${workOrderId}/statistics`),

  /** Экспорт цепочки для аудита */
  exportChain: (workOrderId: string) =>
    request<{ csv_url: string }>(`${BASE}/chains/${workOrderId}/export`),

  /** Создать fallback-подпись (без хеш-цепочек) */
  signFallback: (data: {
    work_order_id: string;
    document_type: SignatureChain['document_type'];
    signature: string;
    signed_by: string;
    reason: string;
  }) =>
    request<SignatureChain>(`${BASE}/chains/fallback`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
};
