// ═══════════════════════════════════════════════════════════════════════
// WorkflowCELInput — редактор CEL выражений (P2-2.1)
//
// Простой текстовый редактор с подсветкой синтаксиса CEL:
//   - Подсветка ключевых слов, строк, чисел
//   - Кнопка валидации
//   - Подсказки по синтаксису
// ═══════════════════════════════════════════════════════════════════════

import React, { useMemo, useCallback, useState } from 'react';

// ═══════════════════════════════════════════════════════════════════════
// CEL Syntax Highlighting
// ═══════════════════════════════════════════════════════════════════════

const CEL_KEYWORDS = [
  'true', 'false', 'null', 'in', 'as', '&&', '||', '!',
  '==', '!=', '<', '>', '<=', '>=', '+', '-', '*', '/', '%',
  'matches', 'contains', 'startsWith', 'endsWith', 'size',
  'has', 'all', 'exists', 'one', 'filter', 'map',
];

const CEL_KEYWORD_SET = new Set(CEL_KEYWORDS);

interface Token {
  type: 'keyword' | 'string' | 'number' | 'comment' | 'identifier' | 'punctuation' | 'whitespace';
  value: string;
}

function tokenizeCEL(code: string): Token[] {
  const tokens: Token[] = [];
  const re = /("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')|(\/\/.*)|(\b\d+\.?\d*\b)|(\b[a-zA-Z_]\w*\b)|(\s+)|(.)/g;
  let match: RegExpExecArray | null;

  while ((match = re.exec(code)) !== null) {
    if (match[1]) {
      tokens.push({ type: 'string', value: match[1] });
    } else if (match[2]) {
      tokens.push({ type: 'comment', value: match[2] });
    } else if (match[3]) {
      tokens.push({ type: 'number', value: match[3] });
    } else if (match[4]) {
      const word = match[4];
      tokens.push({
        type: CEL_KEYWORD_SET.has(word) ? 'keyword' : 'identifier',
        value: word,
      });
    } else if (match[5]) {
      tokens.push({ type: 'whitespace', value: match[5] });
    } else if (match[6]) {
      tokens.push({ type: 'punctuation', value: match[6] });
    }
  }

  return tokens;
}

// ═══════════════════════════════════════════════════════════════════════
// Color Map
// ═══════════════════════════════════════════════════════════════════════

const TOKEN_COLORS: Record<Token['type'], string> = {
  keyword: '#7c3aed',       // violet-600
  string: '#059669',        // emerald-600
  number: '#d97706',        // amber-600
  comment: '#94a3b8',       // slate-400
  identifier: '#1e293b',    // slate-800
  punctuation: '#64748b',   // slate-500
  whitespace: 'transparent',
};

// ═══════════════════════════════════════════════════════════════════════
// Help snippets
// ═══════════════════════════════════════════════════════════════════════

const CEL_SNIPPETS = [
  { label: 'Priority check', code: 'event.priority == "critical"' },
  { label: 'Time range', code: 'event.timestamp >= timestamp("2024-01-01T00:00:00Z")' },
  { label: 'Camera match', code: 'event.camera_id.startsWith("CAM-")' },
  { label: 'AND condition', code: 'event.priority == "high" && event.type == "motion"' },
  { label: 'OR condition', code: 'event.source == "sensor_01" || event.source == "sensor_02"' },
  { label: 'Size check', code: 'event.data.size() > 1024' },
];

// ═══════════════════════════════════════════════════════════════════════
// CEL Validation (простая проверка скобок)
// ═══════════════════════════════════════════════════════════════════════

function validateCEL(expr: string): { valid: boolean; error?: string } {
  if (!expr.trim()) {
    return { valid: false, error: 'Expression is empty' };
  }

  // Проверка парных скобок
  let depth = 0;
  for (const ch of expr) {
    if (ch === '(') depth++;
    if (ch === ')') depth--;
    if (depth < 0) return { valid: false, error: 'Unmatched closing parenthesis' };
  }
  if (depth > 0) return { valid: false, error: 'Unmatched opening parenthesis' };

  // Проверка парных кавычек
  const quotes = expr.match(/"/g);
  if (quotes && quotes.length % 2 !== 0) {
    return { valid: false, error: 'Unmatched double quotes' };
  }

  return { valid: true };
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

interface WorkflowCELInputProps {
  value: string;
  onChange: (value: string) => void;
  label?: string;
}

export function WorkflowCELInput({
  value,
  onChange,
  label = 'CEL Expression',
}: WorkflowCELInputProps) {
  const [showSnippets, setShowSnippets] = useState(false);

  const tokens = useMemo(() => tokenizeCEL(value), [value]);
  const validation = useMemo(() => validateCEL(value), [value]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      // Tab → insert 2 spaces
      if (e.key === 'Tab') {
        e.preventDefault();
        const target = e.currentTarget;
        const start = target.selectionStart;
        const end = target.selectionEnd;
        const newValue = value.slice(0, start) + '  ' + value.slice(end);
        onChange(newValue);
        // Restore cursor after React re-render
        requestAnimationFrame(() => {
          target.selectionStart = target.selectionEnd = start + 2;
        });
      }
    },
    [value, onChange]
  );

  const insertSnippet = useCallback(
    (code: string) => {
      onChange(code);
      setShowSnippets(false);
    },
    [onChange]
  );

  return (
    <div className="space-y-2">
      {/* ─── Label + Validation Status ───────────────────────────── */}
      <div className="flex items-center justify-between">
        <label className="text-sm font-medium text-slate-700 dark:text-slate-300">
          {label}
        </label>
        <div className="flex items-center gap-2">
          {value.trim() && (
            <span
              className={`text-xs px-1.5 py-0.5 rounded-full font-medium ${
                validation.valid
                  ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                  : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
              }`}
            >
              {validation.valid ? '✓ Valid' : '✗ Error'}
            </span>
          )}
          <button
            type="button"
            onClick={() => setShowSnippets(!showSnippets)}
            className="text-xs text-blue-600 dark:text-blue-400 hover:underline"
          >
            {showSnippets ? 'Hide snippets' : 'Snippets'}
          </button>
        </div>
      </div>

      {/* ─── Snippets ────────────────────────────────────────────── */}
      {showSnippets && (
        <div className="flex flex-wrap gap-1.5 p-2 bg-slate-50 dark:bg-slate-900/50 rounded-lg border border-slate-200 dark:border-slate-700">
          {CEL_SNIPPETS.map((snippet) => (
            <button
              key={snippet.label}
              type="button"
              onClick={() => insertSnippet(snippet.code)}
              className="text-xs px-2 py-1 rounded bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-600 text-slate-700 dark:text-slate-300 hover:bg-blue-50 dark:hover:bg-blue-900/20 hover:border-blue-300 transition-colors"
              title={snippet.code}
            >
              {snippet.label}
            </button>
          ))}
        </div>
      )}

      {/* ─── CodeMirror-like Editor ──────────────────────────────── */}
      <div className="relative">
        {/* Highlighted overlay */}
        <div
          className="absolute inset-0 p-3 font-mono text-sm leading-relaxed whitespace-pre-wrap break-all pointer-events-none"
          aria-hidden="true"
        >
          {tokens.map((token, i) => (
            <span key={i} style={{ color: TOKEN_COLORS[token.type] }}>
              {token.value}
            </span>
          ))}
        </div>

        {/* Transparent textarea for editing */}
        <textarea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={'Enter CEL expression, e.g. event.priority == "critical"'}
          className={[
            'relative w-full min-h-[80px] p-3 font-mono text-sm leading-relaxed',
            'bg-transparent text-transparent caret-slate-800 dark:caret-slate-200',
            'border rounded-lg resize-y',
            'focus:outline-none focus:ring-2 focus:ring-blue-500',
            validation.valid || !value.trim()
              ? 'border-slate-300 dark:border-slate-600'
              : 'border-red-300 dark:border-red-700',
          ].join(' ')}
          spellCheck={false}
        />
      </div>

      {/* ─── Error Message ─────────────────────────────────────────── */}
      {!validation.valid && value.trim() && (
        <p className="text-xs text-red-500">{validation.error}</p>
      )}

      {/* ─── Available Variables ──────────────────────────────────── */}
      <details className="text-xs text-slate-500 dark:text-slate-400">
        <summary className="cursor-pointer hover:text-slate-700 dark:hover:text-slate-300">
          Available variables
        </summary>
        <div className="mt-1 p-2 bg-slate-50 dark:bg-slate-900/50 rounded font-mono text-[11px] space-y-0.5">
          <div><span className="text-violet-600">event</span> — workflow trigger event</div>
          <div className="pl-4">event.type <span className="text-slate-400">string</span></div>
          <div className="pl-4">event.priority <span className="text-slate-400">string</span></div>
          <div className="pl-4">event.timestamp <span className="text-slate-400">timestamp</span></div>
          <div className="pl-4">event.camera_id <span className="text-slate-400">string</span></div>
          <div className="pl-4">event.source <span className="text-slate-400">string</span></div>
          <div className="pl-4">event.data <span className="text-slate-400">map(string, any)</span></div>
          <div><span className="text-violet-600">device</span> — target device info</div>
          <div className="pl-4">device.status <span className="text-slate-400">string</span></div>
          <div className="pl-4">device.type <span className="text-slate-400">string</span></div>
        </div>
      </details>
    </div>
  );
}
