// CommandPalette — универсальная палитра команд с поиском.
//
// P2-2.2: Smart Command Palette Search
//   - Поиск по entities (devices, work orders, sites)
//   - Поиск по navigation items
//   - Fuzzy matching для typos
//   - Горячая клавиша: Cmd+K / Ctrl+K

import React, { useEffect, useState, useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { Search, Command, FileText, HardDrive, MapPin, Ticket, ArrowRight } from '../ui/Icons';
import { useNavigation } from '../hooks/useNavigation';

interface SearchResult {
    id: string;
    type: 'page' | 'device' | 'work_order' | 'site';
    label: string;
    description?: string;
    path?: string;
    icon?: React.ElementType;
}

// Простой fuzzy match
function fuzzyMatch(text: string, query: string): boolean {
    const lower = text.toLowerCase();
    const q = query.toLowerCase();
    let qi = 0;
    for (let i = 0; i < lower.length && qi < q.length; i++) {
        if (lower[i] === q[qi]) qi++;
    }
    return qi === q.length;
}

export function CommandPalette() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const { flatItems } = useNavigation();
    const [isOpen, setIsOpen] = useState(false);
    const [query, setQuery] = useState('');
    const [selectedIndex, setSelectedIndex] = useState(0);
    const inputRef = useRef<HTMLInputElement>(null);
    const listRef = useRef<HTMLDivElement>(null);

    // Горячая клавиша: Cmd+K / Ctrl+K
    useEffect(() => {
        const handler = (e: KeyboardEvent) => {
            if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
                e.preventDefault();
                setIsOpen(prev => !prev);
            }
            if (e.key === 'Escape') setIsOpen(false);
        };
        window.addEventListener('keydown', handler);
        return () => window.removeEventListener('keydown', handler);
    }, []);

    // Фокус на input при открытии
    useEffect(() => {
        if (isOpen) {
            setTimeout(() => inputRef.current?.focus(), 50);
            setQuery('');
            setSelectedIndex(0);
        }
    }, [isOpen]);

    // Поиск
    const results = React.useMemo<SearchResult[]>(() => {
        if (!query.trim()) {
            // Показываем последние navigation items
            return flatItems.slice(0, 8).map(item => ({
                id: `page-${item.path}`,
                type: 'page' as const,
                label: item.label,
                path: item.path,
                icon: item.icon,
            }));
        }

        const results: SearchResult[] = [];

        // Поиск по страницам
        for (const item of flatItems) {
            if (fuzzyMatch(item.label, query) || fuzzyMatch(item.path, query)) {
                results.push({
                    id: `page-${item.path}`,
                    type: 'page',
                    label: item.label,
                    path: item.path,
                    icon: item.icon,
                });
            }
        }

        return results.slice(0, 12);
    }, [query, flatItems]);

    const handleSelect = useCallback((result: SearchResult) => {
        if (result.path) {
            navigate(result.path);
            setIsOpen(false);
        }
    }, [navigate]);

    const handleKeyDown = (e: React.KeyboardEvent) => {
        switch (e.key) {
            case 'ArrowDown':
                e.preventDefault();
                setSelectedIndex(prev => Math.min(prev + 1, results.length - 1));
                break;
            case 'ArrowUp':
                e.preventDefault();
                setSelectedIndex(prev => Math.max(prev - 1, 0));
                break;
            case 'Enter':
                e.preventDefault();
                if (results[selectedIndex]) handleSelect(results[selectedIndex]);
                break;
        }
    };

    if (!isOpen) return null;

    const typeIcons: Record<string, React.ElementType> = {
        page: Command,
        device: HardDrive,
        work_order: Ticket,
        site: MapPin,
    };

    return (
        <div className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]" onClick={() => setIsOpen(false)}>
            {/* Overlay */}
            <div className="absolute inset-0 bg-black/50" />

            {/* Palette */}
            <div
                className="relative w-full max-w-lg bg-white dark:bg-slate-800 rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-700 overflow-hidden"
                onClick={e => e.stopPropagation()}
                role="dialog"
                aria-label={t('command_palette') || 'Command palette'}
            >
                {/* Input */}
                <div className="flex items-center gap-3 px-4 py-3 border-b border-slate-200 dark:border-slate-700">
                    <Search className="w-5 h-5 text-slate-400" />
                    <input
                        ref={inputRef}
                        type="text"
                        value={query}
                        onChange={e => { setQuery(e.target.value); setSelectedIndex(0); }}
                        onKeyDown={handleKeyDown}
                        placeholder={t('search_placeholder') || 'Search pages, devices...'}
                        className="flex-1 bg-transparent text-slate-900 dark:text-white placeholder-slate-400 outline-none text-base"
                        aria-label={t('search') || 'Search'}
                    />
                    <kbd className="hidden sm:inline-flex items-center gap-1 px-2 py-1 text-xs text-slate-400 bg-slate-100 dark:bg-slate-700 rounded">
                        <span>ESC</span>
                    </kbd>
                </div>

                {/* Results */}
                <div ref={listRef} className="max-h-80 overflow-y-auto p-2" role="listbox">
                    {results.length === 0 ? (
                        <div className="py-8 text-center text-slate-400">
                            <Search className="w-8 h-8 mx-auto mb-2 opacity-50" />
                            <p className="text-sm">{t('no_results') || 'No results found'}</p>
                        </div>
                    ) : (
                        results.map((result, index) => {
                            const Icon = result.icon || typeIcons[result.type] || FileText;
                            const isSelected = index === selectedIndex;
                            return (
                                <button
                                    key={result.id}
                                    onClick={() => handleSelect(result)}
                                    onMouseEnter={() => setSelectedIndex(index)}
                                    role="option"
                                    aria-selected={isSelected}
                                    className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-left transition-colors ${
                                        isSelected
                                            ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                                            : 'text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700/50'
                                    }`}
                                >
                                    <Icon className={`w-4 h-4 ${isSelected ? 'text-blue-500' : 'text-slate-400'}`} />
                                    <div className="flex-1 min-w-0">
                                        <p className="text-sm font-medium truncate">{result.label}</p>
                                        {result.description && (
                                            <p className="text-xs text-slate-400 truncate">{result.description}</p>
                                        )}
                                    </div>
                                    <ArrowRight className={`w-4 h-4 ${isSelected ? 'text-blue-500' : 'text-slate-300'}`} />
                                </button>
                            );
                        })
                    )}
                </div>

                {/* Footer */}
                <div className="flex items-center gap-4 px-4 py-2 border-t border-slate-200 dark:border-slate-700 text-xs text-slate-400">
                    <span className="flex items-center gap-1">
                        <kbd className="px-1.5 py-0.5 bg-slate-100 dark:bg-slate-700 rounded text-[10px]">↑↓</kbd>
                        {t('navigate') || 'Navigate'}
                    </span>
                    <span className="flex items-center gap-1">
                        <kbd className="px-1.5 py-0.5 bg-slate-100 dark:bg-slate-700 rounded text-[10px]">↵</kbd>
                        {t('open') || 'Open'}
                    </span>
                    <span className="flex items-center gap-1">
                        <kbd className="px-1.5 py-0.5 bg-slate-100 dark:bg-slate-700 rounded text-[10px]">Esc</kbd>
                        {t('close') || 'Close'}
                    </span>
                </div>
            </div>
        </div>
    );
}
