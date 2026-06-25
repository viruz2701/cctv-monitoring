import { useState, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';

interface Language {
  code: string;
  label: string;
  flag: string;
}

type LanguageGroup = {
  group: string;
  languages: Language[];
};

const LANGUAGES: LanguageGroup[] = [
  {
    group: 'CIS',
    languages: [
      { code: 'ru', label: 'Русский', flag: '🇷🇺' },
      { code: 'be', label: 'Беларуская', flag: '🇧🇾' },
      { code: 'uk', label: 'Українська', flag: '🇺🇦' },
    ],
  },
  {
    group: 'European',
    languages: [
      { code: 'en', label: 'English', flag: '🇬🇧' },
      { code: 'de', label: 'Deutsch', flag: '🇩🇪' },
      { code: 'fr', label: 'Français', flag: '🇫🇷' },
      { code: 'es', label: 'Español', flag: '🇪🇸' },
      { code: 'pt', label: 'Português', flag: '🇵🇹' },
      { code: 'it', label: 'Italiano', flag: '🇮🇹' },
      { code: 'pl', label: 'Polski', flag: '🇵🇱' },
    ],
  },
  {
    group: 'Asian',
    languages: [
      { code: 'zh', label: '中文', flag: '🇨🇳' },
      { code: 'ja', label: '日本語', flag: '🇯🇵' },
      { code: 'ko', label: '한국어', flag: '🇰🇷' },
      { code: 'tr', label: 'Türkçe', flag: '🇹🇷' },
    ],
  },
  {
    group: 'Other',
    languages: [
      { code: 'ar', label: 'العربية', flag: '🇸🇦' },
    ],
  },
];

function getLanguageLabel(code: string): string {
  for (const group of LANGUAGES) {
    const lang = group.languages.find((l) => l.code === code);
    if (lang) return `${lang.flag} ${lang.label}`;
  }
  return code.toUpperCase();
}

export function LanguageSwitcher() {
  const { i18n } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState('');
  const dropdownRef = useRef<HTMLDivElement>(null);

  const currentLabel = getLanguageLabel(i18n.language);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setSearch('');
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng);
    localStorage.setItem('language', lng);
    setIsOpen(false);
    setSearch('');
  };

  const filteredGroups = LANGUAGES.map((group) => ({
    ...group,
    languages: search
      ? group.languages.filter(
          (lang) =>
            lang.label.toLowerCase().includes(search.toLowerCase()) ||
            lang.code.toLowerCase().includes(search.toLowerCase())
        )
      : group.languages,
  })).filter((group) => group.languages.length > 0);

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-1.5 text-xs font-medium rounded-lg bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600 transition-colors"
        aria-haspopup="listbox"
        aria-expanded={isOpen}
      >
        <span>{currentLabel}</span>
        <svg
          className={`w-3 h-3 transition-transform ${isOpen ? 'rotate-180' : ''}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-2 w-56 bg-white dark:bg-slate-800 rounded-xl shadow-lg border border-slate-200 dark:border-slate-700 z-50">
          {/* Search */}
          <div className="p-2 border-b border-slate-200 dark:border-slate-700">
            <div className="relative">
              <svg
                className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-400"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search language..."
                className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-slate-100 dark:bg-slate-700 border border-slate-200 dark:border-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400"
                autoFocus
              />
            </div>
          </div>

          {/* Language list */}
          <div className="max-h-64 overflow-y-auto p-1">
            {filteredGroups.map((group) => (
              <div key={group.group}>
                <div className="px-2 py-1 mt-1 text-[10px] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">
                  {group.group}
                </div>
                {group.languages.map((lang) => (
                  <button
                    key={lang.code}
                    type="button"
                    onClick={() => changeLanguage(lang.code)}
                    className={`w-full flex items-center gap-3 px-2 py-1.5 text-sm rounded-lg transition-colors ${
                      i18n.language === lang.code
                        ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                        : 'hover:bg-slate-100 dark:hover:bg-slate-700/50 text-slate-700 dark:text-slate-300'
                    }`}
                    role="option"
                    aria-selected={i18n.language === lang.code}
                  >
                    <span className="text-base leading-none">{lang.flag}</span>
                    <span className="text-xs">{lang.label}</span>
                    {i18n.language === lang.code && (
                      <svg className="ml-auto w-3.5 h-3.5 text-blue-500" fill="currentColor" viewBox="0 0 20 20">
                        <path
                          fillRule="evenodd"
                          d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                          clipRule="evenodd"
                        />
                      </svg>
                    )}
                  </button>
                ))}
              </div>
            ))}
            {filteredGroups.length === 0 && (
              <div className="px-2 py-4 text-xs text-center text-slate-400">
                No languages found
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
