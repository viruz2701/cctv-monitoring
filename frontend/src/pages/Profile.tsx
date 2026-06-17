import React, { useState, useEffect } from 'react';
import { z } from 'zod';
import { User, Mail, Shield, Smartphone, Moon, Sun, Lock, LogOut, Camera, Save, X, Edit2, MapPin, Briefcase, Trash2, Clock, Calendar, CheckCircle } from 'lucide-react';
import { QRCodeSVG } from 'qrcode.react';
import { Card, CardHeader, CardBody, Button, useToast, ConfirmModal, Modal, Input } from '../components/ui';
import { useAuth } from '../hooks/useAuth';
import { useTheme } from '../context/ThemeContext';
import { useTranslation } from 'react-i18next';
import { api } from '../services/api';

export function Profile() {
    const { t } = useTranslation();
    const { user, logout, updateUser } = useAuth();
    const { theme, setTheme, isDark } = useTheme();
    const toast = useToast();

    const profileSchema = z.object({
        name: z.string().min(2, t('name_min_length') || 'Name must be at least 2 characters'),
        email: z.string().email(t('invalid_email') || 'Invalid email address'),
        phone: z.string().min(10, t('phone_min_length') || 'Phone number must be at least 10 digits').optional().or(z.literal('')),
        location: z.string().max(100, t('location_max_length') || 'Location is too long').optional(),
        avatar: z.string().optional(),
    });

    const fileInputRef = React.useRef<HTMLInputElement>(null);

    const handleImageUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (file) {
            const reader = new FileReader();
            reader.onloadend = () => {
                setFormData(prev => ({ ...prev, avatar: reader.result as string }));
            };
            reader.readAsDataURL(file);
        }
    };

    const [isEditing, setIsEditing] = useState(false);
    const [isLogoutModalOpen, setIsLogoutModalOpen] = useState(false);
    const [errors, setErrors] = useState<Record<string, string>>({});
    const [formData, setFormData] = useState(() => ({
        name: user?.name ?? '',
        email: user?.email ?? '',
        role: user?.role ?? '',
        avatar: user?.avatar ?? '',
        phone: '+1 (555) 123-4567',
        location: 'New York, USA',
        department: 'Security Operations',
    }));

    // ═══ Password Change States ═══
    const [showPasswordModal, setShowPasswordModal] = useState(false);
    const [passwordForm, setPasswordForm] = useState({ current: '', newPass: '', confirm: '' });
    const [passwordLoading, setPasswordLoading] = useState(false);
    const [passwordError, setPasswordError] = useState('');

    // ═══ Session Management States ═══
    const [sessions, setSessions] = useState<any[]>([]);
    const [sessionsLoading, setSessionsLoading] = useState(false);
    const [showRevokeAllConfirm, setShowRevokeAllConfirm] = useState(false);

    // ═══ 2FA States ═══
    const [show2FAModal, setShow2FAModal] = useState(false);
    const [totpEnabled, setTotpEnabled] = useState(false);
    const [totpSecret, setTotpSecret] = useState('');
    const [totpUri, setTotpUri] = useState('');
    const [totpCode, setTotpCode] = useState('');
    const [totpLoading, setTotpLoading] = useState(false);
    const [totpError, setTotpError] = useState('');
    const [showDisable2FAModal, setShowDisable2FAModal] = useState(false);
    const [disable2FAPassword, setDisable2FAPassword] = useState('');

    // ═══ Telegram States ═══
    const [telegramLinked, setTelegramLinked] = useState(false);
    const [telegramAlerts, setTelegramAlerts] = useState(false);
    const [telegram2FA, setTelegram2FA] = useState(false);
    const [telegramToken, setTelegramToken] = useState('');
    const [telegramLoading, setTelegramLoading] = useState(false);

    useEffect(() => {
        if (user && !isEditing) {
            setFormData(prev => {
                const newName = user.name ?? prev.name;
                const newEmail = user.email ?? prev.email;
                const newRole = user.role ?? prev.role;
                const newAvatar = user.avatar ?? '';

                if (prev.name === newName && prev.email === newEmail && prev.role === newRole && prev.avatar === newAvatar) {
                    return prev;
                }
                return { ...prev, name: newName, email: newEmail, role: newRole, avatar: newAvatar };
            });
        }
    }, [user, isEditing]);

    // Load Telegram status
    useEffect(() => {
        const loadTelegramStatus = async () => {
            try {
                const status = await api.getTelegramStatus();
                setTelegramLinked(status.linked);
                setTelegramAlerts(status.alerts);
                setTelegram2FA(status.tfa);
            } catch (err) {
                console.error('Failed to load Telegram status:', err);
            }
        };
        loadTelegramStatus();
    }, []);

    if (!user) return null;

    // ═══ Password Change Handlers ═══
    const handlePasswordReset = () => {
        setPasswordForm({ current: '', newPass: '', confirm: '' });
        setPasswordError('');
        setShowPasswordModal(true);
    };

    const handleChangePassword = async () => {
        setPasswordError('');
        
        if (!passwordForm.current || !passwordForm.newPass || !passwordForm.confirm) {
            setPasswordError(t('all_fields_required') || 'All fields are required');
            return;
        }
        if (passwordForm.newPass.length < 6) {
            setPasswordError(t('password_min_length') || 'New password must be at least 6 characters');
            return;
        }
        if (passwordForm.newPass !== passwordForm.confirm) {
            setPasswordError(t('passwords_do_not_match') || 'New passwords do not match');
            return;
        }

        setPasswordLoading(true);
        try {
            await api.changePassword(passwordForm.current, passwordForm.newPass);
            toast.success(t('password_changed') || 'Password changed successfully');
            setShowPasswordModal(false);
            setPasswordForm({ current: '', newPass: '', confirm: '' });
        } catch (err: any) {
            setPasswordError(err.message || t('password_change_failed') || 'Failed to change password');
        } finally {
            setPasswordLoading(false);
        }
    };

    // ═══ Session Management Handlers ═══
    useEffect(() => {
        loadSessions();
    }, []);

    const loadSessions = async () => {
        setSessionsLoading(true);
        try {
            const data = await api.getSessions();
            setSessions(data ?? []);
        } catch (err: any) {
            console.error('Failed to load sessions', err);
            setSessions([]);
        } finally {
            setSessionsLoading(false);
        }
    };

    const handleRevokeSession = async (sessionId: string) => {
        try {
            await api.revokeSession(sessionId);
            toast.success(t('session_revoked'));
            loadSessions();
        } catch (err: any) {
            toast.error(err.message || 'Failed to revoke session');
        }
    };

    const handleRevokeAllOtherSessions = async () => {
        try {
            // We don't have the current session ID, so we pass empty string
            // The backend will revoke all sessions for this user
            await api.revokeAllOtherSessions('');
            toast.success(t('sessions_revoked'));
            loadSessions();
        } catch (err: any) {
            toast.error(err.message || 'Failed to revoke sessions');
        }
        setShowRevokeAllConfirm(false);
    };

    // ═══ 2FA Handlers ═══
    const handleSetup2FA = async () => {
        setTotpError('');
        setTotpLoading(true);
        try {
            const { secret, uri } = await api.setup2FA();
            setTotpSecret(secret);
            setTotpUri(uri);
            setShow2FAModal(true);
        } catch (err: any) {
            setTotpError(err.message || 'Failed to setup 2FA');
        } finally {
            setTotpLoading(false);
        }
    };

    const handleVerify2FA = async () => {
        setTotpError('');
        if (!totpCode || totpCode.length !== 6) {
            setTotpError(t('2fa_code_invalid') || 'Please enter a valid 6-digit code');
            return;
        }
        setTotpLoading(true);
        try {
            await api.verify2FA(totpCode);
            toast.success(t('2fa_enabled') || '2FA enabled successfully');
            setTotpEnabled(true);
            setShow2FAModal(false);
            setTotpCode('');
            setTotpSecret('');
            setTotpUri('');
        } catch (err: any) {
            setTotpError(err.message || 'Invalid 2FA code');
        } finally {
            setTotpLoading(false);
        }
    };

    const handleDisable2FA = async () => {
        setTotpError('');
        if (!disable2FAPassword) {
            setTotpError(t('password_required') || 'Password is required');
            return;
        }
        setTotpLoading(true);
        try {
            await api.disable2FA(disable2FAPassword);
            toast.success(t('2fa_disabled') || '2FA disabled successfully');
            setTotpEnabled(false);
            setShowDisable2FAModal(false);
            setDisable2FAPassword('');
        } catch (err: any) {
            setTotpError(err.message || 'Failed to disable 2FA');
        } finally {
            setTotpLoading(false);
        }
    };

    // ═══ Telegram Handlers ═══
    const handleGenerateTelegramLink = async () => {
        setTelegramLoading(true);
        try {
            const result = await api.generateTelegramLink();
            setTelegramToken(result.token);
            toast.success(t('telegram_link_generated'));
        } catch (err: any) {
            toast.error(err.message);
        } finally {
            setTelegramLoading(false);
        }
    };

    const handleUpdateTelegramSettings = async (alerts: boolean, tfa: boolean) => {
        try {
            await api.updateTelegramSettings({ alerts, tfa });
            setTelegramAlerts(alerts);
            setTelegram2FA(tfa);
            toast.success(t('telegram_settings_updated'));
        } catch (err: any) {
            toast.error(err.message);
        }
    };

    const handleSave = () => {
        const result = profileSchema.safeParse(formData);
        if (!result.success) {
            const formattedErrors: Record<string, string> = {};
            const issues = result.error?.errors || result.error?.issues || [];
            if (issues.length > 0) {
                issues.forEach((err: any) => {
                    const path = err.path?.[0];
                    if (path) {
                        formattedErrors[path] = err.message;
                        toast.error(err.message);
                    }
                });
            } else {
                toast.error(t('form_errors') || 'Please check the form for errors');
            }
            setErrors(formattedErrors);
            return;
        }
        setErrors({});
        updateUser({ name: formData.name, email: formData.email, avatar: formData.avatar });
        toast.success(t('profile_updated') || 'Profile updated successfully');
        setIsEditing(false);
    };

    const handleCancel = () => {
        setFormData(prev => ({
            ...prev,
            name: user.name ?? prev.name,
            email: user.email ?? prev.email,
            role: user.role ?? prev.role,
            avatar: user.avatar ?? '',
        }));
        setIsEditing(false);
    };

    const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        const { name, value } = e.target;
        setFormData(prev => ({ ...prev, [name]: value }));
    };

    const initials = (formData.name || 'User')
        .split(' ')
        .map(n => n[0])
        .join('')
        .toUpperCase()
        .slice(0, 2);

    return (
        <div className="space-y-5 max-w-5xl mx-auto pb-10">
            {/* Profile Header Card */}
            <div className="bg-white dark:bg-slate-800 rounded-2xl border border-slate-200 dark:border-slate-700 p-5 md:p-8 shadow-sm">
                <div className="flex flex-col md:flex-row items-center md:items-start gap-5">
                    {/* Avatar */}
                    <div className="relative group flex-shrink-0">
                        <div className="p-1 rounded-full bg-gradient-to-br from-indigo-500 via-blue-500 to-cyan-400">
                            <div
                                className={`w-20 h-20 md:w-24 md:h-24 rounded-full border-[3px] border-white dark:border-slate-800 bg-slate-100 dark:bg-slate-700 flex items-center justify-center overflow-hidden relative ${isEditing ? 'cursor-pointer' : ''}`}
                                onClick={() => isEditing && fileInputRef.current?.click()}
                            >
                                {formData.avatar && formData.avatar.length > 4 ? (
                                    <img src={formData.avatar} alt={formData.name} className="w-full h-full object-cover" />
                                ) : (
                                    <span className="text-2xl md:text-3xl font-bold text-indigo-600 dark:text-indigo-300">{initials}</span>
                                )}
                                {isEditing && (
                                    <div className="absolute inset-0 bg-black/50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity duration-200 rounded-full">
                                        <Camera className="w-5 h-5 md:w-6 md:h-6 text-white" />
                                    </div>
                                )}
                            </div>
                        </div>
                        {isEditing && formData.avatar && formData.avatar.length > 4 && (
                            <button
                                className="absolute -top-1 -right-1 p-1.5 bg-red-500 text-white rounded-full shadow-md hover:bg-red-600 transition-colors z-10"
                                onClick={(e) => { e.stopPropagation(); setFormData(prev => ({ ...prev, avatar: '' })); }}
                                title={t('remove_photo')}
                            >
                                <Trash2 className="w-3 h-3" />
                            </button>
                        )}
                        <input type="file" ref={fileInputRef} className="hidden" accept="image/*" onChange={handleImageUpload} />
                    </div>

                    {/* Name / Meta */}
                    <div className="flex-1 text-center md:text-left min-w-0">
                        <h1 className="text-xl md:text-2xl font-bold text-slate-900 dark:text-white leading-tight">{formData.name || t('user')}</h1>
                        <div className="flex flex-wrap items-center justify-center md:justify-start gap-2 mt-1.5">
                            <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300 uppercase tracking-wide">
                                {user.role}
                            </span>
                            <span className="flex items-center gap-1 text-xs text-slate-400 dark:text-slate-500">
                                <Calendar className="w-3 h-3" />
                                {t('joined')} Jan 2026
                            </span>
                        </div>
                    </div>

                    {/* Action Button */}
                    <div className="flex-shrink-0">
                        {isEditing ? (
                            <div className="flex gap-2">
                                <Button variant="outline" onClick={handleCancel}>{t('cancel')}</Button>
                                <Button onClick={handleSave} icon={<Save className="w-4 h-4" />}>{t('save_changes')}</Button>
                            </div>
                        ) : (
                            <Button onClick={() => setIsEditing(true)} icon={<Edit2 className="w-4 h-4" />}>{t('update_profile')}</Button>
                        )}
                    </div>
                </div>
            </div>

            {/* Main Content Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
                <div className="lg:col-span-2 space-y-5">
                    {/* Core Identity */}
                    <div className="bg-white dark:bg-slate-800 rounded-2xl border border-slate-200 dark:border-slate-700 p-5 md:p-6 shadow-sm">
                        <h3 className="text-[11px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-[0.12em] mb-5">{t('core_identity')}</h3>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-x-5 gap-y-4">
                            <div>
                                <label className="text-[11px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-[0.1em] mb-1.5 block">{t('full_name')}</label>
                                {isEditing ? (
                                    <>
                                        <input type="text" name="name" value={formData.name} onChange={handleChange} className={`w-full px-3 py-2.5 bg-white dark:bg-slate-900 border ${errors.name ? 'border-red-500' : 'border-slate-200 dark:border-slate-700'} rounded-xl focus:ring-2 focus:ring-indigo-500 outline-none text-sm`} />
                                        {errors.name && <p className="text-red-500 text-[10px] mt-1 ml-1">{errors.name}</p>}
                                    </>
                                ) : (
                                    <div className="flex items-center gap-3 px-3 py-2.5 bg-slate-50 dark:bg-slate-900/50 border border-slate-100 dark:border-slate-700/50 rounded-xl">
                                        <User className="w-4 h-4 text-slate-400" /><span className="text-sm font-medium text-slate-800 dark:text-slate-200 truncate">{formData.name}</span>
                                    </div>
                                )}
                            </div>
                            <div>
                                <label className="text-[11px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-[0.1em] mb-1.5 block">{t('primary_email')}</label>
                                {isEditing ? (
                                    <>
                                        <input type="email" name="email" value={formData.email} onChange={handleChange} className={`w-full px-3 py-2.5 bg-white dark:bg-slate-900 border ${errors.email ? 'border-red-500' : 'border-slate-200 dark:border-slate-700'} rounded-xl focus:ring-2 focus:ring-indigo-500 outline-none text-sm`} />
                                        {errors.email && <p className="text-red-500 text-[10px] mt-1 ml-1">{errors.email}</p>}
                                    </>
                                ) : (
                                    <div className="flex items-center gap-3 px-3 py-2.5 bg-slate-50 dark:bg-slate-900/50 border border-slate-100 dark:border-slate-700/50 rounded-xl">
                                        <Mail className="w-4 h-4 text-slate-400" /><span className="text-sm font-medium text-slate-800 dark:text-slate-200 truncate">{formData.email}</span>
                                    </div>
                                )}
                            </div>
                            <div>
                                <label className="text-[11px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-[0.1em] mb-1.5 block">{t('contact_number')}</label>
                                {isEditing ? (
                                    <>
                                        <input type="text" name="phone" value={formData.phone} onChange={handleChange} className={`w-full px-3 py-2.5 bg-white dark:bg-slate-900 border ${errors.phone ? 'border-red-500' : 'border-slate-200 dark:border-slate-700'} rounded-xl focus:ring-2 focus:ring-indigo-500 outline-none text-sm`} />
                                        {errors.phone && <p className="text-red-500 text-[10px] mt-1 ml-1">{errors.phone}</p>}
                                    </>
                                ) : (
                                    <div className="flex items-center gap-3 px-3 py-2.5 bg-slate-50 dark:bg-slate-900/50 border border-slate-100 dark:border-slate-700/50 rounded-xl">
                                        <Smartphone className="w-4 h-4 text-slate-400" /><span className="text-sm font-medium text-slate-800 dark:text-slate-200 truncate">{formData.phone}</span>
                                    </div>
                                )}
                            </div>
                            <div>
                                <label className="text-[11px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-[0.1em] mb-1.5 block">{t('location_profile')}</label>
                                {isEditing ? (
                                    <>
                                        <input type="text" name="location" value={formData.location} onChange={handleChange} className={`w-full px-3 py-2.5 bg-white dark:bg-slate-900 border ${errors.location ? 'border-red-500' : 'border-slate-200 dark:border-slate-700'} rounded-xl focus:ring-2 focus:ring-indigo-500 outline-none text-sm`} />
                                        {errors.location && <p className="text-red-500 text-[10px] mt-1 ml-1">{errors.location}</p>}
                                    </>
                                ) : (
                                    <div className="flex items-center gap-3 px-3 py-2.5 bg-slate-50 dark:bg-slate-900/50 border border-slate-100 dark:border-slate-700/50 rounded-xl">
                                        <MapPin className="w-4 h-4 text-slate-400" /><span className="text-sm font-medium text-slate-800 dark:text-slate-200 truncate">{formData.location}</span>
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>

                    {/* Role & Permissions */}
                    <div className="bg-white dark:bg-slate-800 rounded-2xl border border-slate-200 dark:border-slate-700 p-5 md:p-6 shadow-sm">
                        <h3 className="text-[11px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-[0.12em] mb-5">{t('role_permissions')}</h3>
                        <div className="flex items-start gap-4 p-4 bg-slate-50 dark:bg-slate-900/50 rounded-xl border border-slate-100 dark:border-slate-700/50">
                            <div className="p-3 bg-indigo-100 dark:bg-indigo-900/30 rounded-xl"><Shield className="w-5 h-5 text-indigo-600 dark:text-indigo-400" /></div>
                            <div className="flex-1 min-w-0">
                                <div className="flex items-center gap-2 mb-1">
                                    <h4 className="font-semibold text-slate-900 dark:text-white capitalize">{user.role}</h4>
                                    <span className="inline-flex items-center gap-1 px-2 py-0.5 text-[10px] font-semibold bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300 rounded-full uppercase"><CheckCircle className="w-3 h-3" /> {t('verified')}</span>
                                </div>
                                <p className="text-sm text-slate-500 dark:text-slate-400 mb-3">{t('full_access_desc') || 'Full access to device management, user administration, and system configuration.'}</p>
                                <div className="grid grid-cols-2 gap-2 text-[13px]">
                                    {[t('internal_monitoring'), t('user_management'), t('system_config'), t('report_generation')].map(perm => (
                                        <div key={perm} className="flex items-center gap-2 text-slate-600 dark:text-slate-300"><div className="w-1.5 h-1.5 rounded-full bg-emerald-500"></div>{perm}</div>
                                    ))}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <div className="space-y-5">
                    {/* Appearance */}
                    <div className="bg-white dark:bg-slate-800 rounded-2xl border border-slate-200 dark:border-slate-700 p-5 shadow-sm">
                        <h3 className="text-[11px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-[0.12em] mb-4">{t('appearance')}</h3>
                        <div className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-900/50 rounded-xl mb-3">
                            <div className="flex items-center gap-3">
                                {theme === 'dark' || (theme === 'system' && isDark) ? <Moon className="w-4 h-4 text-slate-500 dark:text-slate-400" /> : <Sun className="w-4 h-4 text-amber-500" />}
                                <span className="text-sm font-medium text-slate-700 dark:text-slate-200">{t('theme')}</span>
                            </div>
                            <span className="text-xs text-slate-400 capitalize">{theme}</span>
                        </div>
                        <div className="grid grid-cols-3 gap-2">
                            {(['light', 'dark', 'system'] as const).map(tMode => (
                                <button key={tMode} onClick={() => setTheme(tMode)} className={`py-2 text-xs font-medium border rounded-lg transition-all capitalize ${theme === tMode ? 'bg-indigo-50 border-indigo-200 text-indigo-700 shadow-sm dark:bg-indigo-900/20 dark:border-indigo-700 dark:text-indigo-300' : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700/50'}`}>{tMode}</button>
                            ))}
                        </div>
                    </div>

                    {/* Security & Login */}
                    <div className="bg-white dark:bg-slate-800 rounded-2xl border border-slate-200 dark:border-slate-700 p-5 shadow-sm">
                        <h3 className="text-[11px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-[0.12em] mb-4">{t('security_login')}</h3>
                        <div className="space-y-3">
                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-3"><div className="p-2 bg-slate-100 dark:bg-slate-700 rounded-lg"><Lock className="w-4 h-4 text-slate-500 dark:text-slate-400" /></div><div><p className="text-sm font-medium text-slate-800 dark:text-white">{t('password')}</p><p className="text-[11px] text-slate-400">{t('updated_3mo')}</p></div></div>
                                <button onClick={handlePasswordReset} className="text-xs font-semibold text-indigo-600 dark:text-indigo-400 hover:underline">{t('change_password')}</button>
                            </div>
                            {user.role === 'admin' && (
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-3">
                                        <div className="p-2 bg-slate-100 dark:bg-slate-700 rounded-lg"><Smartphone className="w-4 h-4 text-slate-500 dark:text-slate-400" /></div>
                                        <div>
                                            <p className="text-sm font-medium text-slate-800 dark:text-white">{t('2fa_auth')}</p>
                                            <p className="text-[11px] text-slate-400">{totpEnabled ? t('enabled') : t('not_enabled')}</p>
                                        </div>
                                    </div>
                                    {totpEnabled ? (
                                        <button onClick={() => setShowDisable2FAModal(true)} className="text-xs font-semibold text-red-600 dark:text-red-400 hover:underline">{t('disable')}</button>
                                    ) : (
                                        <button onClick={handleSetup2FA} disabled={totpLoading} className="text-xs font-semibold text-indigo-600 dark:text-indigo-400 hover:underline disabled:opacity-50">{t('enable')}</button>
                                    )}
                                </div>
                            )}
                        </div>
                    </div>

                    {/* Telegram Integration */}
                    <div className="bg-white dark:bg-slate-800 rounded-2xl border border-slate-200 dark:border-slate-700 p-5 shadow-sm">
                        <h3 className="text-[11px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-[0.12em] mb-4">{t('telegram_notifications')}</h3>
                        <div className="space-y-3">
                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-3">
                                    <div className="p-2 bg-slate-100 dark:bg-slate-700 rounded-lg">
                                        <svg className="w-4 h-4 text-slate-500 dark:text-slate-400" viewBox="0 0 24 24" fill="currentColor">
                                            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm4.64 6.8c-.15 1.58-.8 5.42-1.13 7.19-.14.75-.42 1-.68 1.03-.58.05-1.02-.38-1.58-.75-.88-.58-1.38-.94-2.23-1.5-.99-.65-.35-1.01.22-1.59.15-.15 2.71-2.48 2.76-2.69.01-.03.01-.14-.07-.2-.08-.06-.19-.04-.27-.02-.12.03-1.99 1.27-5.62 3.72-.53.36-1.01.54-1.44.53-.47-.01-1.38-.27-2.06-.49-.83-.27-1.49-.42-1.43-.88.03-.24.37-.49 1.02-.75 3.99-1.74 6.65-2.89 7.98-3.44 3.8-1.58 4.59-1.86 5.1-1.87.11 0 .37.03.54.17.14.12.18.28.2.45-.01.06.01.24 0 .38z"/>
                                        </svg>
                                    </div>
                                    <div>
                                        <p className="text-sm font-medium text-slate-800 dark:text-white">{t('telegram_link_account')}</p>
                                        <p className="text-[11px] text-slate-400">{telegramLinked ? t('telegram_linked') : t('telegram_not_linked')}</p>
                                    </div>
                                </div>
                                {!telegramLinked ? (
                                    <button
                                        onClick={handleGenerateTelegramLink}
                                        disabled={telegramLoading}
                                        className="text-xs font-semibold text-indigo-600 dark:text-indigo-400 hover:underline disabled:opacity-50"
                                    >
                                        {telegramLoading ? t('loading') : t('telegram_generate_link')}
                                    </button>
                                ) : (
                                    <span className="text-xs font-semibold text-green-600 dark:text-green-400">{t('telegram_connected')}</span>
                                )}
                            </div>
                            
                            {telegramToken && (
                                <div className="mt-3 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
                                    <p className="text-xs text-blue-700 dark:text-blue-300 mb-2">{t('telegram_send_to_bot')}</p>
                                    <code className="block p-2 bg-white dark:bg-slate-800 rounded text-xs font-mono break-all text-blue-900 dark:text-blue-100">
                                        /start {telegramToken}
                                    </code>
                                    <p className="text-[10px] text-blue-600 dark:text-blue-400 mt-2">{t('telegram_token_expires')}</p>
                                </div>
                            )}
                            
                            {telegramLinked && (
                                <>
                                    <div className="flex items-center justify-between pt-3 border-t border-slate-200 dark:border-slate-700">
                                        <div>
                                            <p className="text-sm font-medium text-slate-800 dark:text-white">{t('telegram_alerts')}</p>
                                            <p className="text-[11px] text-slate-400">{t('telegram_alerts_desc')}</p>
                                        </div>
                                        <label className="relative inline-flex items-center cursor-pointer">
                                            <input
                                                type="checkbox"
                                                checked={telegramAlerts}
                                                onChange={(e) => handleUpdateTelegramSettings(e.target.checked, telegram2FA)}
                                                className="sr-only peer"
                                            />
                                            <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-indigo-300 dark:peer-focus:ring-indigo-800 rounded-full peer dark:bg-slate-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-slate-600 peer-checked:bg-indigo-600"></div>
                                        </label>
                                    </div>
                                    
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <p className="text-sm font-medium text-slate-800 dark:text-white">{t('telegram_2fa')}</p>
                                            <p className="text-[11px] text-slate-400">{t('telegram_2fa_desc')}</p>
                                        </div>
                                        <label className="relative inline-flex items-center cursor-pointer">
                                            <input
                                                type="checkbox"
                                                checked={telegram2FA}
                                                onChange={(e) => handleUpdateTelegramSettings(telegramAlerts, e.target.checked)}
                                                className="sr-only peer"
                                            />
                                            <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-indigo-300 dark:peer-focus:ring-indigo-800 rounded-full peer dark:bg-slate-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-slate-600 peer-checked:bg-indigo-600"></div>
                                        </label>
                                    </div>
                                </>
                            )}
                        </div>
                    </div>

                    {/* Active Sessions */}
                    <div className="bg-white dark:bg-slate-800 rounded-2xl border border-slate-200 dark:border-slate-700 p-5 shadow-sm">
                        <div className="flex items-center justify-between mb-4">
                            <h3 className="text-[11px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-[0.12em]">{t('active_sessions')}</h3>
                            {(sessions?.length ?? 0) > 1 && (
                                <button
                                    onClick={() => setShowRevokeAllConfirm(true)}
                                    className="text-xs font-semibold text-red-600 dark:text-red-400 hover:underline"
                                >
                                    {t('revoke_all')}
                                </button>
                            )}
                        </div>
                        {sessionsLoading ? (
                            <div className="text-center py-4 text-sm text-slate-400">{t('loading')}</div>
                        ) : sessions.length === 0 ? (
                            <div className="text-center py-4 text-sm text-slate-400">{t('no_data')}</div>
                        ) : (
                            <div className="space-y-3">
                                {sessions.map((session, idx) => (
                                    <div
                                        key={session.id}
                                        className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-900/50 rounded-xl border border-slate-100 dark:border-slate-700/50"
                                    >
                                        <div className="flex items-center gap-3 min-w-0 flex-1">
                                            <div className="p-2 bg-slate-100 dark:bg-slate-700 rounded-lg flex-shrink-0">
                                                <Smartphone className="w-4 h-4 text-slate-500 dark:text-slate-400" />
                                            </div>
                                            <div className="min-w-0 flex-1">
                                                <div className="flex items-center gap-2 mb-0.5">
                                                    <p className="text-sm font-medium text-slate-800 dark:text-white truncate">
                                                        {session.user_agent || 'Unknown Device'}
                                                    </p>
                                                    {idx === 0 && (
                                                        <span className="inline-flex items-center px-1.5 py-0.5 text-[9px] font-semibold bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300 rounded-full uppercase">
                                                            {t('current_session')}
                                                        </span>
                                                    )}
                                                </div>
                                                <div className="flex items-center gap-3 text-[11px] text-slate-400">
                                                    <span>{t('ip_address')}: {session.ip_address || 'N/A'}</span>
                                                    <span>{t('last_active')}: {new Date(session.created_at).toLocaleString()}</span>
                                                </div>
                                            </div>
                                        </div>
                                        {idx !== 0 && (
                                            <button
                                                onClick={() => handleRevokeSession(session.id)}
                                                className="ml-3 text-xs font-semibold text-red-600 dark:text-red-400 hover:underline flex-shrink-0"
                                            >
                                                {t('revoke')}
                                            </button>
                                        )}
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Session */}
                    <div className="bg-white dark:bg-slate-800 rounded-2xl border border-amber-200 dark:border-amber-900/50 p-5 shadow-sm">
                        <h3 className="text-[11px] font-semibold text-amber-600 dark:text-amber-400 uppercase tracking-[0.12em] mb-3 flex items-center gap-1.5"><LogOut className="w-3.5 h-3.5" /> {t('session')}</h3>
                        <p className="text-sm text-slate-500 dark:text-slate-400 mb-4">{t('session_desc')}</p>
                        <button onClick={() => setIsLogoutModalOpen(true)} className="w-full flex items-center justify-center gap-2 py-2.5 px-4 border-2 border-amber-300 dark:border-amber-700 text-amber-700 dark:text-amber-300 rounded-xl font-semibold text-sm hover:bg-amber-50 dark:hover:bg-amber-900/20 transition-colors"><LogOut className="w-4 h-4" /> {t('sign_out')}</button>
                    </div>
                </div>
            </div>

            {/* ═══ Password Change Modal ═══ */}
            <Modal
                isOpen={showPasswordModal}
                onClose={() => setShowPasswordModal(false)}
                title={t('change_password')}
                size="sm"
                footer={
                    <div className="flex justify-end gap-3">
                        <Button variant="outline" onClick={() => setShowPasswordModal(false)}>{t('cancel')}</Button>
                        <Button onClick={handleChangePassword} loading={passwordLoading}>{t('save')}</Button>
                    </div>
                }
            >
                <div className="space-y-4">
                    {passwordError && (
                        <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-600 dark:text-red-400">
                            {passwordError}
                        </div>
                    )}
                    <Input
                        label={t('current_password') || 'Current Password'}
                        type="password"
                        value={passwordForm.current}
                        onChange={(e) => setPasswordForm({ ...passwordForm, current: e.target.value })}
                        autoComplete="current-password"
                    />
                    <Input
                        label={t('new_password') || 'New Password'}
                        type="password"
                        value={passwordForm.newPass}
                        onChange={(e) => setPasswordForm({ ...passwordForm, newPass: e.target.value })}
                        autoComplete="new-password"
                    />
                    <Input
                        label={t('confirm_password') || 'Confirm New Password'}
                        type="password"
                        value={passwordForm.confirm}
                        onChange={(e) => setPasswordForm({ ...passwordForm, confirm: e.target.value })}
                        autoComplete="new-password"
                    />
                </div>
            </Modal>

            <ConfirmModal
                isOpen={isLogoutModalOpen}
                onClose={() => setIsLogoutModalOpen(false)}
                onConfirm={logout}
                title={t('sign_out')}
                message={t('sign_out_confirm')}
                confirmText={t('sign_out')}
                cancelText={t('cancel')}
            />

            <ConfirmModal
                isOpen={showRevokeAllConfirm}
                onClose={() => setShowRevokeAllConfirm(false)}
                onConfirm={handleRevokeAllOtherSessions}
                title={t('revoke_all')}
                message={t('revoke_all_confirm')}
                confirmText={t('revoke_all')}
                cancelText={t('cancel')}
            />

            {/* 2FA Setup Modal */}
            <Modal
                isOpen={show2FAModal}
                onClose={() => {
                    setShow2FAModal(false);
                    setTotpCode('');
                    setTotpError('');
                }}
                title={t('2fa_setup') || 'Setup Two-Factor Authentication'}
                size="md"
            >
                <div className="space-y-4">
                    {totpError && (
                        <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-600 dark:text-red-400">
                            {totpError}
                        </div>
                    )}
                    <div className="text-center">
                        <p className="text-sm text-slate-600 dark:text-slate-400 mb-4">
                            {t('2fa_scan_qr') || 'Scan this QR code with your authenticator app (Google Authenticator, Authy, etc.)'}
                        </p>
                        <div className="flex justify-center mb-4">
                            <div className="p-4 bg-white rounded-lg">
                                <QRCodeSVG value={totpUri} size={200} />
                            </div>
                        </div>
                        <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">
                            {t('2fa_or_enter_secret') || 'Or enter this secret manually:'}
                        </p>
                        <code className="block p-2 bg-slate-100 dark:bg-slate-700 rounded text-xs font-mono break-all">
                            {totpSecret}
                        </code>
                    </div>
                    <div className="pt-4 border-t border-slate-200 dark:border-slate-700">
                        <Input
                            label={t('2fa_enter_code') || 'Enter 6-digit code'}
                            type="text"
                            value={totpCode}
                            onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                            placeholder="123456"
                            maxLength={6}
                        />
                    </div>
                    <div className="flex justify-end gap-3 pt-4">
                        <Button variant="outline" onClick={() => {
                            setShow2FAModal(false);
                            setTotpCode('');
                            setTotpError('');
                        }}>
                            {t('cancel')}
                        </Button>
                        <Button onClick={handleVerify2FA} loading={totpLoading}>
                            {t('verify') || 'Verify'}
                        </Button>
                    </div>
                </div>
            </Modal>

            {/* Disable 2FA Modal */}
            <Modal
                isOpen={showDisable2FAModal}
                onClose={() => {
                    setShowDisable2FAModal(false);
                    setDisable2FAPassword('');
                    setTotpError('');
                }}
                title={t('2fa_disable') || 'Disable Two-Factor Authentication'}
                size="sm"
            >
                <div className="space-y-4">
                    {totpError && (
                        <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-600 dark:text-red-400">
                            {totpError}
                        </div>
                    )}
                    <p className="text-sm text-slate-600 dark:text-slate-400">
                        {t('2fa_disable_confirm') || 'Enter your password to disable 2FA'}
                    </p>
                    <Input
                        label={t('password') || 'Password'}
                        type="password"
                        value={disable2FAPassword}
                        onChange={(e) => setDisable2FAPassword(e.target.value)}
                    />
                    <div className="flex justify-end gap-3 pt-4">
                        <Button variant="outline" onClick={() => {
                            setShowDisable2FAModal(false);
                            setDisable2FAPassword('');
                            setTotpError('');
                        }}>
                            {t('cancel')}
                        </Button>
                        <Button onClick={handleDisable2FA} loading={totpLoading} variant="danger">
                            {t('disable') || 'Disable'}
                        </Button>
                    </div>
                </div>
            </Modal>
        </div>
    );
}