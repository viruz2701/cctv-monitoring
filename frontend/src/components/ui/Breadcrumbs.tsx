import React, { useMemo } from 'react';
import { ChevronRight, type LucideIcon } from 'lucide-react';

// ═══════════════════════════════════════════════════════════════════════
// Breadcrumbs Component
// Responsive breadcrumb navigation with icon support.
// On mobile — only the last item is shown.
// Uses ChevronRight separator by default.
// ═══════════════════════════════════════════════════════════════════════

interface BreadcrumbItem {
  label: string;
  href?: string;
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

export function Breadcrumbs({
  items,
  separator,
  className = '',
  maxItems = 6,
}: BreadcrumbsProps) {
  // Responsive: показать на мобильных только последний элемент
  const displayItems = useMemo(() => {
    if (items.length <= maxItems) return items;
    const collapsed: BreadcrumbItem[] = [
      items[0],
      { label: '...', icon: undefined },
      ...items.slice(-(maxItems - 2)),
    ];
    return collapsed;
  }, [items, maxItems]);

  const defaultSeparator = separator ?? <ChevronRight size={14} className="text-slate-400 dark:text-slate-500 flex-shrink-0" aria-hidden="true" />;

  return (
    <nav aria-label="Breadcrumb" className={className}>
      {/* Desktop: полная цепочка */}
      <ol className="hidden sm:flex items-center gap-1.5 flex-wrap">
        {displayItems.map((item, idx) => {
          const isLast = idx === displayItems.length - 1;
          const Icon = item.icon;
          return (
            <li key={`${item.label}-${idx}`} className="flex items-center gap-1.5">
              {idx > 0 && defaultSeparator}
              {item.href && !isLast ? (
                <a
                  href={item.href}
                  className="inline-flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-blue-600 dark:text-slate-400 dark:hover:text-blue-400 transition-colors"
                >
                  {Icon && <Icon size={12} aria-hidden="true" />}
                  {item.label}
                </a>
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
                  {item.label}
                </span>
              )}
            </li>
          );
        })}
      </ol>

      {/* Mobile: только последний элемент */}
      {(() => {
        const lastItem = items[items.length - 1];
        if (!lastItem) return null;
        const LastIcon = lastItem.icon;
        return (
          <ol className="sm:hidden flex items-center">
            <li>
              <span className="inline-flex items-center gap-1 text-sm font-medium text-slate-900 dark:text-white" aria-current="page">
                {LastIcon && <LastIcon size={14} aria-hidden="true" />}
                {lastItem.label}
              </span>
            </li>
          </ol>
        );
      })()}
    </nav>
  );
}
