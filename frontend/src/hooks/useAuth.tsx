import { useState, createContext, useContext, useEffect, ReactNode } from 'react';
import { api, setAuthToken } from '../services/api';

export interface AuthUser {
    id: string;
    username: string;
    role: 'admin' | 'support' | 'owner';
    owner_id?: string | null;
    name?: string;
    email?: string;
    avatar?: string;
    sites?: string[];
}

interface AuthContextType {
    user: AuthUser | null;
    token: string | null;
    login: (username: string, password: string) => Promise<void>;
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

   const login = async (username: string, password: string) => {
    console.log('login start');
    const { token: newToken, user: userData } = await api.login(username, password);
    console.log('got token', newToken);
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
    console.log('user set', userData.username);
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
        <AuthContext.Provider value={{ user, token, login, logout, hasPermission, updateUser }}>
            {children}
        </AuthContext.Provider>
    );
}

export function useAuth() {
    const ctx = useContext(AuthContext);
    if (!ctx) throw new Error('useAuth must be used within AuthProvider');
    return ctx;
}