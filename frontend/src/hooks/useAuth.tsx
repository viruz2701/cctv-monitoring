// ═══════════════════════════════════════════════════════════════════════
// Auth Provider + Hook
// ARCH.1: Использует Zustand authStore для состояния.
// AuthProvider — только инициализация (fetch /users/me при монтировании).
// Новый код ДОЛЖЕН импортировать useAuthStore напрямую из '../store'.
// ═══════════════════════════════════════════════════════════════════════

import { createContext, useContext, useEffect, ReactNode, useCallback } from 'react';
import { useAuthStore, type AuthUser } from '../store/authStore';

// P1-SEC.1: Читаем CSRF токен из cookie для отправки в заголовке.
function getCSRFCookie(): string | null {
    if (typeof document === 'undefined') return null;
    const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
    return match ? match[1] : null;
}

// P1-SEC.1: Вызов logout на backend — очищает HttpOnly cookies.
async function callLogoutAPI(): Promise<void> {
    try {
        await fetch('/api/v1/auth/logout', {
            method: 'POST',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': getCSRFCookie() || '',
            },
        });
    } catch {
        // Ignore network errors on logout — cookie will be cleared client-side
    }
}

// ─── Auth Context (backward compat) ─────────────────────────────────

interface AuthContextType {
    user: AuthUser | null;
    token: string | null;
    login: (username: string, password: string) => Promise<{ requires2FA?: boolean; sessionToken?: string }>;
    login2FA: (sessionToken: string, code: string) => Promise<void>;
    logout: () => void;
    hasPermission: (roles: string | string[]) => boolean;
    updateUser: (data: Partial<AuthUser>) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
    const user = useAuthStore((s) => s.user);
    const token = useAuthStore((s) => s.token);
    const setUser = useAuthStore((s) => s.setUser);
    const setToken = useAuthStore((s) => s.setToken);
    const setLoading = useAuthStore((s) => s.setLoading);
    const setInitialized = useAuthStore((s) => s.setInitialized);
    const storeLogout = useAuthStore((s) => s.logout);
    const storeUpdateUser = useAuthStore((s) => s.updateUser);
    const storeHasPermission = useAuthStore((s) => s.hasPermission);

    // P1-SEC.1: При загрузке проверяем, аутентифицирован ли пользователь,
    // вызывая /users/me (JWT передаётся через HttpOnly cookie).
    useEffect(() => {
        let cancelled = false;

        fetch('/api/v1/users/me', {
            credentials: 'include',
            headers: {
                'X-CSRF-Token': getCSRFCookie() || '',
            },
        })
            .then((res) => {
                if (!res.ok) throw new Error('Not authenticated');
                return res.json();
            })
            .then((data) => {
                if (cancelled) return;
                const mapped: AuthUser = {
                    id: data.id,
                    username: data.username,
                    role: data.role,
                    owner_id: data.owner_id,
                    name: data.username,
                    email: data.username,
                    avatar: data.avatar || '',
                    sites: data.sites || [],
                };
                setUser(mapped);
                setLoading(false);
                setInitialized(true);
            })
            .catch(() => {
                if (!cancelled) {
                    setLoading(false);
                    setInitialized(true);
                }
            });

        return () => {
            cancelled = true;
        };
    }, [setUser, setLoading, setInitialized]);

    const login = useCallback(
        async (username: string, password: string): Promise<{ requires2FA?: boolean; sessionToken?: string }> => {
            const response = await fetch('/api/v1/auth/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password }),
                credentials: 'include',
            });

            if (response.status === 202) {
                // 2FA required
                const data = await response.json();
                return { requires2FA: true, sessionToken: data.session_token };
            }

            if (!response.ok) {
                const body = await response.text();
                try {
                    const parsed = JSON.parse(body);
                    const msg = parsed?.error?.message;
                    if (msg && typeof msg === 'string') {
                        throw new Error(msg);
                    }
                } catch (e) {
                    if (e instanceof Error) throw e;
                }
                throw new Error(body || 'Login failed');
            }

            // P1-SEC.1: JWT уже в HttpOnly cookie, не извлекаем из body.
            const { user: userData } = await response.json();
            setUser({
                id: userData.id,
                username: userData.username,
                role: userData.role,
                owner_id: userData.owner_id,
                name: userData.username,
                email: userData.username,
                avatar: userData.avatar || '',
                sites: userData.sites || [],
            });
            return {};
        },
        [setUser],
    );

    const login2FAFn = useCallback(
        async (sessionToken: string, code: string) => {
            const response = await fetch('/api/v1/auth/login/2fa', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ session_token: sessionToken, code }),
                credentials: 'include',
            });

            if (!response.ok) {
                const err = await response.text();
                throw new Error(err || '2FA verification failed');
            }

            // P1-SEC.1: JWT уже в HttpOnly cookie.
            const { user: userData } = await response.json();
            setUser({
                id: userData.id,
                username: userData.username,
                role: userData.role,
                owner_id: userData.owner_id,
                name: userData.username,
                email: userData.username,
                avatar: userData.avatar || '',
                sites: userData.sites || [],
            });
        },
        [setUser],
    );

    const logout = useCallback(async () => {
        await callLogoutAPI();
        storeLogout();
    }, [storeLogout]);

    const hasPermission = useCallback(
        (roles: string | string[]) => storeHasPermission(roles),
        [storeHasPermission],
    );

    const updateUser = useCallback(
        (data: Partial<AuthUser>) => storeUpdateUser(data),
        [storeUpdateUser],
    );

    return (
        <AuthContext.Provider value={{ user, token, login, login2FA: login2FAFn, logout, hasPermission, updateUser }}>
            {children}
        </AuthContext.Provider>
    );
}

export function useAuth() {
    const ctx = useContext(AuthContext);
    if (!ctx) throw new Error('useAuth must be used within AuthProvider');
    return ctx;
}
