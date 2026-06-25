import React, { useState, useRef, useEffect, useCallback, useId } from 'react';

// ═══════════════════════════════════════════════════════════════════════
// Dropdown Component
// Keyboard-navigable dropdown menu.
// Navigation: ArrowUp/Down, Enter/Space to select, Escape to close.
// Click outside to close. Supports dark mode.
// ═══════════════════════════════════════════════════════════════════════

export interface DropdownItem {
  id: string;
  label: string;
  icon?: React.ReactNode;
  disabled?: boolean;
  divider?: boolean;
  danger?: boolean;
}

type DropdownPlacement = 'bottom-start' | 'bottom-end' | 'top-start' | 'top-end';

interface DropdownProps {
  /** Menu items */
  items: DropdownItem[];
  /** Selection handler */
  onSelect: (item: DropdownItem) => void;
  /** Trigger element */
  anchor: React.ReactNode;
  /** Menu placement relative to anchor. Default: 'bottom-start' */
  placement?: DropdownPlacement;
  /** Additional CSS classes */
  className?: string;
  /** Disable the dropdown */
  disabled?: boolean;
}

const placementStyles: Record<DropdownPlacement, string> = {
  'bottom-start': 'top-full left-0 mt-1',
  'bottom-end': 'top-full right-0 mt-1',
  'top-start': 'bottom-full left-0 mb-1',
  'top-end': 'bottom-full right-0 mb-1',
};

export function Dropdown({
  items,
  onSelect,
  anchor,
  placement = 'bottom-start',
  className = '',
  disabled = false,
}: DropdownProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [focusIndex, setFocusIndex] = useState(-1);
  const containerRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const menuId = useId();

  const enabledItems = items.filter((item) => !item.disabled && !item.divider);

  const close = useCallback(() => {
    setIsOpen(false);
    setFocusIndex(-1);
  }, []);

  const toggle = useCallback(() => {
    if (!disabled) {
      setIsOpen((prev) => !prev);
      setFocusIndex(-1);
    }
  }, [disabled]);

  // Click outside
  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        close();
      }
    };

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        close();
        // Return focus to trigger
        const trigger = containerRef.current?.querySelector('[data-dropdown-trigger]');
        if (trigger instanceof HTMLElement) trigger.focus();
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEscape);

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [isOpen, close]);

  // Focus management when menu opens
  useEffect(() => {
    if (isOpen && enabledItems.length > 0) {
      setFocusIndex(0);
    }
  }, [isOpen, enabledItems.length]);

  // Focus the active item
  useEffect(() => {
    if (isOpen && focusIndex >= 0 && menuRef.current) {
      const items_ = menuRef.current.querySelectorAll<HTMLElement>('[role="menuitem"]');
      items_[focusIndex]?.focus();
    }
  }, [isOpen, focusIndex]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (!isOpen) return;

      switch (e.key) {
        case 'ArrowDown': {
          e.preventDefault();
          setFocusIndex((prev) => (prev < enabledItems.length - 1 ? prev + 1 : 0));
          break;
        }
        case 'ArrowUp': {
          e.preventDefault();
          setFocusIndex((prev) => (prev > 0 ? prev - 1 : enabledItems.length - 1));
          break;
        }
        case 'Enter':
        case ' ': {
          e.preventDefault();
          const selected = enabledItems[focusIndex];
          if (selected) {
            onSelect(selected);
            close();
          }
          break;
        }
        case 'Escape': {
          e.preventDefault();
          close();
          const trigger = containerRef.current?.querySelector('[data-dropdown-trigger]');
          if (trigger instanceof HTMLElement) trigger.focus();
          break;
        }
      }
    },
    [isOpen, focusIndex, enabledItems, onSelect, close],
  );

  return (
    <div ref={containerRef} className={`relative inline-flex ${className}`}>
      {/* Trigger */}
      <div
        data-dropdown-trigger
        role="button"
        tabIndex={disabled ? -1 : 0}
        aria-haspopup="true"
        aria-expanded={isOpen}
        aria-controls={menuId}
        aria-disabled={disabled}
        onClick={toggle}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown') {
            e.preventDefault();
            toggle();
          }
        }}
        className={`${disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'}`}
      >
        {anchor}
      </div>

      {/* Menu */}
      {isOpen && (
        <div
          ref={menuRef}
          id={menuId}
          role="menu"
          aria-label="Dropdown menu"
          className={`
            absolute z-50 min-w-[180px]
            bg-white dark:bg-slate-800
            border border-slate-200 dark:border-slate-700
            rounded-lg shadow-lg
            py-1
            animate-fadeIn
            ${placementStyles[placement]}
          `}
          onKeyDown={handleKeyDown}
        >
          {items.map((item, idx) => {
            if (item.divider) {
              return (
                <div
                  key={item.id}
                  className="my-1 border-t border-slate-200 dark:border-slate-700"
                  role="separator"
                  aria-orientation="horizontal"
                />
              );
            }

            const enabledIdx = enabledItems.indexOf(item);
            const isFocused = enabledIdx === focusIndex;

            return (
              <button
                key={item.id}
                role="menuitem"
                tabIndex={isFocused ? 0 : -1}
                disabled={item.disabled}
                onClick={() => {
                  if (!item.disabled) {
                    onSelect(item);
                    close();
                  }
                }}
                onMouseEnter={() => setFocusIndex(enabledIdx)}
                className={`
                  w-full flex items-center gap-2 px-3 py-2 text-sm text-left
                  transition-colors
                  ${item.disabled ? 'opacity-40 cursor-not-allowed' : ''}
                  ${item.danger ? 'text-red-600 dark:text-red-400' : 'text-slate-700 dark:text-slate-300'}
                  ${isFocused && !item.disabled ? 'bg-slate-100 dark:bg-slate-700/50' : ''}
                  ${!item.disabled && !isFocused ? 'hover:bg-slate-100 dark:hover:bg-slate-700/50' : ''}
                `}
              >
                {item.icon && (
                  <span className="w-4 h-4 flex-shrink-0 text-slate-400 dark:text-slate-500">
                    {item.icon}
                  </span>
                )}
                <span className="flex-1 truncate">{item.label}</span>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
