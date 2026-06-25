// ═══════════════════════════════════════════════════════════════════════
// UsersContext — Bridge to React Query (ARCH-02/03)
//
// Вместо useState + useEffect использует React Query.
// После полной миграции: удалить этот файл.
// ═══════════════════════════════════════════════════════════════════════

import React, { createContext, useContext, ReactNode, useMemo, useCallback } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../components/ui/Toast';
import {
    useUsers as useUsersQuery,
    useCreateUser,
    useUpdateUser,
    useDeleteUser,
} from '../hooks/useApiQuery';
import type { User } from '../services/api';

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
    const { user } = useAuth();
    const toast = useToast();

    // React Query hooks
    const { data: rawUsers = [], isFetching, refetch } = useUsersQuery();
    const createUserMut = useCreateUser();
    const updateUserMut = useUpdateUser();
    const deleteUserMut = useDeleteUser();

    // Map API response — нормализация полей (name fallback, snake_case → camelCase)
    const users = useMemo(() => {
        return rawUsers.map(u => {
            const raw = u as User & { last_login?: string };
            return {
                ...raw,
                name: raw.name || raw.username,
                lastLogin: raw.last_login || raw.lastLogin,
            };
        });
    }, [rawUsers]);

    const addUser = useCallback(async (newUser: NewUserInput) => {
        try {
            await createUserMut.mutateAsync({
                username: newUser.username || newUser.name || newUser.email || '',
                password: newUser.password,
                role: newUser.role,
                email: newUser.email,
            });
            toast.success("Пользователь создан");
        } catch (err: unknown) {
            toast.error(getErrorMessage(err, "Ошибка создания"));
            throw err;
        }
    }, [createUserMut, toast]);

    const updateUser = useCallback(async (id: string, updates: UserUpdates) => {
        try {
            await updateUserMut.mutateAsync({ id, updates });
            toast.success("Данные обновлены");
        } catch (err: unknown) {
            toast.error(getErrorMessage(err, "Ошибка обновления"));
            throw err;
        }
    }, [updateUserMut, toast]);

    const deleteUser = useCallback(async (id: string) => {
        try {
            await deleteUserMut.mutateAsync(id);
            toast.success("Пользователь удален");
        } catch (err: unknown) {
            toast.error(getErrorMessage(err, "Ошибка удаления"));
            throw err;
        }
    }, [deleteUserMut, toast]);

    const value = useMemo<UsersContextType>(() => ({
        users,
        loading: isFetching,
        refresh: async () => { await refetch(); },
        addUser,
        updateUser,
        deleteUser,
    }), [users, isFetching, refetch, addUser, updateUser, deleteUser]);

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