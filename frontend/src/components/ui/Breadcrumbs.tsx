import React, { useMemo } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { ChevronRight, type LucideIcon } from 'lucide-react';

// ═══════════════════════════════════════════════════════════════════════
// Breadcrumbs Component
// Responsive breadcrumb navigation with icon support.
// On mobile — only the last item is shown (with back link if previous exists).
// Uses ChevronRight separator by default.
// Supports i18n translation for labels via t().
// ═══════════════════════════════════════════════════════════════════════

export interface BreadcrumbItem {
  /** Label text — can be a plain string OR an i18n key (auto-detected) */
  label: string;
  /** Optional href for clickable crumbs. Last item is always plain text. */
  href?: string;
  /** Optional icon */
  icon?: LucideIcon;
}

interface BreadcrumbsProps {
  items: BreadcrumbItem[];
  /** Custom separator element. Default: ChevronRight icon */
  separator?: React.ReactNode;
  /** Custom class name */
  className?: string;
  /** Max items before truncation (desktop). Default: 6 */
  maxItems?: number;
}

/**
 * Resolve label: if label matches a known i18n key, translate it;
 * otherwise use the label as-is.
 */
function resolveLabel(label: string, t: (key: string) => string): string {
  // Check if the label is an i18n key by trying translation
  const translated = t(label);
  // If translation returns the key unchanged, it's not a valid key → use raw label
  return translated !== label ? translated : label;
}

export function Breadcrumbs({
  items,
  separator,
  className = '',
  maxItems = 6,
}: BreadcrumbsProps) {
  const { t } = useTranslation();

  // Resolve labels with i18n
  const resolvedItems = useMemo<BreadcrumbItem[]>(
    () =>
      items.map((item) => ({
        ...item,
        label: resolveLabel(item.label, t),
      })),
    [items, t],
  );

  // Truncate long chains on desktop
  const displayItems = useMemo(() => {
    if (resolvedItems.length <= maxItems) return resolvedItems;
    return [
      resolvedItems[0],
      { label: '…' },
      ...resolvedItems.slice(-(maxItems - 2)),
    ];
  }, [resolvedItems, maxItems]);

  const defaultSeparator =
    separator ?? (
      <ChevronRight
        size={14}
        className="text-slate-400 dark:text-slate-500 flex-shrink-0"
        aria-hidden="true"
      />
    );

  return (
    <nav aria-label={t('breadcrumb')} className={className}>
      {/* ── Desktop: full chain ── */}
      <ol className="hidden sm:flex items-center gap-1.5 flex-wrap">
        {displayItems.map((item, idx) => {
          const isLast = idx === displayItems.length - 1;
          const Icon = item.icon;
          return (
            <li
              key={`${item.label}-${idx}`}
              className="flex items-center gap-1.5"
            >
              {idx > 0 && defaultSeparator}
              {item.href && !isLast ? (
                <Link
                  to={item.href}
                  className="inline-flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-blue-600 dark:text-slate-400 dark:hover:text-blue-400 transition-colors rounded focus-visible:outline-2 focus-visible:outline-blue-500"
                >
                  {Icon && <Icon size={12} aria-hidden="true" />}
                  <span className="truncate max-w-[160px]">{item.label}</span>
                </Link>
              ) : (
                <span
                  className={`inline-flex items-center gap-1 text-xs font-medium ${
                    isLast
                      ? 'text-slate-900 dark:text-white'
                      : 'text-slate-500 dark:text-slate-400'
                  }`}
                  aria-current={isLast ? 'page' : undefined}
                >
                  {Icon && <Icon size={12} aria-hidden="true" />}
                  <span className="truncate max-w-[200px]">{item.label}</span>
                </span>
              )}
            </li>
          );
        })}
      </ol>

      {/* ── Mobile: last item + back link ── */}
      {(() => {
        const lastItem = resolvedItems[resolvedItems.length - 1];
        const prevItem = resolvedItems.length > 1
          ? resolvedItems[resolvedItems.length - 2]
          : null;
        if (!lastItem) return null;
        const LastIcon = lastItem.icon;
        return (
          <ol className="sm:hidden flex items-center gap-1.5">
            {prevItem?.href && (
              <>
                <li>
                  <Link
                    to={prevItem.href}
                    className="inline-flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-blue-600 dark:text-slate-400 dark:hover:text-blue-400 transition-colors rounded focus-visible:outline-2 focus-visible:outline-blue-500"
                    aria-label={t('back')}
                  >
                    <ChevronRight
                      size={14}
                      className="rotate-180 flex-shrink-0"
                      aria-hidden="true"
                    />
                    <span className="truncate max-w-[100px]">
                      {prevItem.label}
                    </span>
                  </Link>
                </li>
                <li aria-hidden="true">
                  <ChevronRight
                    size={14}
                    className="text-slate-400 dark:text-slate-500 flex-shrink-0"
                  />
                </li>
              </>
            )}
            <li>
              <span
                className="inline-flex items-center gap-1 text-sm font-medium text-slate-900 dark:text-white"
                aria-current="page"
              >
                {LastIcon && <LastIcon size={14} aria-hidden="true" />}
                <span className="truncate max-w-[180px]">
                  {lastItem.label}
                </span>
              </span>
            </li>
          </ol>
        );
      })()}
    </nav>
  );
}
