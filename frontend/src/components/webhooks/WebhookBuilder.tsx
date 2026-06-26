// ═══════════════════════════════════════════════════════════════════════
// WebhookBuilder — визуальный конструктор вебхуков (заглушка).
//
// P2-3.1: Webhook Builder UI
//   - Event type selector
//   - Payload preview
//   - Test button с mock event
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { useTranslation } from 'react-i18next';
import { Webhook, Send, Copy } from 'lucide-react';

export function WebhookBuilder() {
  const { t } = useTranslation();

  return (
    <div className="p-4 md:p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
            {t('webhook_builder') || 'Webhook Builder'}
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400">
            {t('webhook_builder_desc') || 'Configure and test outgoing webhooks'}
          </p>
        </div>
        <button
          type="button"
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
        >
          <Send className="w-4 h-4" />
          {t('test_webhook') || 'Test Webhook'}
        </button>
      </div>

      {/* Coming Soon Placeholder */}
      <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-12 flex flex-col items-center justify-center text-slate-400">
        <Webhook className="w-16 h-16 mb-4 opacity-50" />
        <p className="text-lg font-medium text-slate-500 dark:text-slate-300 mb-2">
          {t('webhook_coming_soon') || 'Webhook Builder — Coming Soon'}
        </p>
        <p className="text-sm text-center max-w-md">
          {t('webhook_coming_soon_desc') ||
            'Visual webhook builder with event type selector, payload preview, and test mode will be available in the next release.'}
        </p>
      </div>
    </div>
  );
}
