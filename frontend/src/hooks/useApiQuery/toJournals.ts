// ═══════════════════════════════════════════════════════════════════════
// toJournals.ts — React Query hooks for TO Journals
//
// Track 3: TO Compliance Automation
//   - UX-3.1: TO Journals with Regulatory Templates
//   - UX-3.4: AI Copilot for TO Journals
// ═══════════════════════════════════════════════════════════════════════

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toJournalsApi } from '../../services/toJournalsApi';
import type {
  TOJournal,
  TOJournalTemplate,
  JournalFilter,
  GenerateJournalRequest,
  TOJournalStatus,
} from '../../services/toJournalsApi';
import { queryKeys, CACHE } from './shared';

// ═══════════════════════════════════════════════════════════════════════
// Queries
// ═══════════════════════════════════════════════════════════════════════

/** Получить список журналов с фильтрацией */
export function useTOJournals(filters?: JournalFilter) {
  return useQuery({
    queryKey: [...queryKeys.toJournals.all, filters],
    queryFn: () => toJournalsApi.getJournals(filters),
    staleTime: CACHE.LIST_STALE,
    gcTime: CACHE.LIST_GC,
  });
}

/** Получить один журнал по ID */
export function useTOJournal(id: string) {
  return useQuery({
    queryKey: queryKeys.toJournals.detail(id),
    queryFn: () => toJournalsApi.getJournal(id),
    staleTime: CACHE.LIST_STALE,
    gcTime: CACHE.LIST_GC,
    enabled: !!id,
  });
}

/** Получить шаблоны для региона */
export function useTOTemplates(regionCode: string) {
  return useQuery({
    queryKey: queryKeys.toJournals.templates(regionCode),
    queryFn: () => toJournalsApi.getTemplates(regionCode),
    staleTime: CACHE.REF_STALE,
    gcTime: CACHE.REF_GC,
    enabled: !!regionCode,
  });
}

/** Получить доступные регионы */
export function useRegions() {
  return useQuery({
    queryKey: queryKeys.toJournals.regions,
    queryFn: () => toJournalsApi.getRegions(),
    staleTime: CACHE.REF_STALE,
    gcTime: CACHE.REF_GC,
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Mutations
// ═══════════════════════════════════════════════════════════════════════

/** Сгенерировать журнал за период */
export function useGenerateJournal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: GenerateJournalRequest) =>
      toJournalsApi.generateJournal(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.toJournals.all });
    },
  });
}

/** Предпросмотр PDF перед генерацией */
export function usePreviewJournal() {
  return useMutation({
    mutationFn: (data: GenerateJournalRequest) =>
      toJournalsApi.previewJournal(data),
  });
}

/** Обновить статус журнала */
export function useUpdateJournalStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: TOJournalStatus }) =>
      toJournalsApi.updateJournalStatus(id, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.toJournals.all });
    },
  });
}

/** Удалить журнал */
export function useDeleteJournal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => toJournalsApi.deleteJournal(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.toJournals.all });
    },
  });
}

/** Скачать PDF журнала */
export function useDownloadJournal() {
  return useMutation({
    mutationFn: (id: string) => toJournalsApi.downloadJournal(id),
  });
}

// ═══════════════════════════════════════════════════════════════════════
// UX-3.4: AI Copilot Mutations
// ═══════════════════════════════════════════════════════════════════════

/** AI: Получить suggested narrative */
export function useSuggestNarrative() {
  return useMutation({
    mutationFn: ({ journalId, context }: { journalId: string; context?: Record<string, unknown> }) =>
      toJournalsApi.suggestNarrative(journalId, context),
  });
}

/** AI: Отправить feedback */
export function useSendAIFeedback() {
  return useMutation({
    mutationFn: ({ journalId, messageId, score }: {
      journalId: string;
      messageId: string;
      score: 'like' | 'dislike';
    }) => toJournalsApi.sendAIFeedback(journalId, messageId, score),
  });
}
