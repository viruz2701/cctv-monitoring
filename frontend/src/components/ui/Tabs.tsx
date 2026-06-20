import React, { useState } from 'react';

interface Tab {
  id: string;
  label: string;
  icon?: React.ReactNode;
  badge?: number | string;
  disabled?: boolean;
}

interface TabsProps {
  tabs: Tab[];
  activeTab?: string;
  onChange: (tabId: string) => void;
  children: React.ReactNode;
  variant?: 'default' | 'pills' | 'underline';
  className?: string;
}

export function Tabs({
  tabs,
  activeTab,
  onChange,
  children,
  variant = 'default',
  className = '',
}: TabsProps) {
  const [internalActive, setInternalActive] = useState(tabs[0]?.id || '');
  const current = activeTab ?? internalActive;

  const handleChange = (tabId: string) => {
    setInternalActive(tabId);
    onChange(tabId);
  };

  const variantStyles = {
    default: {
      container: 'border-b border-slate-200 dark:border-slate-700',
      tab: (active: boolean) =>
        `px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
          active
            ? 'border-blue-600 text-blue-600 dark:text-blue-400 dark:border-blue-400'
            : 'border-transparent text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300 hover:border-slate-300 dark:hover:border-slate-600'
        }`,
    },
    pills: {
      container: 'gap-1',
      tab: (active: boolean) =>
        `px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
          active
            ? 'bg-blue-600 text-white shadow-sm'
            : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'
        }`,
    },
    underline: {
      container: 'gap-1',
      tab: (active: boolean) =>
        `px-4 py-2.5 text-sm font-medium transition-colors relative ${
          active
            ? 'text-blue-600 dark:text-blue-400'
            : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
        }`,
    },
  };

  const styles = variantStyles[variant];

  return (
    <div className={className}>
      <div className={`flex ${variant === 'default' ? '' : 'flex-wrap'} ${styles.container}`}>
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => !tab.disabled && handleChange(tab.id)}
            disabled={tab.disabled}
            className={`${styles.tab(tab.id === current)} ${
              tab.disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'
            }`}
          >
            <span className="flex items-center gap-1.5">
              {tab.icon}
              {tab.label}
              {tab.badge != null && (
                <span className="inline-flex items-center justify-center min-w-[20px] h-5 px-1.5 text-xs font-semibold rounded-full bg-red-500 text-white">
                  {tab.badge}
                </span>
              )}
            </span>
            {variant === 'underline' && tab.id === current && (
              <span className="absolute bottom-0 left-1/2 -translate-x-1/2 w-8 h-0.5 bg-blue-600 dark:bg-blue-400 rounded-full" />
            )}
          </button>
        ))}
      </div>
      <div className="mt-4">{children}</div>
    </div>
  );
}