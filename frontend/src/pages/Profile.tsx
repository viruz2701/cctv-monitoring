import React, { useState, useEffect } from 'react';
import { z } from 'zod';
import { User, Mail, Shield, Smartphone, Moon, Sun, Lock, LogOut, Camera, Save, X, Edit2, MapPin, Briefcase, Trash2, Clock, Calendar, CheckCircle } from 'lucide-react';
import { Card, CardHeader, CardBody, Button, useToast, ConfirmModal } from '../components/ui';
import { useAuth } from '../hooks/useAuth';
import { useTheme } from '../context/ThemeContext';
import { useTranslation } from 'react-i18next';

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

    if (!user) return null;

    const handlePasswordReset = () => {
        toast.info(t('password_reset_soon') || 'Password reset functionality coming soon');
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
                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-3"><div className="p-2 bg-slate-100 dark:bg-slate-700 rounded-lg"><Smartphone className="w-4 h-4 text-slate-500 dark:text-slate-400" /></div><div><p className="text-sm font-medium text-slate-800 dark:text-white">{t('2fa_auth')}</p><p className="text-[11px] text-slate-400">{t('not_enabled')}</p></div></div>
                                <button onClick={() => toast.info('2FA setup coming soon')} className="text-xs font-semibold text-indigo-600 dark:text-indigo-400 hover:underline">{t('enable')}</button>
                            </div>
                        </div>
                    </div>

                    {/* Session */}
                    <div className="bg-white dark:bg-slate-800 rounded-2xl border border-amber-200 dark:border-amber-900/50 p-5 shadow-sm">
                        <h3 className="text-[11px] font-semibold text-amber-600 dark:text-amber-400 uppercase tracking-[0.12em] mb-3 flex items-center gap-1.5"><LogOut className="w-3.5 h-3.5" /> {t('session')}</h3>
                        <p className="text-sm text-slate-500 dark:text-slate-400 mb-4">{t('session_desc')}</p>
                        <button onClick={() => setIsLogoutModalOpen(true)} className="w-full flex items-center justify-center gap-2 py-2.5 px-4 border-2 border-amber-300 dark:border-amber-700 text-amber-700 dark:text-amber-300 rounded-xl font-semibold text-sm hover:bg-amber-50 dark:hover:bg-amber-900/20 transition-colors"><LogOut className="w-4 h-4" /> {t('sign_out')}</button>
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
        </div>
    );
}