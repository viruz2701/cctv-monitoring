import React, { useState, useCallback, useMemo } from 'react';
import { useLocation, Link, useNavigate } from 'react-router-dom';
import {
    Bell,
    Search,
    ChevronDown,
    User,
    LogOut,
    Settings,
    Menu,
    X,
    Sun,
    Moon,
    Monitor
} from 'lucide-react';
import { useAuth } from '../../hooks/useAuth';
import { LanguageSwitcher } from '../LanguageSwitcher';
import { WorkspaceSwitcher } from './WorkspaceSwitcher';
import { useDevices, useSites, useNotifications, useMarkAllNotificationsRead } from '../../hooks/useApiQuery';
import { ConfirmModal } from '../ui/Modal';
import { useTranslation } from 'react-i18next';
import { useThemeStore, type Theme } from '../../store/themeStore';

interface HeaderProps {
    onMobileMenuToggle?: () => void;
    sidebarCollapsed: boolean;
}

export function Header({ onMobileMenuToggle, sidebarCollapsed }: HeaderProps) {
    const { t, i18n } = useTranslation();
    const navigate = useNavigate();
    const location = useLocation();
    const { user, logout } = useAuth();
    const { data: apiNotifications = [] } = useNotifications();
    const markAllAsReadMut = useMarkAllNotificationsRead();
    const { data: rawDevices = [] } = useDevices();
    const { data: rawSites = [] } = useSites();

    const notifications = useMemo(() => apiNotifications.map(n => ({
        id: n.id,
        title: n.title,
        message: n.message,
        type: n.type,
        read: n.read,
        link: n.link,
        timestamp: n.created_at,
    })), [apiNotifications]);

    const unreadCount = useMemo(() => notifications.filter(n => !n.read).length, [notifications]);

    const markAllAsRead = () => markAllAsReadMut.mutate();

    const mapAPIDeviceToUI = (d: Record<string, any>): { id: string; name: string; siteId: string; siteName: string; type: string; status: string; health: string; recordingStatus: string; lastSeen: string; ipAddress: string; model: string; firmware: string; owner_id: any } => ({
        id: d.device_id,
        name: d.name || d.device_id,
        siteId: d.site_id || 'site-default',
        siteName: d.location || 'Unknown',
        type: d.vendor_type === 'camera' ? 'camera' : 'nvr',
        status: (d.status || 'offline').toLowerCase(),
        health: d.status === 'online' ? 'healthy' : 'faulty',
        recordingStatus: 'recording',
        lastSeen: d.last_seen || new Date().toISOString(),
        ipAddress: '',
        model: d.vendor_type || '',
        firmware: '',
        owner_id: d.owner_id,
    });

    const devices = useMemo(() => {
        const devsArray = Array.isArray(rawDevices) ? rawDevices : (rawDevices && typeof rawDevices === 'object' && 'devices' in rawDevices ? (rawDevices as any).devices : []);
        return devsArray.map(mapAPIDeviceToUI);
    }, [rawDevices]);

    const mapAPISiteToUI = (s: Record<string, any>) => ({
        id: s.id,
        name: s.name || 'Unnamed',
        address: s.address || '',
        city: s.city || '',
        organization: s.organization || '',
        latitude: s.latitude || 0,
        longitude: s.longitude || 0,
        status: s.status || 'active',
        lastSync: s.last_sync || new Date().toISOString(),
    });

    const sites = useMemo(() => rawSites.map(mapAPISiteToUI), [rawSites]);

    const { theme, setTheme } = useThemeStore();
    const [userMenuOpen, setUserMenuOpen] = useState(false);
    const [notificationsOpen, setNotificationsOpen] = useState(false);
    const [isLogoutModalOpen, setIsLogoutModalOpen] = useState(false);

    const [searchQuery, setSearchQuery] = useState('');
    const [showSearchResults, setShowSearchResults] = useState(false);

    // Динамический заголовок страницы (пока не переводим, можно оставить английские ключи или добавить)
    const pageTitleMap: Record<string, string> = {
        '/dashboard': t('dashboard'),
        '/sites': t('sites'),
        '/devices': t('devices'),
        '/tickets': t('tickets'),
        '/reports': t('reports'),
        '/users': t('users'),
        '/settings': t('settings'),
        '/notifications': t('notifications'),
        '/alerts': t('alerts'),
        '/profile': t('profile'),
        '/analytics': t('analytics'),
        '/logs': t('logs'),
    };
    const currentTitle = Object.entries(pageTitleMap).find(([path]) =>
        location.pathname.startsWith(path)
    )?.[1] || t('dashboard');

    const searchResults = React.useMemo((): { devices: Array<Record<string, any>>; sites: Array<Record<string, any>> } => {
        if (!searchQuery.trim()) return { devices: [], sites: [] };
        const query = searchQuery.toLowerCase();
        return {
            sites: sites.filter(s => s.name.toLowerCase().includes(query) || s.address.toLowerCase().includes(query)).slice(0, 3),
            devices: devices.filter(d => d.name.toLowerCase().includes(query) || d.type.toLowerCase().includes(query)).slice(0, 3)
        };
    }, [searchQuery, sites, devices]);

    const themeCycle: Theme[] = ['light', 'dark', 'system'];
    const themeIcons: Record<Theme, React.ReactNode> = {
        light: <Sun className="w-5 h-5" />,
        dark: <Moon className="w-5 h-5" />,
        system: <Monitor className="w-5 h-5" />,
    };
    const themeLabels: Record<Theme, string> = {
        light: t('theme_light') || 'Light',
        dark: t('theme_dark') || 'Dark',
        system: t('theme_system') || 'System',
    };

    const cycleTheme = useCallback(() => {
        const currentIndex = themeCycle.indexOf(theme);
        const nextTheme = themeCycle[(currentIndex + 1) % themeCycle.length];
        setTheme(nextTheme);
    }, [theme, setTheme]);

    const handleSearchSelect = (path: string) => {
        setSearchQuery('');
        setShowSearchResults(false);
        navigate(path);
    };

    return (
        <header className={`fixed top-0 right-0 left-0 z-30 h-16 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 transition-all duration-300 ${sidebarCollapsed ? 'lg:left-20' : 'lg:left-64'}`}>
            <div className="flex items-center justify-between h-full px-6">
                <div className="flex items-center gap-2">
                    <WorkspaceSwitcher />
                    <div className="h-6 w-px bg-slate-200 dark:bg-slate-700 mx-1" />
                    <LanguageSwitcher />
                    <button onClick={onMobileMenuToggle} className="lg:hidden p-2 text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg">
                        <Menu className="w-5 h-5" />
                    </button>
                    <div>
                        <h1 className="text-lg md:text-xl font-bold text-slate-900 dark:text-white">{currentTitle}</h1>
                        <p className="hidden sm:block text-sm text-slate-500 dark:text-slate-400">
                            {new Date().toLocaleDateString(i18n.language === 'ru' ? 'ru-RU' : 'en-US', {
                                weekday: 'long',
                                year: 'numeric',
                                month: 'long',
                                day: 'numeric',
                            })}
                        </p>
                    </div>
                </div>

                <div className="flex items-center gap-4">
                    {/* Search */}
                    <div className="hidden md:flex items-center relative z-50">
                        <div className="relative">
                            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                            <input
                                type="text"
                                placeholder={t('search_placeholder') || "Search devices, sites..."}
                                value={searchQuery}
                                onChange={(e) => { setSearchQuery(e.target.value); setShowSearchResults(true); }}
                                onFocus={() => setShowSearchResults(true)}
                                onBlur={() => setTimeout(() => setShowSearchResults(false), 200)}
                                className="w-64 pl-10 pr-10 py-2 text-sm bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-white border-0 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 placeholder:text-slate-400 dark:placeholder:text-slate-500"
                            />
                            {searchQuery && (
                                <button onClick={() => setSearchQuery('')} className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600">
                                    <X className="w-3 h-3" />
                                </button>
                            )}
                            {showSearchResults && searchQuery && (searchResults.sites.length > 0 || searchResults.devices.length > 0) && (
                                <div className="absolute top-full left-0 right-0 mt-2 bg-white dark:bg-slate-900 rounded-xl shadow-xl border border-slate-200 dark:border-slate-800 overflow-hidden">
                                    {searchResults.sites.length > 0 && (
                                        <div className="py-2">
                                            <h4 className="px-4 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('sites')}</h4>
                                            {searchResults.sites.map(site => (
                                                <button key={site.id} onMouseDown={(e) => e.preventDefault()} onClick={() => handleSearchSelect(`/sites?search=${encodeURIComponent(site.name)}`)} className="w-full text-left px-4 py-2 hover:bg-slate-50 dark:hover:bg-slate-800 flex items-center gap-2">
                                                    <div className="p-1 bg-blue-100 dark:bg-blue-900/30 rounded text-blue-600 dark:text-blue-400"><Search className="w-3 h-3" /></div>
                                                    <div><p className="text-sm font-medium text-slate-900 dark:text-white">{site.name}</p><p className="text-xs text-slate-500 dark:text-slate-400 truncate">{site.address}</p></div>
                                                </button>
                                            ))}
                                        </div>
                                    )}
                                    {searchResults.devices.length > 0 && (
                                        <div className="py-2 border-t border-slate-100 dark:border-slate-800">
                                            <h4 className="px-4 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('devices')}</h4>
                                            {searchResults.devices.map(device => (
                                                <button key={device.id} onMouseDown={(e) => e.preventDefault()} onClick={() => handleSearchSelect(`/devices/${device.id}`)} className="w-full text-left px-4 py-2 hover:bg-slate-50 dark:hover:bg-slate-800 flex items-center gap-2">
                                                    <div className="p-1 bg-emerald-100 dark:bg-emerald-900/30 rounded text-emerald-600 dark:text-emerald-400"><Search className="w-3 h-3" /></div>
                                                    <div><p className="text-sm font-medium text-slate-900 dark:text-white">{device.name}</p><p className="text-xs text-slate-500 dark:text-slate-400 capitalize">{device.type} • {device.status}</p></div>
                                                </button>
                                            ))}
                                        </div>
                                    )}
                                </div>
                            )}
                        </div>
                    </div>

                    {/* Theme Toggle */}
                    <button
                        onClick={cycleTheme}
                        title={themeLabels[theme]}
                        className="relative p-2 text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg group"
                    >
                        <span className="block transition-transform duration-300 ease-in-out group-active:rotate-90">
                            {themeIcons[theme]}
                        </span>
                        <span className="absolute -bottom-8 left-1/2 -translate-x-1/2 px-2 py-1 text-xs font-medium text-white bg-slate-800 dark:bg-slate-700 rounded-md opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap pointer-events-none">
                            {themeLabels[theme]}
                        </span>
                    </button>

                    {/* Notifications */}
                    <div className="relative">
                        <button onClick={() => setNotificationsOpen(!notificationsOpen)} className="relative p-2 text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg">
                            <Bell className="w-5 h-5" />
                            {unreadCount > 0 && <span className="absolute top-1 right-1 flex h-4 w-4 items-center justify-center rounded-full bg-red-500 text-[10px] text-white">{unreadCount > 9 ? '9+' : unreadCount}</span>}
                        </button>
                        {notificationsOpen && (
                            <>
                                <div className="fixed inset-0 z-40" onClick={() => setNotificationsOpen(false)} />
                                <div className="fixed inset-x-3 top-14 z-50 sm:absolute sm:inset-auto sm:right-0 sm:top-auto sm:mt-2 sm:w-80 max-h-[70vh] sm:max-h-[28rem] flex flex-col bg-white dark:bg-slate-900 rounded-xl shadow-xl border border-slate-200 dark:border-slate-800 overflow-hidden">
                                    <div className="px-4 py-2.5 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between flex-shrink-0">
                                        <h3 className="text-sm font-semibold text-slate-900 dark:text-white">{t('notifications')}</h3>
                                        {unreadCount > 0 && <button onClick={markAllAsRead} className="text-xs text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 font-medium">{t('mark_all_read')}</button>}
                                    </div>
                                    <div className="flex-1 overflow-y-auto overscroll-contain">
                                        {notifications.filter(n => !n.read).length === 0 ? (
                                            <div className="px-4 py-8 text-center text-slate-500 dark:text-slate-400"><p className="text-sm">{t('no_notifications')}</p></div>
                                        ) : (
                                            notifications.filter(n => !n.read).map((notification) => (
                                                <div key={notification.id} className="px-4 py-3 hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer border-b border-slate-50 dark:border-slate-800/50 last:border-0">
                                                    <p className="text-sm font-medium text-slate-900 dark:text-gray-200">{notification.title}</p>
                                                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1 line-clamp-2">{notification.message}</p>
                                                    <p className="text-[10px] text-slate-400 mt-1">{new Date(notification.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</p>
                                                </div>
                                            ))
                                        )}
                                    </div>
                                    <div className="px-4 py-2 border-t border-slate-100 dark:border-slate-800 flex-shrink-0">
                                        <Link to="/notifications" className="text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 block text-center" onClick={() => setNotificationsOpen(false)}>{t('view_all')}</Link>
                                    </div>
                                </div>
                            </>
                        )}
                    </div>

                    {/* User Menu */}
                    <div className="relative">
                        <button onClick={() => setUserMenuOpen(!userMenuOpen)} className="flex items-center gap-3 p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800">
                            <div className="flex items-center justify-center w-8 h-8 bg-blue-600 text-white text-sm font-semibold rounded-full overflow-hidden">
                                {user?.avatar && user.avatar.length > 4 ? (
                                    <img src={user.avatar} alt={user.name} className="w-full h-full object-cover" />
                                ) : (
                                    <span className="text-sm">{user?.avatar || (user?.name || '').split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)}</span>
                                )}
                            </div>
                            <div className="hidden md:block text-left">
                                <p className="text-sm font-medium text-slate-900 dark:text-white">{user?.name}</p>
                                <p className="text-xs text-slate-500 dark:text-slate-400 capitalize">{user?.role}</p>
                            </div>
                            <ChevronDown className="w-4 h-4 text-slate-400" />
                        </button>
                        {userMenuOpen && (
                            <div className="absolute right-0 mt-2 w-48 bg-white dark:bg-slate-900 rounded-xl shadow-lg border border-slate-200 dark:border-slate-800 py-2 z-50">
                                <Link to="/profile" className="flex items-center gap-3 px-4 py-2 text-sm text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-800" onClick={() => setUserMenuOpen(false)}><User className="w-4 h-4" /> {t('profile')}</Link>
                                <Link to="/settings" className="flex items-center gap-3 px-4 py-2 text-sm text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-800" onClick={() => setUserMenuOpen(false)}><Settings className="w-4 h-4" /> {t('settings')}</Link>
                                <hr className="my-2 border-slate-100 dark:border-slate-800" />
                                <button onClick={() => { setUserMenuOpen(false); setIsLogoutModalOpen(true); }} className="flex w-full items-center gap-3 px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/10"><LogOut className="w-4 h-4" /> {t('sign_out')}</button>
                            </div>
                        )}
                    </div>
                </div>
            </div>
            <ConfirmModal
                isOpen={isLogoutModalOpen}
                onClose={() => setIsLogoutModalOpen(false)}
                onConfirm={logout}
                title={t('sign_out')}
                message={t('sign_out_confirm')}
                confirmText={t('sign_out')}
                cancelText={t('cancel')}
            />
        </header>
    );
}