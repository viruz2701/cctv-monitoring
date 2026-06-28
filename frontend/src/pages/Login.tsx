import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Camera, Eye, EyeOff, Lock, Mail, Shield } from 'lucide-react';
import { Button } from '../components/ui';
import { useAuth } from '../hooks/useAuth';
import { useSettings } from '../context/SettingsContext';
import { useTranslation } from 'react-i18next';

export function Login() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const { login, login2FA } = useAuth();
    const { settings } = useSettings();

    const [step, setStep] = useState<'credentials' | '2fa'>('credentials');
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [otp, setOtp] = useState('');
    const [sessionToken, setSessionToken] = useState('');

    const [showPassword, setShowPassword] = useState(false);
    const [rememberMe, setRememberMe] = useState(false);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const validatePassword = (pwd: string) => {
        if (settings.security.passwordPolicy === 'strong') {
            const strongRegex = /^(?=.*[0-9!@#$%^&*])(?=.{8,})/;
            return strongRegex.test(pwd);
        }
        return pwd.length >= 6;
    };

    const handleCredentialsSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError('');
        setLoading(true);

        if (!email || !password) {
            setError(t('login_error_empty') || 'Please enter your email and password');
            setLoading(false);
            return;
        }

        if (!validatePassword(password)) {
            const msg = settings.security.passwordPolicy === 'strong'
                ? (t('login_error_password_strong') || 'Password must be at least 8 characters and contain a number or symbol.')
                : (t('login_error_password_basic') || 'Password must be at least 6 characters.');
            setError(msg);
            setLoading(false);
            return;
        }

        try {
            const result = await login(email, password);
            
            if (result.requires2FA && result.sessionToken) {
                // 2FA required - show OTP input
                setSessionToken(result.sessionToken);
                setStep('2fa');
                setLoading(false);
                return;
            }

            // No 2FA required - login successful
            navigate('/dashboard');
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : String(err);
            setError(message || t('login_failed') || 'Login failed');
        } finally {
            setLoading(false);
        }
    };

    const handleOtpSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError('');
        setLoading(true);

        if (!otp || otp.length !== 6) {
            setError(t('login_error_otp_invalid') || 'Please enter a valid 6-digit code');
            setLoading(false);
            return;
        }

        try {
            await login2FA(sessionToken, otp);
            navigate('/dashboard');
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : String(err);
            setError(message || t('login_error_otp') || 'Invalid authentication code');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen flex">
            {/* Левая панель (брендинг) – статические тексты, можно не переводить */}
            <div className="hidden lg:flex lg:w-1/2 bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900 p-12 flex-col justify-between">
                <div className="flex items-center gap-3">
                    <div className="flex items-center justify-center w-12 h-12 bg-blue-600 rounded-xl">
                        <Camera className="w-6 h-6 text-white" />
                    </div>
                    <div>
                        <h1 className="text-xl font-bold text-white">CCTV Monitor</h1>
                        <p className="text-sm text-blue-300">Health Dashboard</p>
                    </div>
                </div>

                <div className="space-y-6">
                    <h2 className="text-4xl font-bold text-white leading-tight">
                        Enterprise-grade CCTV<br />
                        <span className="text-blue-400">Health Monitoring</span>
                    </h2>
                    <p className="text-lg text-slate-300 max-w-md">
                        {step === '2fa'
                            ? t('2fa_description') || "Secure your account with Two-Factor Authentication."
                            : t('login_description') || "Monitor device health, manage tickets, and ensure your surveillance infrastructure runs smoothly 24/7."}
                    </p>

                    {step === 'credentials' && (
                        <div className="flex gap-8">
                            <div><p className="text-3xl font-bold text-white">248+</p><p className="text-sm text-slate-400">{t('devices_monitored') || "Devices Monitored"}</p></div>
                            <div><p className="text-3xl font-bold text-white">99.5%</p><p className="text-sm text-slate-400">{t('uptime') || "Uptime"}</p></div>
                            <div><p className="text-3xl font-bold text-white">5</p><p className="text-sm text-slate-400">{t('sites') || "Sites"}</p></div>
                        </div>
                    )}
                </div>

                <p className="text-sm text-slate-500 dark:text-slate-400">{t('copyright') || "© 2026 CCTV Health Monitor. All rights reserved."}</p>
            </div>

            {/* Правая панель – форма входа (полностью локализована) */}
            <div className="flex-1 flex items-center justify-center p-8 bg-slate-50 dark:bg-slate-900">
                <div className="w-full max-w-md">
                    <div className="lg:hidden flex items-center justify-center gap-3 mb-8">
                        <div className="flex items-center justify-center w-12 h-12 bg-blue-600 rounded-xl">
                            <Camera className="w-6 h-6 text-white" />
                        </div>
                        <div>
                            <h1 className="text-xl font-bold text-slate-900 dark:text-white">CCTV Monitor</h1>
                            <p className="text-sm text-slate-500 dark:text-slate-400">Health Dashboard</p>
                        </div>
                    </div>
                    <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-8 border border-slate-200 dark:border-slate-700/50">
                        <div className="text-center mb-8">
                            <h2 className="text-2xl font-bold text-slate-900 dark:text-white">
                                {step === '2fa' ? t('2fa_title') || 'Two-Factor Authentication' : t('welcome_back')}
                            </h2>
                            <p className="text-slate-500 dark:text-slate-400 mt-2">
                                {step === '2fa'
                                    ? t('2fa_instruction') || 'Enter the 6-digit code from your authenticator app'
                                    : t('sign_in')}
                            </p>
                        </div>

                        {error && (
                            <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/10 border border-red-200 dark:border-red-800/50 rounded-lg text-sm text-red-600 dark:text-red-400">
                                {error}
                            </div>
                        )}

                        {step === 'credentials' ? (
                            <form onSubmit={handleCredentialsSubmit} className="space-y-5">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">{t('email')}</label>
                                    <div className="relative">
                                        <Mail className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
                                        <input
                                            type="email"
                                            value={email}
                                            onChange={e => setEmail(e.target.value)}
                                            placeholder="admin@example.com"
                                            className="w-full pl-12 pr-4 py-3 text-sm border border-slate-300 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-400"
                                        />
                                    </div>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">{t('password')}</label>
                                    <div className="relative">
                                        <Lock className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
                                        <input
                                            type={showPassword ? 'text' : 'password'}
                                            value={password}
                                            onChange={e => setPassword(e.target.value)}
                                            placeholder="••••••••"
                                            className="w-full pl-12 pr-12 py-3 text-sm border border-slate-300 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-400"
                                        />
                                        <button
                                            type="button"
                                            onClick={() => setShowPassword(!showPassword)}
                                            className="absolute right-3.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
                                        >
                                            {showPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                                        </button>
                                    </div>
                                </div>
                                <div className="flex items-center justify-between">
                                    <label className="flex items-center gap-2 cursor-pointer">
                                        <input
                                            type="checkbox"
                                            checked={rememberMe}
                                            onChange={e => setRememberMe(e.target.checked)}
                                            className="w-4 h-4 text-blue-600 border-slate-300 dark:border-slate-600 rounded focus:ring-blue-500 dark:bg-slate-900/50"
                                        />
                                        <span className="text-sm text-slate-600 dark:text-slate-300">{t('remember_me')}</span>
                                    </label>
                                    <a href="/forgot-password" className="text-sm font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300">
                                        {t('forgot_password')}
                                    </a>
                                </div>
                                <Button type="submit" fullWidth size="lg" loading={loading}>
                                    {t('login')}
                                </Button>
                            </form>
                        ) : (
                            <form onSubmit={handleOtpSubmit} className="space-y-5">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">{t('2fa_code') || 'Authentication Code'}</label>
                                    <div className="relative">
                                        <Shield className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
                                        <input
                                            type="text"
                                            value={otp}
                                            onChange={e => { if (e.target.value.length <= 6 && /^\d*$/.test(e.target.value)) setOtp(e.target.value); }}
                                            placeholder="000000"
                                            className="w-full pl-12 pr-4 py-3 text-lg tracking-widest border border-slate-300 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-400 text-center font-mono"
                                            autoFocus
                                        />
                                    </div>
                                    <p className="text-xs text-center text-slate-500 mt-2">{t('2fa_example') || 'Enter code: 123456'}</p>
                                </div>
                                <Button type="submit" fullWidth size="lg" loading={loading}>
                                    {t('2fa_verify') || 'Verify Code'}
                                </Button>
                                <button
                                    type="button"
                                    onClick={() => { setStep('credentials'); setOtp(''); setError(''); }}
                                    className="w-full text-sm text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300"
                                >
                                    {t('back_to_login') || 'Back to Login'}
                                </button>
                            </form>
                        )}
                        <div className="mt-6 text-center">
                            <p className="text-sm text-slate-500 dark:text-slate-400">{t('demo_creds')}</p>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}