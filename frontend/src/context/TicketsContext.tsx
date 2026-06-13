import React, { createContext, useContext, useState, ReactNode, useMemo, useCallback } from 'react';
import { tickets as initialTickets } from '../data/mockData';
import type { Ticket, TicketComment } from '../types';
import { useAuth } from '../hooks/useAuth';
import { useDevicesSites } from './DevicesSitesContext';

interface TicketsContextType {
    tickets: Ticket[];
    addTicket: (ticket: Ticket) => void;
    updateTicket: (id: string, updates: Partial<Ticket>) => void;
    deleteTicket: (id: string) => void;
    addTicketComment: (ticketId: string, comment: TicketComment) => void;
}

const TicketsContext = createContext<TicketsContextType | undefined>(undefined);

export function TicketsProvider({ children }: { children: ReactNode }) {
    const { user } = useAuth();
    const { devices } = useDevicesSites();

    // Raw State
    const [rawTickets, setRawTickets] = useState<Ticket[]>(initialTickets);

    // 3. Visible Tickets (linked to visible devices)
    const tickets = useMemo(() => {
        if (!user) return [];
        if (user.role === 'admin') return rawTickets;
        const visibleDeviceIds = devices.map(d => d.id);
        return rawTickets.filter(ticket => visibleDeviceIds.includes(ticket.deviceId));
    }, [user, rawTickets, devices]);

    // Ticket Actions
    const addTicket = useCallback((ticket: Ticket) => {
        setRawTickets(prev => [ticket, ...prev]);
    }, []);

    const updateTicket = useCallback((id: string, updates: Partial<Ticket>) => {
        setRawTickets(prev => prev.map(t => t.id === id ? { ...t, ...updates, updatedAt: new Date().toISOString() } : t));
    }, []);

    const deleteTicket = useCallback((id: string) => {
        setRawTickets(prev => prev.filter(t => t.id !== id));
    }, []);

    const addTicketComment = useCallback((ticketId: string, comment: TicketComment) => {
        setRawTickets(prev => prev.map(t => {
            if (t.id === ticketId) {
                return {
                    ...t,
                    comments: [...(t.comments || []), comment],
                    updatedAt: new Date().toISOString()
                };
            }
            return t;
        }));
    }, []);

    const value = useMemo<TicketsContextType>(() => ({
        tickets, addTicket, updateTicket, deleteTicket, addTicketComment,
    }), [tickets, addTicket, updateTicket, deleteTicket, addTicketComment]);

    return (
        <TicketsContext.Provider value={value}>
            {children}
        </TicketsContext.Provider>
    );
}

export function useTickets() {
    const context = useContext(TicketsContext);
    if (context === undefined) {
        throw new Error('useTickets must be used within a TicketsProvider');
    }
    return context;
}
