// ═══════════════════════════════════════════════════════════════════════
// TicketsContext — Bridge to React Query (ARCH-02/03)
//
// Вместо mockData использует API через React Query.
// После полной миграции: удалить этот файл.
// ═══════════════════════════════════════════════════════════════════════

import React, { createContext, useContext, ReactNode, useMemo, useCallback } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useDevicesSites } from './DevicesSitesContext';
import { useTickets as useTicketsQuery, useCreateTicket, useUpdateTicket, useDeleteTicket } from '../hooks/useApiQuery';
import type { Ticket as APITicket } from '../services/api';
import type { Ticket, TicketComment } from '../types';

interface TicketsContextType {
    tickets: Ticket[];
    addTicket: (ticket: Ticket) => void;
    updateTicket: (id: string, updates: Partial<Ticket>) => void;
    deleteTicket: (id: string) => void;
    addTicketComment: (ticketId: string, comment: TicketComment) => void;
}

const TicketsContext = createContext<TicketsContextType | undefined>(undefined);

// ═══ Helper: API Ticket → UI Ticket ═══
function mapAPITicketToUI(t: APITicket): Ticket {
    return {
        id: t.id,
        title: t.title,
        description: t.description,
        deviceId: t.device_id || '',
        deviceName: '',
        siteName: '',
        priority: (t.priority as Ticket['priority']) || 'medium',
        status: (t.status as Ticket['status']) || 'open',
        assignee: t.assignee || '',
        createdAt: t.created_at,
        updatedAt: t.updated_at,
        comments: t.comments?.map((c: any) => ({
            id: c.id,
            ticketId: c.ticket_id,
            userId: c.user_id,
            userName: c.user_name || '',
            content: c.content,
            createdAt: c.created_at,
        })) || [],
    };
}

export function TicketsProvider({ children }: { children: ReactNode }) {
    const { user } = useAuth();
    const { devices } = useDevicesSites();

    // React Query hooks
    const { data: apiTickets = [] } = useTicketsQuery();
    const createTicket = useCreateTicket();
    const updateTicketMut = useUpdateTicket();
    const deleteTicketMut = useDeleteTicket();

    // Map API → UI + role-based filtering
    const tickets = useMemo(() => {
        const mapped = apiTickets.map(mapAPITicketToUI);
        if (!user) return [];
        if (user.role === 'admin') return mapped;
        const visibleDeviceIds = devices.map(d => d.id);
        return mapped.filter(ticket => visibleDeviceIds.includes(ticket.deviceId));
    }, [apiTickets, user, devices]);

    const addTicket = useCallback(async (ticket: Ticket) => {
        try {
            await createTicket.mutateAsync({
                title: ticket.title,
                description: ticket.description,
                device_id: ticket.deviceId,
                priority: ticket.priority,
                status: ticket.status,
            });
        } catch (err) {
            console.error('Failed to create ticket:', err);
        }
    }, [createTicket]);

    const updateTicket = useCallback(async (id: string, updates: Partial<Ticket>) => {
        try {
            await updateTicketMut.mutateAsync({
                id,
                updates: {
                    title: updates.title,
                    description: updates.description,
                    priority: updates.priority,
                    status: updates.status,
                    assignee: updates.assignee,
                },
            });
        } catch (err) {
            console.error('Failed to update ticket:', err);
        }
    }, [updateTicketMut]);

    const deleteTicket = useCallback(async (id: string) => {
        try {
            await deleteTicketMut.mutateAsync(id);
        } catch (err) {
            console.error('Failed to delete ticket:', err);
        }
    }, [deleteTicketMut]);

    const addTicketComment = useCallback(async (ticketId: string, comment: TicketComment) => {
        try {
            const { api } = await import('../services/api');
            await api.addTicketComment(ticketId, comment.content);
        } catch (err) {
            console.error('Failed to add comment:', err);
        }
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
