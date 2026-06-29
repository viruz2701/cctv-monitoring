import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, Mail, Lock } from '../components/ui/Icons';
import { Button } from '../components/ui';
import { useTranslation } from 'react-i18next';

type Step = 'email' | 'reset';

export function ForgotPassword() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const [step, setStep] = useState<Step>('email');
    const [email, setEmail] = useState('');
    const [resetToken, setResetToken] = useState('');
    const [newPassword, setNewPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');

    const handleEmailSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError('');
        setSuccess('');
        setLoading(true);

        if (!email) {
            setError(t('email_required') || 'Email is required');
            setLoading(false);
            return;
        }

        try {
            const response = await fetch('/api/v1/auth/forgot-password', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email }),
            });

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || 'Failed to process request');
            }

            // В dev-режиме токен возвращается в ответе
            if (data.reset_token) {
                setResetToken(data.reset_token);
                setSuccess(t('reset_token_generated') || 'Reset token generated. Check console for details.');
                setStep('reset');
            } else {
                setSuccess(t('reset_email_sent') || 'If the email exists, a reset link has been sent.');
            }
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unknown error';
            setError(message || t('reset_request_failed') || 'Failed to process request');
        } finally {
            setLoading(false);
        }
    };

    const handleResetSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError('');
        setLoading(true);

        if (!resetToken) {
            setError(t('token_required') || 'Reset token is required');
            setLoading(false);
            return;
        }

        if (!newPassword || !confirmPassword) {
            setError(t('all_fields_required') || 'All fields are required');
            setLoading(false);
            return;
        }

        if (newPassword.length < 6) {
            setError(t('password_min_length') || 'Password must be at least 6 characters');
            setLoading(false);
            return;
        }

        if (newPassword !== confirmPassword) {
            setError(t('passwords_do_not_match') || 'Passwords do not match');
            setLoading(false);
            return;
        }

        try {
            const response = await fetch('/api/v1/auth/reset-password', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ token: resetToken, new_password: newPassword }),
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || 'Failed to reset password');
            }

            setSuccess(t('password_reset_success') || 'Password reset successfully. You can now log in.');
            setTimeout(() => navigate('/login'), 2000);
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unknown error';
            setError(message || t('reset_failed') || 'Failed to reset password');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen flex items-center justify-center p-8 bg-slate-50 dark:bg-slate-900">
            <div className="w-full max-w-md">
                <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-8 border border-slate-200 dark:border-slate-700/50">
                    <div className="text-center mb-8">
                        <h2 className="text-2xl font-bold text-slate-900 dark:text-white">
                            {step === 'email' ? t('forgot_password') : t('reset_password')}
                        </h2>
                        <p className="text-slate-500 dark:text-slate-400 mt-2">
                            {step === 'email'
                                ? t('forgot_password_desc') || 'Enter your email to receive a reset token'
                                : t('enter_new_password') || 'Enter your new password'}
                        </p>
                    </div>

                    {error && (
                        <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/10 border border-red-200 dark:border-red-800/50 rounded-lg text-sm text-red-600 dark:text-red-400">
                            {error}
                        </div>
                    )}

                    {success && (
                        <div className="mb-6 p-4 bg-green-50 dark:bg-green-900/10 border border-green-200 dark:border-green-800/50 rounded-lg text-sm text-green-600 dark:text-green-400">
                            {success}
                        </div>
                    )}

                    {step === 'email' ? (
                        <form onSubmit={handleEmailSubmit} className="space-y-5">
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                                    {t('email')}
                                </label>
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
                            <Button type="submit" fullWidth size="lg" loading={loading}>
                                {t('send_reset_token') || 'Send Reset Token'}
                            </Button>
                        </form>
                    ) : (
                        <form onSubmit={handleResetSubmit} className="space-y-5">
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                                    {t('reset_token') || 'Reset Token'}
                                </label>
                                <input
                                    type="text"
                                    value={resetToken}
                                    onChange={e => setResetToken(e.target.value)}
                                    placeholder="Enter reset token"
                                    className="w-full px-4 py-3 text-sm border border-slate-300 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-400 font-mono"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                                    {t('new_password')}
                                </label>
                                <div className="relative">
                                    <Lock className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
                                    <input
                                        type="password"
                                        value={newPassword}
                                        onChange={e => setNewPassword(e.target.value)}
                                        placeholder="••••••••"
                                        className="w-full pl-12 pr-4 py-3 text-sm border border-slate-300 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-400"
                                    />
                                </div>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                                    {t('confirm_password')}
                                </label>
                                <div className="relative">
                                    <Lock className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
                                    <input
                                        type="password"
                                        value={confirmPassword}
                                        onChange={e => setConfirmPassword(e.target.value)}
                                        placeholder="••••••••"
                                        className="w-full pl-12 pr-4 py-3 text-sm border border-slate-300 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-400"
                                    />
                                </div>
                            </div>
                            <Button type="submit" fullWidth size="lg" loading={loading}>
                                {t('reset_password_btn') || 'Reset Password'}
                            </Button>
                        </form>
                    )}

                    <div className="mt-6 text-center">
                        <button
                            type="button"
                            onClick={() => navigate('/login')}
                            className="inline-flex items-center gap-2 text-sm text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300"
                        >
                            <ArrowLeft className="w-4 h-4" />
                            {t('back_to_login') || 'Back to Login'}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
}
