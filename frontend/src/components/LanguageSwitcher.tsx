import { useTranslation } from 'react-i18next';

export function LanguageSwitcher() {
  const { i18n } = useTranslation();
  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng);
    localStorage.setItem('language', lng);
  };
  return (
    <div className="flex gap-2">
      <button onClick={() => changeLanguage('en')} className="px-2 py-1 text-xs font-medium rounded bg-slate-200 dark:bg-slate-700">EN</button>
      <button onClick={() => changeLanguage('ru')} className="px-2 py-1 text-xs font-medium rounded bg-slate-200 dark:bg-slate-700">RU</button>
    </div>
  );
}