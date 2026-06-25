import React, { useState, useCallback } from 'react';
import { Search, X, Filter, ChevronDown, ChevronUp, Save, Clock } from 'lucide-react';
import { Badge, Button } from './index';

// ── Types ────────────────────────────────────────────────────────────

export interface SearchFacet {
  key: string;
  label: string;
  options: { value: string; label: string; count?: number }[];
}

export interface SearchFilters {
  query: string;
  facets: Record<string, string[]>;
  dateFrom?: string;
  dateTo?: string;
}

export interface SavedSearch {
  id: string;
  name: string;
  filters: SearchFilters;
}

interface AdvancedSearchProps {
  /** Placeholder для строки поиска */
  placeholder?: string;
  /** Доступные фасеты для фильтрации */
  facets?: SearchFacet[];
  /** Текущие фильтры */
  filters: SearchFilters;
  /** Колбэк при изменении фильтров */
  onFiltersChange: (filters: SearchFilters) => void;
  /** Колбэк поиска */
  onSearch: () => void;
  /** Сохранённые поиски */
  savedSearches?: SavedSearch[];
  /** Загрузка */
  loading?: boolean;
}

// ── Component ────────────────────────────────────────────────────────

export function AdvancedSearch({
  placeholder = 'Поиск...',
  facets = [],
  filters,
  onFiltersChange,
  onSearch,
  savedSearches = [],
  loading = false,
}: AdvancedSearchProps) {
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showSaved, setShowSaved] = useState(false);

  const activeFacetCount = Object.values(filters.facets).reduce((sum, arr) => sum + arr.length, 0);
  const hasActiveFilters = filters.query !== '' || activeFacetCount > 0 || !!filters.dateFrom || !!filters.dateTo;

  const updateQuery = (query: string) => onFiltersChange({ ...filters, query });

  const toggleFacet = (facetKey: string, value: string) => {
    const current = filters.facets[facetKey] || [];
    const updated = current.includes(value)
      ? current.filter(v => v !== value)
      : [...current, value];
    onFiltersChange({
      ...filters,
      facets: { ...filters.facets, [facetKey]: updated },
    });
  };

  const clearAll = () => {
    onFiltersChange({ query: '', facets: {}, dateFrom: undefined, dateTo: undefined });
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') onSearch();
  };

  return (
    <div className="space-y-2">
      {/* Search Bar */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
        <input
          type="text"
          className={`w-full pl-9 pr-20 py-2 rounded-lg border text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 ${
            filters.query ? 'border-blue-400 bg-blue-50' : 'border-slate-300'
          }`}
          placeholder={placeholder}
          value={filters.query}
          onChange={e => updateQuery(e.target.value)}
          onKeyDown={handleKeyDown}
        />

        <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1">
          {/* Saved searches */}
          {savedSearches.length > 0 && (
            <div className="relative">
              <button
                onClick={() => setShowSaved(!showSaved)}
                className="p-1.5 text-slate-400 hover:text-slate-600 rounded"
                title="Saved searches"
              >
                <Clock className="w-4 h-4" />
              </button>
              {showSaved && (
                <div className="absolute right-0 top-full mt-1 bg-white border border-slate-200 rounded-lg shadow-lg p-1 z-50 min-w-[200px]">
                  {savedSearches.map(s => (
                    <button
                      key={s.id}
                      onClick={() => { onFiltersChange(s.filters); setShowSaved(false); onSearch(); }}
                      className="w-full text-left px-3 py-2 text-sm text-slate-700 hover:bg-slate-50 rounded"
                    >
                      {s.name}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Advanced toggle */}
          {facets.length > 0 && (
            <button
              onClick={() => setShowAdvanced(!showAdvanced)}
              className={`p-1.5 rounded ${activeFacetCount > 0 ? 'text-blue-600 bg-blue-50' : 'text-slate-400 hover:text-slate-600'}`}
              title="Advanced filters"
            >
              <Filter className="w-4 h-4" />
            </button>
          )}

          {/* Clear */}
          {hasActiveFilters && (
            <button onClick={clearAll} className="p-1.5 text-slate-400 hover:text-red-500 rounded" title="Clear">
              <X className="w-4 h-4" />
            </button>
          )}

          {/* Search button */}
          <Button size="sm" onClick={onSearch} loading={loading}>
            {t('search') || 'Search'}
          </Button>
        </div>
      </div>

      {/* Active filter badges */}
      {hasActiveFilters && (
        <div className="flex flex-wrap gap-1.5">
          {filters.query && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700">
              <Search className="w-3 h-3" /> "{filters.query}"
              <button onClick={() => updateQuery('')}><X className="w-3 h-3" /></button>
            </span>
          )}
          {Object.entries(filters.facets).map(([key, values]) =>
            values.map(v => (
              <span key={`${key}-${v}`} className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-indigo-50 text-indigo-700">
                {v}
                <button onClick={() => toggleFacet(key, v)}><X className="w-3 h-3" /></button>
              </span>
            ))
          )}
          {(filters.dateFrom || filters.dateTo) && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-50 text-amber-700">
              {filters.dateFrom || '...'} — {filters.dateTo || '...'}
              <button onClick={() => onFiltersChange({ ...filters, dateFrom: undefined, dateTo: undefined })}>
                <X className="w-3 h-3" />
              </button>
            </span>
          )}
        </div>
      )}

      {/* Advanced filters panel */}
      {showAdvanced && facets.length > 0 && (
        <div className="p-4 bg-white border border-slate-200 rounded-lg shadow-sm">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {facets.map(facet => (
              <div key={facet.key}>
                <h4 className="text-xs font-semibold text-slate-500 uppercase mb-2">{facet.label}</h4>
                <div className="space-y-1 max-h-40 overflow-y-auto">
                  {facet.options.map(opt => {
                    const isSelected = (filters.facets[facet.key] || []).includes(opt.value);
                    return (
                      <label
                        key={opt.value}
                        className={`flex items-center gap-2 px-2 py-1 rounded cursor-pointer transition-colors ${
                          isSelected ? 'bg-blue-50' : 'hover:bg-slate-50'
                        }`}
                      >
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => toggleFacet(facet.key, opt.value)}
                          className="rounded border-slate-300 text-blue-600"
                        />
                        <span className="text-sm text-slate-700 flex-1">{opt.label}</span>
                        {opt.count !== undefined && (
                          <Badge variant="info" size="sm">{opt.count}</Badge>
                        )}
                      </label>
                    );
                  })}
                </div>
              </div>
            ))}

            {/* Date range */}
            <div>
              <h4 className="text-xs font-semibold text-slate-500 uppercase mb-2">Date Range</h4>
              <div className="space-y-2">
                <input
                  type="date"
                  className="w-full rounded-lg border border-slate-300 px-2 py-1.5 text-xs"
                  value={filters.dateFrom || ''}
                  onChange={e => onFiltersChange({ ...filters, dateFrom: e.target.value || undefined })}
                  placeholder="From"
                />
                <input
                  type="date"
                  className="w-full rounded-lg border border-slate-300 px-2 py-1.5 text-xs"
                  value={filters.dateTo || ''}
                  onChange={e => onFiltersChange({ ...filters, dateTo: e.target.value || undefined })}
                  placeholder="To"
                />
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function t(key: string): string {
  const dict: Record<string, string> = {
    'search': 'Поиск',
  };
  return dict[key] || key;
}
