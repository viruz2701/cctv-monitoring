import React, { useState, useCallback, useRef, useEffect, useMemo } from 'react';
import { NavLink, Link } from 'react-router-dom';
import { useNavigation } from '../../hooks/useNavigation';
import { useTranslation } from 'react-i18next';
import { isFeatureEnabled } from '../../config/featureFlags';
import {
  ALWAYS_VISIBLE_ITEMS,
  SIDEBAR_GROUPS,
  DEFAULT_QUICK_ACCESS,
  type SidebarItem,
  type SidebarGroup,
} from '../../config/sidebarGroups';
import {
  Camera,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  Star,
  X,
} from '../ui/Icons';

interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
  mobileOpen?: boolean;
  onMobileClose?: () => void;
}

const STORAGE_KEY_COLLAPSED_GROUPS = 'sidebar_collapsed_groups_v2';

/**
 * Load collapsed groups from localStorage (progressive disclosure mode).
 */
function loadCollapsedGroups(): Record<string, boolean> {
  try {
    const saved = localStorage.getItem(STORAGE_KEY_COLLAPSED_GROUPS);
    if (saved) return JSON.parse(saved);
  } catch { /* ignore */ }
  return {};
}

/**
 * Save collapsed groups to localStorage.
 */
function saveCollapsedGroups(groups: Record<string, boolean>): void {
  try {
    localStorage.setItem(STORAGE_KEY_COLLAPSED_GROUPS, JSON.stringify(groups));
  } catch { /* ignore */ }
}

/**
 * Sidebar — grouped navigation с quick access bar.
 *
 * P0-2.1: 5 групп (Dashboard, Assets, Operations, Insights, Administration)
 * P0-2.2: Role-based filtering через useNavigation()
 * P0-2.3: Quick access bar (3-4 pinned items сверху)
 * P1-UX.4: aria-current="page" на активных ссылках + keyboard navigation
 * P3-MICRO.1: React.memo для предотвращения частых ререндеров при навигации
 *
 * UX-1.1 (feature flag sidebar_progressive_disclosure):
 *   5 доменов: Operations, Assets, Analytics, Governance, Admin
 *   14 always-visible items + 5 group accordions
 *   Collapse state в localStorage
 *
 * UX-1.6: ARIA navigation, focus-visible ring, Arrow keys, Home/End, Skip link
 */
