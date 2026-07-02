// ═══════════════════════════════════════════════════════════════════════
// AICopilot.tsx — AI Assistant for TO Journal narratives
//
// Track 3: TO Compliance Automation
//   - UX-3.4: AI Copilot for TO Journals
//
// Feature Flag: ai_copilot_to_journals
//
// Features:
//   - "Suggest Narrative" — AI предлагает текст на основе контекста
//   - Diff view: предложение AI vs текущий текст
//   - Accept/reject/edit пользователем
//   - Feedback loop 👍/👎
//   - Offline fallback
//
// Compliance:
//   - OWASP ASVS V1.8 (Feature flags)
//   - IEC 62443 SR 7.1 (Timeouts — 60s)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useSuggestNarrative, useSendAIFeedback } from '../../hooks/useApiQuery/toJournals';
import { Button, Card, Badge, Tabs } from '../ui';
import {
  Sparkles,
  ThumbsUp,
  ThumbsDown,
  Check,
  X,
  Edit3,
  RefreshCw,
  AlertTriangle,
  Loader2,
  FileText,
  Wifi,
  WifiOff,
} from '../ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface AICopilotProps {
  journalId: string;
  onClose: () => void;
}

interface DiffLine {
  type: 'same' | 'added' | 'removed';
  text: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Diff Calculator
// ═══════════════════════════════════════════════════════════════════════

function computeDiff(original: string, suggestion: string): DiffLine[] {
  const origLines = original.split('\n');
  const suggLines = suggestion.split('\n');
  const maxLen = Math.max(origLines.length, suggLines.length);
  const result: DiffLine[] = [];

  for (let i = 0; i < maxLen; i++) {
    const orig = origLines[i] ?? '';
    const sugg = suggLines[i] ?? '';

    if (orig === sugg) {
      result.push({ type: 'same', text: orig });
    } else if (orig && sugg) {
      // Lines differ — show both removed and added
      if (orig.trim()) {
        result.push({ type: 'removed', text: orig });
      }
      if (sugg.trim()) {
        result.push({ type: 'added', text: sugg });
      }
    } else if (orig) {
      result.push({ type: 'removed', text: orig });
    } else if (sugg) {
      result.push({ type: 'added', text: sugg });
    }
  }

  return result;
}

// ═══════════════════════════════════════════════════════════════════════
// DiffView Component
// ═══════════════════════════════════════════════════════════════════════

function DiffView({ original, suggestion }: { original: string; suggestion: string }) {
  const diffs = computeDiff(original, suggestion);

  return (
    <div className="font-mono text-xs leading-relaxed space-y-0.5">
      {diffs.map((line, i) => {
        let style = '';
        let prefix = ' ';
        switch (line.type) {
          case 'added':
            style = 'bg-emerald-50 dark:bg-emerald-900/20 text-emerald-800 dark:text-emerald-300';
            prefix = '+';
            break;
          case 'removed':
            style = 'bg-red-50 dark:bg-red-900/20 text-red-800 dark:text-red-300 line-through';
            prefix = '-';
            break;
          default:
            style = 'text-slate-600 dark:text-slate-400';
        }
        return (
          <div key={i} className={`px-3 py-0.5 rounded ${style}`}>
            <span className="select-none mr-2 opacity-50">{prefix}</span>
            {line.text || '\u00A0'}
          </div>
        );
      })}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// AICopilot Component
// ═══════════════════════════════════════════════════════════════════════

export function AICopilot({ journalId, onClose }: AICopilotProps) {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('suggest');
  const [originalText, setOriginalText] = useState('');
  const [suggestedText, setSuggestedText] = useState('');
  const [currentText, setCurrentText] = useState('');
  const [feedback, setFeedback] = useState<'like' | 'dislike' | null>(null);
  const [isOnline, setIsOnline] = useState(navigator.onLine);
  const [showDiff, setShowDiff] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const suggestMutation = useSuggestNarrative();
  const feedbackMutation = useSendAIFeedback();

  // ── Listen for online/offline ─────────────────────────────────
  React.useEffect(() => {
    const handleOnline = () => setIsOnline(true);
    const handleOffline = () => setIsOnline(false);
    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);
    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  // ── Suggest Narrative ─────────────────────────────────────────
  const handleSuggest = useCallback(async () => {
    setError(null);
    setFeedback(null);
    setSuggestedText('');
    setShowDiff(false);

    if (!isOnline) {
      setError(t('ai_copilot.offline', 'You are offline. AI suggestions require an internet connection.'));
      return;
    }

    try {
      const result = await suggestMutation.mutateAsync({
        journalId,
        context: { current_narrative: originalText },
      });
      setSuggestedText(result.suggestion);
      setCurrentText(result.suggestion);
      setShowDiff(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('ai_copilot.error', 'Failed to get AI suggestion'));
    }
  }, [journalId, originalText, isOnline, suggestMutation, t]);

  // ── Accept Suggestion ─────────────────────────────────────────
  const handleAccept = useCallback(() => {
    setOriginalText(currentText);
    setShowDiff(false);
    setSuggestedText('');
  }, [currentText]);

  // ── Reject Suggestion ─────────────────────────────────────────
  const handleReject = useCallback(() => {
    setCurrentText(originalText);
    setShowDiff(false);
    setSuggestedText('');
  }, [originalText]);

  // ── Edit Suggestion ───────────────────────────────────────────
  const handleEdit = useCallback(() => {
    setShowDiff(false);
    textareaRef.current?.focus();
  }, []);

  // ── Send Feedback ─────────────────────────────────────────────
  const handleFeedback = useCallback(async (score: 'like' | 'dislike') => {
    setFeedback(score);
    try {
      await feedbackMutation.mutateAsync({
        journalId,
        messageId: `suggest-${Date.now()}`,
        score,
      });
    } catch {
      // Silent — feedback is non-critical
    }
  }, [journalId, feedbackMutation]);

  // ── Tabs ──────────────────────────────────────────────────────
  const tabs = [
    { id: 'suggest', label: t('ai_copilot.suggest', 'Suggest Narrative'), icon: <Sparkles className="w-4 h-4" /> },
    { id: 'assist', label: t('ai_copilot.assist', 'Assist'), icon: <Edit3 className="w-4 h-4" /> },
  ];

  return (
    <div className="space-y-4">
      {/* ── Header ──────────────────────────────────────────────── */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Sparkles className="w-5 h-5 text-blue-500" />
          <span className="font-medium text-slate-800 dark:text-slate-200">
            {t('ai_copilot.title', 'AI Journal Assistant')}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {/* Online/Offline indicator */}
          <Badge variant={isOnline ? 'success' : 'danger'} size="sm" dot>
            {isOnline ? t('ai_copilot.online', 'Online') : t('ai_copilot.offline_badge', 'Offline')}
          </Badge>
        </div>
      </div>

      {/* ── Tabs ────────────────────────────────────────────────── */}
      <Tabs tabs={tabs} activeTab={activeTab} onChange={setActiveTab}>
        <div className="mt-0">
          {activeTab === 'suggest' && (
            <div className="space-y-4">
              {/* Current Narrative Input */}
              <div>
                <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1.5">
                  {t('ai_copilot.current_narrative', 'Current Narrative')}
                </label>
                <textarea
                  ref={textareaRef}
                  className="w-full min-h-[120px] p-3 text-sm rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 resize-y"
                  value={originalText}
                  onChange={(e) => setOriginalText(e.target.value)}
                  placeholder={t('ai_copilot.narrative_placeholder', 'Enter the current journal narrative...')}
                />
              </div>

              {/* Suggest Button */}
              <div className="flex items-center gap-2">
                <Button
                  variant="primary"
                  icon={suggestMutation.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Sparkles className="w-4 h-4" />}
                  onClick={handleSuggest}
                  loading={suggestMutation.isPending}
                  disabled={!originalText.trim() || suggestMutation.isPending}
                >
                  {t('ai_copilot.suggest_btn', 'Suggest Narrative')}
                </Button>

                {suggestedText && (
                  <Button
                    variant="outline"
                    icon={<RefreshCw className="w-4 h-4" />}
                    onClick={handleSuggest}
                    disabled={suggestMutation.isPending}
                  >
                    {t('ai_copilot.regenerate', 'Regenerate')}
                  </Button>
                )}
              </div>

              {/* Error */}
              {error && (
                <div className="flex items-start gap-2 p-3 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
                  <AlertTriangle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
                  <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
                </div>
              )}

              {/* Offline Fallback */}
              {!isOnline && (
                <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800">
                  <WifiOff className="w-4 h-4 text-amber-500 mt-0.5 flex-shrink-0" />
                  <div>
                    <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
                      {t('ai_copilot.offline_title', 'Offline Mode')}
                    </p>
                    <p className="text-xs text-amber-700 dark:text-amber-400 mt-1">
                      {t('ai_copilot.offline_desc', 'AI suggestions are unavailable offline. Connect to the internet to use this feature, or continue editing manually.')}
                    </p>
                  </div>
                </div>
              )}

              {/* Diff View */}
              {showDiff && suggestedText && (
                <Card className="overflow-hidden">
                  <div className="flex items-center justify-between px-4 py-2 bg-slate-50 dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
                    <span className="text-xs font-medium text-slate-500">
                      {t('ai_copilot.diff_title', 'Changes Preview')}
                    </span>
                    <Badge variant="info" size="sm">
                      {t('ai_copilot.ai_suggestion', 'AI Suggestion')}
                    </Badge>
                  </div>
                  <div className="p-0 max-h-48 overflow-y-auto">
                    <DiffView original={originalText} suggestion={suggestedText} />
                  </div>
                </Card>
              )}

              {/* Editable Text Area (after suggestion) */}
              {suggestedText && !showDiff && (
                <div>
                  <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1.5">
                    {t('ai_copilot.edited_text', 'Edited Narrative')}
                  </label>
                  <textarea
                    className="w-full min-h-[120px] p-3 text-sm rounded-lg border border-emerald-300 dark:border-emerald-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-emerald-500 resize-y"
                    value={currentText}
                    onChange={(e) => setCurrentText(e.target.value)}
                  />
                </div>
              )}

              {/* Accept / Reject / Edit Actions */}
              {suggestedText && (
                <div className="flex items-center justify-between pt-2">
                  <div className="flex items-center gap-2">
                    <Button
                      variant="primary"
                      size="sm"
                      icon={<Check className="w-4 h-4" />}
                      onClick={handleAccept}
                    >
                      {t('ai_copilot.accept', 'Accept')}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      icon={<X className="w-4 h-4" />}
                      onClick={handleReject}
                    >
                      {t('ai_copilot.reject', 'Reject')}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      icon={<Edit3 className="w-4 h-4" />}
                      onClick={handleEdit}
                    >
                      {t('ai_copilot.edit', 'Edit')}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      icon={showDiff ? <FileText className="w-4 h-4" /> : <FileText className="w-4 h-4" />}
                      onClick={() => setShowDiff(!showDiff)}
                    >
                      {showDiff ? t('ai_copilot.hide_diff', 'Hide Diff') : t('ai_copilot.show_diff', 'Show Diff')}
                    </Button>
                  </div>

                  {/* Feedback */}
                  <div className="flex items-center gap-1">
                    <span className="text-xs text-slate-400 mr-1">
                      {t('ai_copilot.helpful', 'Was this helpful?')}
                    </span>
                    <Button
                      variant="ghost"
                      size="sm"
                      icon={<ThumbsUp className={`w-4 h-4 ${feedback === 'like' ? 'text-blue-500 fill-blue-500' : ''}`} />}
                      onClick={() => handleFeedback('like')}
                      disabled={feedback !== null}
                    />
                    <Button
                      variant="ghost"
                      size="sm"
                      icon={<ThumbsDown className={`w-4 h-4 ${feedback === 'dislike' ? 'text-red-500 fill-red-500' : ''}`} />}
                      onClick={() => handleFeedback('dislike')}
                      disabled={feedback !== null}
                    />
                  </div>
                </div>
              )}

              {/* Loading indicator */}
              {suggestMutation.isPending && (
                <div className="flex items-center justify-center py-6">
                  <div className="flex items-center gap-3 text-sm text-slate-500">
                    <Loader2 className="w-5 h-5 animate-spin" />
                    {t('ai_copilot.generating', 'AI is generating a suggestion...')}
                  </div>
                </div>
              )}
            </div>
          )}

          {activeTab === 'assist' && (
            <div className="py-8 text-center">
              <Edit3 className="w-12 h-12 text-slate-300 dark:text-slate-600 mx-auto mb-3" />
              <p className="text-sm text-slate-500 dark:text-slate-400">
                {t('ai_copilot.assist_coming', 'Advanced AI assist features coming soon. Use "Suggest Narrative" to get started.')}
              </p>
            </div>
          )}
        </div>
      </Tabs>
    </div>
  );
}
