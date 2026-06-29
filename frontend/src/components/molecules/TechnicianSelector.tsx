import React, { useState, useRef, useEffect, useCallback, useMemo, useId } from 'react';
import { Check, ChevronsUpDown, Search, Users } from '../ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// TechnicianSelector — Combobox with avatars
// Показывает имя, роль, workload. Group by team.
// Keyboard accessible, dark mode.
// ═══════════════════════════════════════════════════════════════════════

export interface Technician {
  id: string;
  name: string;
  role: string;
  team: string;
  avatarUrl?: string;
  workload: number; // 0–100, текущая загрузка
  /** Hex color for initials fallback */
  avatarColor?: string;
}

interface TechnicianSelectorProps {
  technicians: Technician[];
  selectedIds: string[];
  onChange: (ids: string[]) => void;
  placeholder?: string;
  multi?: boolean;
  className?: string;
}

function getInitials(name: string): string {
  return name
    .split(' ')
    .map((w) => w[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);
}

function getWorkloadColor(workload: number): string {
  if (workload >= 90) return 'bg-red-500';
  if (workload >= 70) return 'bg-amber-500';
  return 'bg-emerald-500';
}

// ── Avatar ──────────────────────────────────────────────────────────────

function TechnicianAvatar({ tech, size = 'sm' }: { tech: Technician; size?: 'sm' | 'md' }) {
  const sizeClass = size === 'sm' ? 'w-7 h-7 text-[10px]' : 'w-9 h-9 text-xs';
  const dotSize = size === 'sm' ? 'w-1.5 h-1.5' : 'w-2 h-2';

  return (
    <div className="relative flex-shrink-0">
      {tech.avatarUrl ? (
        <img
          src={tech.avatarUrl}
          alt={tech.name}
          className={`${sizeClass} rounded-full object-cover`}
        />
      ) : (
        <div
          className={`${sizeClass} rounded-full flex items-center justify-center font-semibold text-white`}
          style={{ backgroundColor: tech.avatarColor ?? '#6366f1' }}
        >
          {getInitials(tech.name)}
        </div>
      )}
      {/* Workload dot */}
      <span
        className={`absolute -bottom-0.5 -right-0.5 ${dotSize} rounded-full ring-2 ring-white dark:ring-slate-800 ${getWorkloadColor(tech.workload)}`}
        title={`Workload: ${tech.workload}%`}
      />
    </div>
  );
}

// ── Main Component ──────────────────────────────────────────────────────

export function TechnicianSelector({
  technicians,
  selectedIds,
  onChange,
  placeholder = 'Выберите техника...',
  multi = true,
  className = '',
}: TechnicianSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [focusIdx, setFocusIdx] = useState(-1);
  const containerRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const listboxId = useId();

  // Group by team
  const grouped = useMemo(() => {
    const filtered = technicians.filter(
      (t) => t.name.toLowerCase().includes(search.toLowerCase()) || t.role.toLowerCase().includes(search.toLowerCase()),
    );
    const groups: Record<string, Technician[]> = {};
    for (const t of filtered) {
      if (!groups[t.team]) groups[t.team] = [];
      groups[t.team].push(t);
    }
    return groups;
  }, [technicians, search]);

  const flatItems = useMemo(() => {
    const items: Array<{ type: 'group' | 'tech'; label?: string; tech?: Technician }> = [];
    for (const [team, techs] of Object.entries(grouped)) {
      items.push({ type: 'group', label: team });
      for (const t of techs) items.push({ type: 'tech', tech: t });
    }
    return items;
  }, [grouped]);

  const isSelected = useCallback(
    (id: string) => selectedIds.includes(id),
    [selectedIds],
  );

  const toggleSelection = useCallback(
    (id: string) => {
      if (multi) {
        const next = isSelected(id) ? selectedIds.filter((s) => s !== id) : [...selectedIds, id];
        onChange(next);
      } else {
        onChange(isSelected(id) ? [] : [id]);
        setIsOpen(false);
      }
    },
    [multi, selectedIds, onChange, isSelected],
  );

  // Click outside
  useEffect(() => {
    if (!isOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
        setSearch('');
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [isOpen]);

  // Focus search on open
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => searchRef.current?.focus(), 50);
    }
  }, [isOpen]);

  // Scroll focused item into view
  useEffect(() => {
    if (focusIdx < 0 || !listRef.current) return;
    const items = listRef.current.querySelectorAll<HTMLElement>('[role="option"]');
    items[focusIdx]?.scrollIntoView({ block: 'nearest' });
  }, [focusIdx]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (!isOpen) return;

      const techItems = flatItems.filter((f) => f.type === 'tech');

      switch (e.key) {
        case 'ArrowDown': {
          e.preventDefault();
          setFocusIdx((prev) => (prev < techItems.length - 1 ? prev + 1 : 0));
          break;
        }
        case 'ArrowUp': {
          e.preventDefault();
          setFocusIdx((prev) => (prev > 0 ? prev - 1 : techItems.length - 1));
          break;
        }
        case 'Enter': {
          e.preventDefault();
          const item = techItems[focusIdx];
          if (item?.tech) toggleSelection(item.tech.id);
          break;
        }
        case 'Escape': {
          e.preventDefault();
          setIsOpen(false);
          setSearch('');
          break;
        }
      }
    },
    [isOpen, flatItems, focusIdx, toggleSelection],
  );

  const selectedTechs = technicians.filter((t) => selectedIds.includes(t.id));

  return (
    <div ref={containerRef} className={`relative ${className}`}>
      {/* Trigger */}
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        aria-controls={listboxId}
        onClick={() => setIsOpen((p) => !p)}
        className="
          w-full flex items-center gap-2 px-3 py-2 text-sm
          bg-white dark:bg-slate-800
          border border-slate-300 dark:border-slate-600
          rounded-lg shadow-sm
          hover:border-slate-400 dark:hover:border-slate-500
          focus:outline-none focus:ring-2 focus:ring-blue-500
          transition-colors text-left
        "
      >
        {selectedTechs.length > 0 ? (
          <div className="flex-1 flex items-center gap-1.5 flex-wrap">
            {selectedTechs.map((tech) => (
              <span
                key={tech.id}
                className="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded-full text-xs font-medium"
              >
                <TechnicianAvatar tech={tech} size="sm" />
                {tech.name}
              </span>
            ))}
          </div>
        ) : (
          <span className="flex-1 text-slate-400 dark:text-slate-500">{placeholder}</span>
        )}
        <ChevronsUpDown size={16} className="text-slate-400 flex-shrink-0" />
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div
          className="
            absolute z-50 mt-1 w-full
            bg-white dark:bg-slate-800
            border border-slate-200 dark:border-slate-700
            rounded-lg shadow-lg
            overflow-hidden
            animate-fadeIn
          "
          onKeyDown={handleKeyDown}
        >
          {/* Search */}
          <div className="p-2 border-b border-slate-200 dark:border-slate-700">
            <div className="relative">
              <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                ref={searchRef}
                type="text"
                value={search}
                onChange={(e) => { setSearch(e.target.value); setFocusIdx(-1); }}
                placeholder="Поиск..."
                className="w-full pl-8 pr-3 py-1.5 text-sm bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900 dark:text-white placeholder:text-slate-400"
              />
            </div>
          </div>

          {/* List */}
          {flatItems.length === 0 ? (
            <div className="p-4 text-center text-sm text-slate-400 dark:text-slate-500">
              Ничего не найдено
            </div>
          ) : (
            <div ref={listRef} id={listboxId} role="listbox" className="max-h-60 overflow-y-auto" aria-multiselectable={multi}>
              {flatItems.map((item) => {
                if (item.type === 'group') {
                  return (
                    <div
                      key={item.label}
                      className="flex items-center gap-1.5 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500 bg-slate-50 dark:bg-slate-900/50"
                    >
                      <Users size={12} />
                      {item.label}
                    </div>
                  );
                }

                if (!item.tech) return null;
                const tech = item.tech;
                const selected = isSelected(tech.id);

                return (
                  <div
                    key={tech.id}
                    role="option"
                    aria-selected={selected}
                    onClick={() => toggleSelection(tech.id)}
                    className={`
                      flex items-center gap-3 px-3 py-2 cursor-pointer transition-colors
                      ${selected ? 'bg-blue-50 dark:bg-blue-900/20' : 'hover:bg-slate-50 dark:hover:bg-slate-700/30'}
                    `}
                  >
                    <TechnicianAvatar tech={tech} size="md" />

                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium text-slate-900 dark:text-white truncate">
                        {tech.name}
                      </div>
                      <div className="text-xs text-slate-500 dark:text-slate-400 truncate">
                        {tech.role}
                      </div>
                    </div>

                    {/* Workload badge */}
                    <div className="flex items-center gap-1.5">
                      <div className="flex items-center gap-1">
                        <span className={`w-1.5 h-1.5 rounded-full ${getWorkloadColor(tech.workload)}`} />
                        <span className="text-xs tabular-nums text-slate-500 dark:text-slate-400">
                          {tech.workload}%
                        </span>
                      </div>
                      {multi && selected && (
                        <Check size={14} className="text-blue-600 dark:text-blue-400 flex-shrink-0" />
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
