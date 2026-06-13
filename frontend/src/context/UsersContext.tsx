import React, { createContext, useContext, useState, ReactNode, useMemo, useCallback } from 'react';
import { users as initialUsers } from '../data/mockData';
import type { User } from '../types';

interface UsersContextType {
    users: User[];
    addUser: (user: User) => void;
    updateUser: (id: string, updates: Partial<User>) => void;
    deleteUser: (id: string) => void;
}

const UsersContext = createContext<UsersContextType | undefined>(undefined);

export function UsersProvider({ children }: { children: ReactNode }) {
    const [users, setUsers] = useState<User[]>(initialUsers);

    const addUser = useCallback((newUser: User) => {
        setUsers(prev => [...prev, newUser]);
    }, []);

    const updateUser = useCallback((id: string, updates: Partial<User>) => {
        setUsers(prev => prev.map(u => u.id === id ? { ...u, ...updates } : u));
    }, []);

    const deleteUser = useCallback((id: string) => {
        setUsers(prev => prev.filter(u => u.id !== id));
    }, []);

    const value = useMemo<UsersContextType>(() => ({
        users, addUser, updateUser, deleteUser,
    }), [users, addUser, updateUser, deleteUser]);

    return (
        <UsersContext.Provider value={value}>
            {children}
        </UsersContext.Provider>
    );
}

export function useUsers() {
    const context = useContext(UsersContext);
    if (context === undefined) {
        throw new Error('useUsers must be used within a UsersProvider');
    }
    return context;
}
