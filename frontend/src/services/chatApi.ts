// ═══════════════════════════════════════════════════════════════════════
// P2-CHAT: Real-Time Chat per Work Order — API service
// ═══════════════════════════════════════════════════════════════════════

import { request } from './api';

export interface ChatMessage {
  id: string;
  wo_id: string;
  user_id: string;
  user_name: string;
  text: string;
  message_type: 'text' | 'system' | 'voice' | 'image';
  attachments?: Attachment[];
  mentions?: string[];
  reaction?: string;
  read_by?: string[];
  created_at: string;
}

export interface Attachment {
  id: string;
  file_name: string;
  file_size: number;
  mime_type: string;
  storage_path?: string;
  thumbnail_path?: string;
}

export interface ChatHistoryResponse {
  messages: ChatMessage[];
  limit: number;
}

export interface SendMessageRequest {
  text?: string;
  message_type?: string;
  attachments?: Attachment[];
  mentions?: string[];
}

export const chatApi = {
  /** GET /api/v1/work-orders/{id}/chat — история сообщений */
  getHistory: (woId: string, limit = 50, before?: string) => {
    const params = new URLSearchParams({ limit: String(limit) });
    if (before) params.set('before', before);
    return request<ChatHistoryResponse>(`/work-orders/${woId}/chat?${params}`);
  },

  /** POST /api/v1/work-orders/{id}/chat — новое сообщение */
  sendMessage: (woId: string, data: SendMessageRequest) => {
    return request<ChatMessage>(`/work-orders/${woId}/chat`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  /** POST /api/v1/work-orders/{id}/chat/upload — загрузка файла */
  uploadFile: (woId: string, file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return request<Attachment>(`/work-orders/${woId}/chat/upload`, {
      method: 'POST',
      body: formData,
    });
  },
};
