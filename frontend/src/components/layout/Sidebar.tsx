import React from 'react';
import { NavLink, Link, useLocation } from 'react-router-dom';
import { useAuth } from '../../hooks/useAuth';
import {
    LayoutDashboard,
    MapPin,
    HardDrive,
    Ticket,
    FileText,
    Users,
    Settings,
    ChevronLeft,
    ChevronRight,
    Camera,
    Shield,
    X,
    TrendingUp,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

interface SidebarProps {
    collapsed: boolean;
    onToggle: () => void;
    mobileOpen?: boolean;
    onMobileClose?: () => void;
}

export function Sidebar({ collapsed, onToggle, mobileOpen, onMobileClose }: SidebarProps) {
    const { t } = useTranslation();
    const location = useLocation();
    const { user } = useAuth();

    // Определяем пункты меню ВНУТРИ компонента, чтобы использовать t()
    const allNavItems = [
        { path: '/dashboard', label: t('dashboard'), icon: LayoutDashboard, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'] },
        { path: '/sites', label: t('sites'), icon: MapPin, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'] },
        { path: '/devices', label: t('devices'), icon: HardDrive, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'] },
        { path: '/tickets', label: t('tickets'), icon: Ticket, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'] },
        { path: '/alerts', label: t('alerts'), icon: Shield, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'] },
        { path: '/reports', label: t('reports'), icon: FileText, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'] },
        // CMMS Routes
        { path: '/maintenance', label: t('maintenance') || 'Maintenance', icon: FileText, roles: ['admin', 'manager', 'technician'] },
        { path: '/work-orders', label: t('work_orders') || 'Work Orders', icon: FileText, roles: ['admin', 'manager', 'technician'] },
        { path: '/spare-parts', label: t('spare_parts') || 'Spare Parts', icon: HardDrive, roles: ['admin', 'manager', 'technician'] },
        { path: '/sla', label: t('sla') || 'SLA', icon: TrendingUp, roles: ['admin', 'manager'] },
        { path: '/maintenance-reports', label: t('maintenance_reports') || 'Maintenance Reports', icon: FileText, roles: ['admin', 'manager'] },
        // Admin Only
        { path: '/users', label: t('users'), icon: Users, roles: ['admin'] },
        { path: '/settings', label: t('settings'), icon: Settings, roles: ['admin'] },
        { path: '/analytics', label: t('analytics'), icon: TrendingUp, roles: ['admin', 'support', 'owner'] },
        { path: '/logs', label: t('logs'), icon: FileText, roles: ['admin', 'support'] },
    ];

    const navItems = allNavItems.filter(item =>
        user && item.roles.includes(user.role)
    );

    return (
        <aside
            className={`fixed left-0 top-0 z-40 h-screen bg-slate-900 transition-all duration-300 flex flex-col
                ${collapsed ? 'w-20' : 'w-64'}
                ${mobileOpen ? 'translate-x-0' : '-translate-x-full'} 
                lg:translate-x-0`}
        >
            {/* Logo */}
            <div className="flex items-center justify-between h-16 px-4 border-b border-slate-800">
                <Link to="/dashboard" className="flex items-center gap-3">
                    <div className="flex items-center justify-center w-10 h-10 bg-blue-600 rounded-xl">
                        <Camera className="w-5 h-5 text-white" />
                    </div>
                    {!collapsed && (
                        <div className="overflow-hidden">
                            <h1 className="text-lg font-bold text-white whitespace-nowrap">
                                CCTV Monitor
                            </h1>
                            <p className="text-xs text-slate-400 dark:text-slate-300">Health Dashboard</p>
                        </div>
                    )}
                </Link>
                {mobileOpen && (
                    <button
                        onClick={onMobileClose}
                        className="lg:hidden p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
                        aria-label="Close menu"
                    >
                        <X className="w-5 h-5" />
                    </button>
                )}
            </div>

            {/* Navigation */}
            <nav className="flex-1 px-3 py-4 overflow-y-auto">
                <ul className="space-y-1">
                    {navItems.map((item) => {
                        const Icon = item.icon;
                        const isActive = location.pathname.startsWith(item.path);
                        return (
                            <li key={item.path}>
                                <NavLink
                                    to={item.path}
                                    className={`flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors ${isActive
                                        ? 'bg-blue-600 text-white'
                                        : 'text-slate-300 hover:bg-slate-800 hover:text-white'
                                        }`}
                                >
                                    <Icon className="w-5 h-5 flex-shrink-0" />
                                    {!collapsed && (
                                        <span className="text-sm font-medium whitespace-nowrap">
                                            {item.label}
                                        </span>
                                    )}
                                </NavLink>
                            </li>
                        );
                    })}
                </ul>
            </nav>

            {/* Collapse Toggle */}
            <button
                onClick={onToggle}
                className="hidden lg:flex absolute -right-3 top-20 items-center justify-center w-6 h-6 bg-slate-700 border border-slate-600 rounded-full text-slate-300 hover:bg-slate-600 hover:text-white transition-colors"
            >
                {collapsed ? <ChevronRight className="w-4 h-4" /> : <ChevronLeft className="w-4 h-4" />}
            </button>

        </aside>
    );
}