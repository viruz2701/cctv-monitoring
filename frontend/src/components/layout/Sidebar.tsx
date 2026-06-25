import React, { useState, useCallback } from 'react';
import { NavLink, Link } from 'react-router-dom';
import { useNavigation } from '../../hooks/useNavigation';
import { useTranslation } from 'react-i18next';
import {
    Camera,
    ChevronLeft,
    ChevronRight,
    ChevronDown,
    Star,
    X,
} from 'lucide-react';

interface SidebarProps {
    collapsed: boolean;
    onToggle: () => void;
    mobileOpen?: boolean;
    onMobileClose?: () => void;
}

/**
 * Sidebar — grouped navigation с quick access bar.
 *
 * P0-2.1: 5 групп (Dashboard, Assets, Operations, Insights, Administration)
 * P0-2.2: Role-based filtering через useNavigation()
 * P0-2.3: Quick access bar (3-4 pinned items сверху)
 */
export function Sidebar({ collapsed, onToggle, mobileOpen, onMobileClose }: SidebarProps) {
    const { t } = useTranslation();
    const {
        groups,
        quickAccess,
        isGroupExpanded,
        toggleGroup,
    } = useNavigation();

    const [activeGroup, setActiveGroup] = useState<string | null>(null);

    const handleToggleGroup = useCallback((groupId: string) => {
        toggleGroup(groupId);
        setActiveGroup(prev => prev === groupId ? null : groupId);
    }, [toggleGroup]);

    return (
        <aside
            className={`fixed left-0 top-0 z-40 h-screen bg-slate-900 transition-all duration-300 flex flex-col
                ${collapsed ? 'w-20' : 'w-64'}
                ${mobileOpen ? 'translate-x-0' : '-translate-x-full'} 
                lg:translate-x-0`}
            role="navigation"
            aria-label={t('sidebar_navigation') || 'Sidebar navigation'}
        >
            {/* Logo */}
            <div className="flex items-center justify-between h-16 px-4 border-b border-slate-800">
                <Link to="/dashboard" className="flex items-center gap-3" aria-label="Go to dashboard">
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
                        className="lg:hidden p-2 text-slate-300 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
                        aria-label={t('close_menu') || 'Close menu'}
                    >
                        <X className="w-5 h-5" />
                    </button>
                )}
            </div>

            {/* P0-2.3: Quick Access Bar */}
            {!collapsed && quickAccess.length > 0 && (
                <div className="px-3 pt-3 pb-2 border-b border-slate-800">
                    <div className="flex items-center gap-1.5 mb-2 px-3">
                        <Star className="w-3 h-3 text-amber-400" />
                        <span className="text-xs font-medium text-slate-400 uppercase tracking-wider">
                            {t('quick_access') || 'Quick Access'}
                        </span>
                    </div>
                    <ul className="space-y-0.5">
                        {quickAccess.map((item) => {
                            const Icon = item.icon;
                            return (
                                <li key={item.path}>
                                    <NavLink
                                        to={item.path}
                                        className={({ isActive }) =>
                                            `flex items-center gap-3 px-3 py-2 rounded-lg transition-colors text-sm ${
                                                isActive
                                                    ? 'bg-blue-600 text-white'
                                                    : 'text-slate-300 hover:bg-slate-800 hover:text-white'
                                            }`
                                        }
                                        aria-label={item.label}
                                    >
                                        <Icon className="w-4 h-4 flex-shrink-0" />
                                        <span className="text-sm font-medium truncate">{item.label}</span>
                                    </NavLink>
                                </li>
                            );
                        })}
                    </ul>
                </div>
            )}

            {/* P0-2.1 + P0-2.2: Grouped Navigation */}
            <nav className="flex-1 px-3 py-3 overflow-y-auto">
                <ul className="space-y-2">
                    {groups.map((group) => {
                        const GroupIcon = group.icon;
                        const expanded = isGroupExpanded(group.id);

                        return (
                            <li key={group.id}>
                                {/* Group Header (collapsible) */}
                                {collapsed ? (
                                    // В collapsed режиме показываем только иконку группы
                                    <div className="flex items-center justify-center py-2">
                                        <GroupIcon className="w-5 h-5 text-slate-400" />
                                    </div>
                                ) : (
                                    <>
                                        <button
                                            onClick={() => handleToggleGroup(group.id)}
                                            className="flex items-center justify-between w-full px-3 py-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
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

                                        {/* Group Children (collapsible) */}
                                        {expanded && (
                                            <ul className="mt-1 space-y-0.5 ml-2 border-l border-slate-700 pl-2">
                                                {group.children.map((item) => {
                                                    const Icon = item.icon;
                                                    return (
                                                        <li key={item.path}>
                                                            <NavLink
                                                                to={item.path}
                                                                className={({ isActive }) =>
                                                                    `flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                                                                        isActive
                                                                            ? 'bg-blue-600 text-white'
                                                                            : 'text-slate-300 hover:bg-slate-800 hover:text-white'
                                                                    }`
                                                                }
                                                                aria-label={item.label}
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

            {/* Collapse Toggle */}
            <button
                onClick={onToggle}
                className="hidden lg:flex absolute -right-3 top-20 items-center justify-center w-6 h-6 bg-slate-700 border border-slate-600 rounded-full text-slate-300 hover:bg-slate-600 hover:text-white transition-colors"
                aria-label={collapsed ? t('expand_sidebar') || 'Expand sidebar' : t('collapse_sidebar') || 'Collapse sidebar'}
            >
                {collapsed ? <ChevronRight className="w-4 h-4" /> : <ChevronLeft className="w-4 h-4" />}
            </button>

        </aside>
    );
}
