// ═══════════════════════════════════════════════════════════════════════
// DescriptorPreview — JSON preview дескриптора (PROTO-06)
//
// Отображает текущий дескриптор в форматированном JSON с подсветкой
// синтаксиса. Использует <pre><code> для читаемого preview.
//
// Compliance:
//   - WCAG 2.1 AA (read-only, focusable для screen readers)
// ═══════════════════════════════════════════════════════════════════════

import React, { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Copy, Check } from '../../components/ui/Icons';
import { useCurrentDescriptor } from '../../store/descriptorStore';

// ─── JSON syntax highlighting ──────────────────────────────────────

interface ColorMatch {
  start: number;
  end: number;
  color: string;
  text: string;
}

function highlightJson(json: string): React.ReactNode {
  type Pattern = { regex: RegExp; color: string };
  const patterns: Pattern[] = [
    { regex: /("(?:[^"\\]|\\.)*")\s*:/g, color: 'text-purple-600 dark:text-purple-400' },
    { regex: /:"(?:[^"\\]|\\.)*"/g, color: 'text-emerald-600 dark:text-emerald-400' },
    { regex: /:\s*(true|false)/g, color: 'text-amber-600 dark:text-amber-400' },
    { regex: /:\s*null/g, color: 'text-red-500 dark:text-red-400' },
    { regex: /:\s*-?\d+\.?\d*/g, color: 'text-blue-600 dark:text-blue-400' },
  ];

  const matches: ColorMatch[] = [];

  patterns.forEach(({ regex, color }) => {
    const r = new RegExp(regex.source, 'g');
    let m: RegExpExecArray | null;
    while ((m = r.exec(json)) !== null) {
      matches.push({
        start: m.index,
        end: m.index + m[0].length,
        color,
        text: m[0],
      });
    }
  });

  matches.sort((a, b) => a.start - b.start);

  // Merge overlapping matches
  const merged: ColorMatch[] = [];
  for (const m of matches) {
    if (merged.length > 0 && m.start <= merged[merged.length - 1].end) {
      const last = merged[merged.length - 1];
      if (m.end > last.end) {
        last.end = m.end;
        last.text = json.slice(last.start, last.end);
      }
    } else {
      merged.push({ ...m });
    }
  }

  const parts: React.ReactNode[] = [];
  let lastIndex = 0;

  for (const m of merged) {
    if (m.start > lastIndex) {
      parts.push(
        <span key={`t${lastIndex}`} className="text-slate-700 dark:text-slate-300">
          {json.slice(lastIndex, m.start)}
        </span>,
      );
    }
    const key = `m${m.start}`;
    if (m.text.includes(':') && !m.text.startsWith(':')) {
      // Key-value pair match: color the value part
      const colonIdx = m.text.indexOf(':');
      parts.push(
        <span key={key}>
          <span className="text-purple-600 dark:text-purple-400">{m.text.slice(0, colonIdx)}</span>
          <span className="text-slate-700 dark:text-slate-300">:</span>
          <span className={m.color}>{m.text.slice(colonIdx)}</span>
        </span>,
      );
    } else {
      parts.push(
        <span key={key} className={m.color}>
          {m.text}
        </span>,
      );
    }
    lastIndex = m.end;
  }

  if (lastIndex < json.length) {
    parts.push(
      <span key={`t${lastIndex}`} className="text-slate-700 dark:text-slate-300">
        {json.slice(lastIndex)}
      </span>,
    );
  }

  return parts.length > 0 ? parts : json;
}

// ─── Component ──────────────────────────────────────────────────────

export function DescriptorPreview() {
  const { t } = useTranslation();
  const descriptor = useCurrentDescriptor();
  const [copied, setCopied] = React.useState(false);

  const json = useMemo(
    () => JSON.stringify(descriptor, null, 2),
    [descriptor],
  );

  const highlighted = useMemo(() => highlightJson(json), [json]);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(json);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback for non-secure contexts
      const ta = document.createElement('textarea');
      ta.value = json;
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      document.body.removeChild(ta);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 overflow-hidden">
      {/* Toolbar */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
        <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
          {t('descriptors.previewTitle')}
        </span>
        <button
          type="button"
          onClick={handleCopy}
          className="flex items-center gap-1 px-2 py-1 text-xs rounded
                     hover:bg-slate-200 dark:hover:bg-slate-700
                     text-slate-500 dark:text-slate-400 transition-colors"
          aria-label={t('descriptors.copy')}
        >
          {copied ? (
            <>
              <Check className="w-3.5 h-3.5 text-emerald-500" />
              <span className="text-emerald-500">{t('descriptors.copied')}</span>
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" />
              <span>{t('descriptors.copy')}</span>
            </>
          )}
        </button>
      </div>

      {/* JSON Content */}
      <div className="overflow-auto max-h-[600px]">
        <pre
          className="p-4 text-xs font-mono leading-relaxed whitespace-pre"
          tabIndex={0}
          role="code"
          aria-label={t('descriptors.previewAria')}
        >
          <code>{highlighted}</code>
        </pre>
      </div>
    </div>
  );
}
