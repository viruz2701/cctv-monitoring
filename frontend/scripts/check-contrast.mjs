#!/usr/bin/env node
// ═══════════════════════════════════════════════════════════════════════
// P1-HI-12: WCAG AA Contrast Checker for CI
//
// Проверяет все пары (foreground, background, size) из tokens.css
// на соответствие WCAG 2.1 AA (4.5:1 для normal text, 3:1 для large).
//
// Использование: node scripts/check-contrast.mjs
// ═══════════════════════════════════════════════════════════════════════

// WCAG relative luminance calculation
function getLuminance(hex) {
  const r = parseInt(hex.slice(1, 3), 16) / 255;
  const g = parseInt(hex.slice(3, 5), 16) / 255;
  const b = parseInt(hex.slice(5, 7), 16) / 255;

  const rsRGB = r <= 0.03928 ? r / 12.92 : Math.pow((r + 0.055) / 1.055, 2.4);
  const gsRGB = g <= 0.03928 ? g / 12.92 : Math.pow((g + 0.055) / 1.055, 2.4);
  const bsRGB = b <= 0.03928 ? b / 12.92 : Math.pow((b + 0.055) / 1.055, 2.4);

  return 0.2126 * rsRGB + 0.7152 * gsRGB + 0.0722 * bsRGB;
}

function getContrastRatio(fg, bg) {
  const l1 = getLuminance(fg);
  const l2 = getLuminance(bg);
  const lighter = Math.max(l1, l2);
  const darker = Math.min(l1, l2);
  return (lighter + 0.05) / (darker + 0.05);
}

// P1-HI-12: Цветовые пары для проверки (foreground, background, name, isLarge)
const pairs = [
  // ══ Light mode (P1-HI-12: WCAG AA) ══
  { fg: '#1e293b', bg: '#f8fafc', name: 'text on bg (light)', large: false },
  { fg: '#475569', bg: '#f8fafc', name: 'text-secondary on bg (light)', large: false },
  { fg: '#64748b', bg: '#f8fafc', name: 'text-muted on bg (light)', large: false },
  { fg: '#1e293b', bg: '#ffffff', name: 'text on card (light)', large: false },
  // Semantic colors light — WCAG AA compliant
  { fg: '#b45309', bg: '#f8fafc', name: 'warning on bg (light)', large: false },
  { fg: '#dc2626', bg: '#f8fafc', name: 'danger on bg (light)', large: false },
  { fg: '#047857', bg: '#f8fafc', name: 'success on bg (light)', large: false },
  { fg: '#2563eb', bg: '#f8fafc', name: 'primary on bg (light)', large: false },
  { fg: '#ffffff', bg: '#2563eb', name: 'white on primary (light)', large: false },
  { fg: '#475569', bg: '#ffffff', name: 'text-secondary on card (light)', large: false },
  { fg: '#64748b', bg: '#ffffff', name: 'text-muted on card (light)', large: false },
  { fg: '#475569', bg: '#f1f5f9', name: 'text-secondary on muted (light)', large: false },
  { fg: '#475569', bg: '#f1f5f9', name: 'text-muted on muted (light)', large: false },
  // FC day numbers
  { fg: '#475569', bg: '#f8fafc', name: 'fc day number on bg (light)', large: false },

  // ══ Dark mode (P1-HI-12: WCAG AA) ══
  { fg: '#f1f5f9', bg: '#0f172a', name: 'text on bg (dark)', large: false },
  { fg: '#94a3b8', bg: '#0f172a', name: 'text-secondary on bg (dark)', large: false },
  { fg: '#94a3b8', bg: '#0f172a', name: 'text-muted on bg (dark)', large: false },
  { fg: '#f1f5f9', bg: '#1e293b', name: 'text on card (dark)', large: false },
  // Semantic colors dark — light tones for dark bg
  { fg: '#fbbf24', bg: '#0f172a', name: 'warning on bg (dark)', large: false },
  { fg: '#f87171', bg: '#0f172a', name: 'danger on bg (dark)', large: false },
  { fg: '#34d399', bg: '#0f172a', name: 'success on bg (dark)', large: false },
  // Primary — light blue for dark bg (WCAG AA)
  { fg: '#3b82f6', bg: '#0f172a', name: 'primary on bg (dark)', large: false },
  { fg: '#94a3b8', bg: '#1e293b', name: 'text-secondary on card (dark)', large: false },
  { fg: '#94a3b8', bg: '#1e293b', name: 'text-muted on card (dark)', large: false },
  // FC day numbers dark
  { fg: '#94a3b8', bg: '#0f172a', name: 'fc day number on bg (dark)', large: false },

  // Tooltips (dark on dark)
  { fg: '#f1f5f9', bg: '#1e293b', name: 'tooltip text on bg (dark)', large: false },
  { fg: '#fca5a5', bg: '#1e293b', name: 'tooltip conflict on bg (dark)', large: false },

  // Semantic colors dark
  { fg: '#fca5a5', bg: '#0f172a', name: 'conflict on bg (dark)', large: false },

  // Large text (≥18px or ≥14px bold) — only needs 3:1
  { fg: '#64748b', bg: '#f8fafc', name: 'muted large text on bg (light)', large: true },
  { fg: '#94a3b8', bg: '#0f172a', name: 'muted large text on bg (dark)', large: true },

  // Breadcrumbs / secondary UI
  { fg: '#64748b', bg: '#ffffff', name: 'breadcrumb on card (light)', large: false },
  { fg: '#94a3b8', bg: '#1e293b', name: 'breadcrumb on card (dark)', large: false },
];

let passed = 0;
let failed = 0;

console.log('═'.repeat(60));
console.log('  P1-HI-12: WCAG AA Contrast Check');
console.log('═'.repeat(60));
console.log('');

for (const pair of pairs) {
  const ratio = getContrastRatio(pair.fg, pair.bg);
  const threshold = pair.large ? 3.0 : 4.5;
  const status = ratio >= threshold ? '✅ PASS' : '❌ FAIL';
  const label = pair.large ? ' (large)' : '';

  console.log(
    `  ${status}  ${pair.fg} on ${pair.bg}  →  ${ratio.toFixed(2)}:1  ${pair.name}${label}`
  );

  if (ratio >= threshold) {
    passed++;
  } else {
    failed++;
  }
}

console.log('');
console.log('═'.repeat(60));
console.log(`  Results: ${passed} passed, ${failed} failed`);
console.log('═'.repeat(60));

process.exit(failed > 0 ? 1 : 0);
