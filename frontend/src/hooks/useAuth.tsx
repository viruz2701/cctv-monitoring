import { useState, createContext, useContext, useEffect, ReactNode } from 'react';
import { api, setAuthToken } from '../services/api';

export interface AuthUser {
    id: string;
    username: string;
    role: 'admin' | 'support' | 'owner' | 'manager' | 'technician' | 'viewer';
    owner_id?: string | null;
    name?: string;
    email?: string;
    avatar?: string;
    sites?: string[];
}

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
    const [user, setUser] = useState<AuthUser | null>(null);
    const [token, setToken] = useState<string | null>(localStorage.getItem('token'));

    useEffect(() => {
        if (token) {
            api.getCurrentUser()
                .then(data => {
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
                })
                .catch(() => logout());
        }
    }, [token]);

   const login = async (username: string, password: string): Promise<{ requires2FA?: boolean; sessionToken?: string }> => {
    try {
        const response = await fetch('/api/v1/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
        });

        if (response.status === 202) {
            // 2FA required
            const data = await response.json();
            return { requires2FA: true, sessionToken: data.session_token };
        }

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || 'Login failed');
        }

        const { token: newToken, user: userData } = await response.json();
        setAuthToken(newToken);
        setToken(newToken);
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
    } catch (err: any) {
        throw err;
    }
};

    const login2FAFn = async (sessionToken: string, code: string) => {
        const { token: newToken, user: userData } = await api.login2FA(sessionToken, code);
        setAuthToken(newToken);
        setToken(newToken);
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
    };

    const logout = () => {
        setAuthToken(null);
        setToken(null);
        setUser(null);
    };

    const hasPermission = (roles: string | string[]) => {
        if (!user) return false;
        if (user.role === 'admin') return true;
        const allowed = Array.isArray(roles) ? roles : [roles];
        return allowed.includes(user.role);
    };

    const updateUser = (data: Partial<AuthUser>) => {
        if (user) setUser({ ...user, ...data });
    };

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