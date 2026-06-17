// frontend/src/context/UsersContext.tsx
import React, { createContext, useContext, useState, useEffect, ReactNode, useMemo, useCallback } from 'react';
import { api, User } from '../services/api';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../components/ui/Toast';

interface NewUserInput {
    username?: string;
    name?: string;
    email?: string;
    password: string;
    role: string;
}

interface UserUpdates {
    name?: string;
    role?: User['role'];
    status?: User['status'];
    email?: string;
}

interface UsersContextType {
    users: User[];
    loading: boolean;
    refresh: () => Promise<void>;
    addUser: (user: NewUserInput) => Promise<void>;
    updateUser: (id: string, updates: UserUpdates) => Promise<void>;
    deleteUser: (id: string) => Promise<void>;
}

const UsersContext = createContext<UsersContextType | undefined>(undefined);

function getErrorMessage(err: unknown, fallback: string): string {
    return err instanceof Error ? err.message : fallback;
}

export function UsersProvider({ children }: { children: ReactNode }) {
    const { token } = useAuth();
    const toast = useToast();
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState(false);

    const { user } = useAuth();
    
    const loadData = useCallback(async () => {
        if (!token || !user || user.role !== 'admin') return;
        setLoading(true);
        try {
            const data = await api.getUsers();
            const mappedUsers = data.map(u => {
                const raw = u as User & { last_login?: string };
                return {
                    ...raw,
                    name: raw.name || raw.username,
                    lastLogin: raw.last_login || raw.lastLogin,
                };
            });
            setUsers(mappedUsers);
        } catch (err) {
            console.error("Failed to load users", err);
        } finally {
            setLoading(false);
        }
    }, [token, user]);

    useEffect(() => {
        loadData();
    }, [loadData]);

    const addUser = useCallback(async (newUser: NewUserInput) => {
        try {
            await api.createUser({
                username: newUser.username || newUser.name || newUser.email || '',
                password: newUser.password,
                role: newUser.role,
                email: newUser.email,
            });
            await loadData();
            toast.success("Пользователь создан");
        } catch (err: unknown) {
            toast.error(getErrorMessage(err, "Ошибка создания"));
            throw err;
        }
    }, [loadData, toast]);

    const updateUser = useCallback(async (id: string, updates: UserUpdates) => {
        try {
            await api.updateUser(id, {
                name: updates.name,
                role: updates.role,
                status: updates.status,
                email: updates.email,
            });
            await loadData();
            toast.success("Данные обновлены");
        } catch (err: unknown) {
            toast.error(getErrorMessage(err, "Ошибка обновления"));
            throw err;
        }
    }, [loadData, toast]);

    const deleteUser = useCallback(async (id: string) => {
        try {
            await api.deleteUser(id);
            await loadData();
            toast.success("Пользователь удален");
        } catch (err: unknown) {
            toast.error(getErrorMessage(err, "Ошибка удаления"));
            throw err;
        }
    }, [loadData, toast]);

    const value = useMemo<UsersContextType>(() => ({
        users, loading, refresh: loadData, addUser, updateUser, deleteUser,
    }), [users, loading, loadData, addUser, updateUser, deleteUser]);

    return (
        <UsersContext.Provider value={value}>
            {children}
        </UsersContext.Provider>
    );
}

// eslint-disable-next-line react-refresh/only-export-components
export function useUsers() {
    const context = useContext(UsersContext);
    if (context === undefined) {
        throw new Error('useUsers must be used within a UsersProvider');
    }
    return context;
}