export const Sidebar = React.memo(function Sidebar({
  collapsed,
  onToggle,
  mobileOpen,
  onMobileClose,
}: SidebarProps) {
  const { t } = useTranslation();
  const progressiveEnabled = isFeatureEnabled('sidebar_progressive_disclosure');

  // ── Legacy navigation (feature flag OFF) ──────────────────────────
  const legacyNav = useNavigation();

  // ── Progressive disclosure state ──────────────────────────────────
  const [collapsedGroups, setCollapsedGroups] = useState<Record<string, boolean>>(loadCollapsedGroups);

  const isGroupCollapsed = useCallback(
    (groupId: string): boolean => collapsedGroups[groupId] ?? true,
    [collapsedGroups],
  );

  const toggleGroupCollapsed = useCallback((groupId: string) => {
    setCollapsedGroups((prev) => {
      const next = { ...prev, [groupId]: !prev[groupId] };
      saveCollapsedGroups(next);
      return next;
    });
  }, []);

  // ── Keyboard navigation state ────────────────────────────────────
  const quickAccessRef = useRef<HTMLUListElement>(null);
  const navListRef = useRef<HTMLUListElement>(null);
  const skipLinkRef = useRef<HTMLAnchorElement>(null);
  const [focusedLinkIndex, setFocusedLinkIndex] = useState(-1);

  /** Collect all sidebar links into a flat array for keyboard nav */
  const getSidebarLinks = useCallback((): HTMLElement[] => {
    const links: HTMLElement[] = [];
    if (quickAccessRef.current) {
      links.push(
        ...Array.from(quickAccessRef.current.querySelectorAll<HTMLElement>('[data-sidebar-link]')),
      );
    }
    if (navListRef.current) {
      links.push(
        ...Array.from(navListRef.current.querySelectorAll<HTMLElement>('[data-sidebar-link]')),
      );
    }
    return links;
  }, []);

  useEffect(() => {
    setFocusedLinkIndex(-1);
  }, [collapsed]);

  const handleNavKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (
        e.key !== 'ArrowDown' &&
        e.key !== 'ArrowUp' &&
        e.key !== 'Home' &&
        e.key !== 'End'
      ) {
        return;
      }

      const links = getSidebarLinks();
      if (links.length === 0) return;

      let currentIdx = focusedLinkIndex;

      if (currentIdx === -1) {
        const activeEl = document.activeElement;
        currentIdx = links.indexOf(activeEl as HTMLElement);
        if (currentIdx === -1) currentIdx = 0;
      }

      e.preventDefault();

      switch (e.key) {
        case 'ArrowDown': {
          const next = currentIdx < links.length - 1 ? currentIdx + 1 : 0;
          setFocusedLinkIndex(next);
          links[next]?.focus();
          break;
        }
        case 'ArrowUp': {
          const prev = currentIdx > 0 ? currentIdx - 1 : links.length - 1;
          setFocusedLinkIndex(prev);
          links[prev]?.focus();
          break;
        }
        case 'Home': {
          setFocusedLinkIndex(0);
          links[0]?.focus();
          break;
        }
        case 'End': {
          setFocusedLinkIndex(links.length - 1);
          links[links.length - 1]?.focus();
          break;
        }
      }
    },
    [focusedLinkIndex, getSidebarLinks],
  );

  // ── Filter items by role for progressive mode ─────────────────────
  // We get role from legacyNav which already reads from useAuth
  const role = legacyNav.role;

  const filteredAlwaysVisible = useMemo(
    () => ALWAYS_VISIBLE_ITEMS.filter((item) => item.roles.includes(role)),
    [role],
  );

  const filteredGroups = useMemo(
    () =>
      SIDEBAR_GROUPS.filter((group) => {
        // Check minRole against current role
        const roleHierarchy = ['viewer', 'technician', 'support', 'manager', 'owner', 'admin'];
        const roleIndex = roleHierarchy.indexOf(role);
        const minIndex = roleHierarchy.indexOf(group.minRole);
        return roleIndex >= minIndex && roleIndex !== -1;
      }).map((group) => ({
        ...group,
        items: group.items.filter((item) => item.roles.includes(role)),
      })),
    [role],
  );

  // ── Quick access items for progressive mode ───────────────────────
  const progressiveQuickAccess = useMemo(() => {
    return DEFAULT_QUICK_ACCESS
      .map((path) => filteredAlwaysVisible.find((i) => i.path === path))
      .filter(Boolean) as SidebarItem[];
  }, [filteredAlwaysVisible]);

  // ══════════════════════════════════════════════════════════════════
  // UX-1.6: focus-visible ring class for keyboard users
  // ══════════════════════════════════════════════════════════════════
  const linkBaseClass = ({ isActive }: { isActive: boolean }) =>
    `flex items-center gap-3 px-3 py-2 rounded-lg transition-colors text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-900 ${
      isActive
        ? 'bg-blue-600 text-white'
        : 'text-slate-300 hover:bg-slate-800 hover:text-white'
    }`;

  return (
    <>
      {/* UX-1.6: Skip link — первый фокусируемый элемент в sidebar */}
      <a
        ref={skipLinkRef}
        href="#main-nav-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-4 focus:left-4 focus:z-[100] focus:px-4 focus:py-2 focus:bg-blue-600 focus:text-white focus:rounded-lg focus:shadow-lg focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950 transition-all duration-300"
        onClick={(e) => {
          e.preventDefault();
          const target = document.getElementById('main-nav-content');
          if (target) {
            target.setAttribute('tabindex', '-1');
            target.focus();
            target.addEventListener('blur', () => target.removeAttribute('tabindex'), { once: true });
          }
        }}
      >
        {t('skip_to_navigation') || 'Перейти к навигации'}
      </a>

      <aside
        className={`fixed left-0 top-0 z-40 h-screen bg-slate-900 transition-[width,transform] duration-300 ease-out will-change-transform flex flex-col
          ${collapsed ? 'w-20' : 'w-64'}
          ${mobileOpen ? 'translate-x-0' : '-translate-x-full'}
          lg:translate-x-0`}
        role="navigation"
        aria-label={t('sidebar_navigation') || 'Sidebar navigation'}
      >
        {/* Logo */}
        <div className="flex items-center justify-between h-16 px-4 border-b border-slate-800">
          <Link
            to="/dashboard"
            className="flex items-center gap-3"
            aria-label={t('go_to_dashboard') || 'Go to dashboard'}
          >
            <div className="flex items-center justify-center w-10 h-10 bg-blue-600 rounded-xl">
              <Camera className="w-5 h-5 text-white" />
            </div>
            {!collapsed && (
              <div className="overflow-hidden">
                <h1 className="text-lg font-bold text-white whitespace-nowrap">
                  CCTV Monitor
                </h1>
                <p className="text-xs text-slate-300">Health Dashboard</p>
              </div>
            )}
          </Link>
          {mobileOpen && (
            <button
              onClick={onMobileClose}
              className="lg:hidden p-2 text-slate-300 hover:text-white hover:bg-slate-800 rounded-lg transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400"
              aria-label={t('close_menu') || 'Close menu'}
            >
              <X className="w-5 h-5" />
            </button>
          )}
        </div>

        {progressiveEnabled ? (
          // ══════════════════════════════════════════════════════════
          // UX-1.1: Progressive Disclosure Mode
          // 14 always-visible items + 5 group accordions
          // ══════════════════════════════════════════════════════════
          <>
            {/* Quick Access Bar */}
            {!collapsed && progressiveQuickAccess.length > 0 && (
              <div className="px-3 pt-3 pb-2 border-b border-slate-800">
                <div className="flex items-center gap-1.5 mb-2 px-3">
                  <Star className="w-3 h-3 text-amber-400" />
                  <span className="text-xs font-medium text-slate-400 uppercase tracking-wider">
                    {t('quick_access') || 'Quick Access'}
                  </span>
                </div>
                <ul ref={quickAccessRef} className="space-y-0.5" onKeyDown={handleNavKeyDown}>
                  {progressiveQuickAccess.map((item) => {
                    const Icon = item.icon;
                    return (
                      <li key={item.path}>
                        <NavLink
                          to={item.path}
                          data-sidebar-link
                          className={linkBaseClass}
                          aria-current="page"
                        >
                          <Icon className="w-4 h-4 flex-shrink-0" />
                          <span className="text-sm font-medium truncate">
                            {t(item.label)}
                          </span>
                        </NavLink>
                      </li>
                    );
                  })}
                </ul>
              </div>
            )}

            {/* Always-visible items + Group accordions */}
            <nav
              id="main-nav-content"
              className="flex-1 px-3 py-3 overflow-y-auto"
              aria-label={t('sidebar_main_nav') || 'Main navigation'}
            >
              <ul ref={navListRef} className="space-y-0.5" onKeyDown={handleNavKeyDown}>
                {/* 14 always-visible items */}
                {!collapsed &&
                  filteredAlwaysVisible.map((item) => {
                    const Icon = item.icon;
                    return (
                      <li key={item.path}>
                        <NavLink
                          to={item.path}
                          data-sidebar-link
                          className={linkBaseClass}
                          aria-current="page"
                        >
                          <Icon className="w-4 h-4 flex-shrink-0" />
                          <span className="text-sm font-medium truncate">
                            {t(item.label)}
                          </span>
                        </NavLink>
                      </li>
                    );
                  })}

                {/* 5 Group accordions */}
                {!collapsed &&
                  filteredGroups.map((group) => {
                    if (group.items.length === 0) return null;
                    const GroupIcon = group.icon;
                    const collapsed = isGroupCollapsed(group.id);

                    return (
                      <li key={group.id} role="none">
                        <button
                          onClick={() => toggleGroupCollapsed(group.id)}
                          className="flex items-center justify-between w-full px-3 py-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-900 mt-2"
                          aria-expanded={!collapsed}
                          aria-controls={`sidebar-group-${group.id}`}
                          aria-label={`${t(group.label)} group`}
                        >
                          <div className="flex items-center gap-3">
                            <GroupIcon className="w-4 h-4" />
                            <span className="text-xs font-semibold uppercase tracking-wider">
                              {t(group.label)}
                            </span>
                          </div>
                          <ChevronDown
                            className={`w-4 h-4 transition-transform duration-200 ${
                              collapsed ? '-rotate-90' : 'rotate-0'
                            }`}
                          />
                        </button>

                        {/* Collapsible group items */}
                        {!collapsed && (
                          <ul
                            id={`sidebar-group-${group.id}`}
                            className="mt-1 space-y-0.5 ml-2 border-l border-slate-700 pl-2"
                            role="group"
                            aria-label={`${t(group.label)} items`}
                          >
                            {group.items.map((item) => {
                              const Icon = item.icon;
                              return (
                                <li key={item.path}>
                                  <NavLink
                                    to={item.path}
                                    data-sidebar-link
                                    className={linkBaseClass}
                                    aria-current="page"
                                  >
                                    <Icon className="w-4 h-4 flex-shrink-0" />
                                    <span className="text-sm font-medium truncate">
                                      {t(item.label)}
                                    </span>
                                  </NavLink>
                                </li>
                              );
                            })}
                          </ul>
                        )}
                      </li>
                    );
                  })}
              </ul>
            </nav>
          </>
        ) : (
          // ══════════════════════════════════════════════════════════
          // Legacy Sidebar (feature flag OFF)
          // ══════════════════════════════════════════════════════════
          <>
            {/* Quick Access Bar */}
            {!collapsed && legacyNav.quickAccess.length > 0 && (
              <div className="px-3 pt-3 pb-2 border-b border-slate-800">
                <div className="flex items-center gap-1.5 mb-2 px-3">
                  <Star className="w-3 h-3 text-amber-400" />
                  <span className="text-xs font-medium text-slate-400 uppercase tracking-wider">
                    {t('quick_access') || 'Quick Access'}
                  </span>
                </div>
                <ul ref={quickAccessRef} className="space-y-0.5" onKeyDown={handleNavKeyDown}>
                  {legacyNav.quickAccess.map((item) => {
                    const Icon = item.icon;
                    return (
                      <li key={item.path}>
                        <NavLink
                          to={item.path}
                          data-sidebar-link
                          className={linkBaseClass}
                          aria-current="page"
                        >
                          <Icon className="w-4 h-4 flex-shrink-0" />
                          <span className="text-sm font-medium truncate">
                            {item.label}
                          </span>
                        </NavLink>
                      </li>
                    );
                  })}
                </ul>
              </div>
            )}

            {/* Grouped Navigation */}
            <nav className="flex-1 px-3 py-3 overflow-y-auto">
              <ul ref={navListRef} className="space-y-2" onKeyDown={handleNavKeyDown}>
                {legacyNav.groups.map((group) => {
                  const GroupIcon = group.icon;
                  const expanded = legacyNav.isGroupExpanded(group.id);

                  return (
                    <li key={group.id}>
                      {collapsed ? (
                        <div className="flex items-center justify-center py-2">
                          <GroupIcon className="w-5 h-5 text-slate-400" />
                        </div>
                      ) : (
                        <>
                          <button
                            onClick={() => legacyNav.toggleGroup(group.id)}
                            className="flex items-center justify-between w-full px-3 py-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-900"
                            aria-expanded={expanded}
                            aria-label={`${group.label} group`}
                          >
                            <div className="flex items-center gap-3">
                              <GroupIcon className="w-4 h-4" />
                              <span className="text-xs font-semibold uppercase tracking-wider">
                                {group.label}
                              </span>
                            </div>
                            <ChevronDown
                              className={`w-4 h-4 transition-transform duration-200 ${
                                expanded ? 'rotate-0' : '-rotate-90'
                              }`}
                            />
                          </button>

                          {expanded && (
                            <ul className="mt-1 space-y-0.5 ml-2 border-l border-slate-700 pl-2">
                              {group.children.map((item) => {
                                const Icon = item.icon;
                                return (
                                  <li key={item.path}>
                                    <NavLink
                                      to={item.path}
                                      data-sidebar-link
                                      className={linkBaseClass}
                                      aria-current="page"
                                    >
                                      <Icon className="w-4 h-4 flex-shrink-0" />
                                      <span className="text-sm font-medium truncate">
                                        {item.label}
                                      </span>
                                    </NavLink>
                                  </li>
                                );
                              })}
                            </ul>
                          )}
                        </>
                      )}
                    </li>
                  );
                })}
              </ul>
            </nav>
          </>
        )}

        {/* Collapse Toggle */}
        <button
          onClick={onToggle}
          className="hidden lg:flex absolute -right-3 top-20 items-center justify-center w-6 h-6 bg-slate-700 border border-slate-600 rounded-full text-slate-300 hover:bg-slate-600 hover:text-white transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-900"
          aria-label={
            collapsed
              ? t('expand_sidebar') || 'Expand sidebar'
              : t('collapse_sidebar') || 'Collapse sidebar'
          }
        >
          {collapsed ? (
            <ChevronRight className="w-4 h-4" />
          ) : (
            <ChevronLeft className="w-4 h-4" />
          )}
        </button>
      </aside>
    </>
  );
});
