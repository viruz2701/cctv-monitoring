// ═══════════════════════════════════════════════════════════════════════
// Sites & Tickets API
// ARCH.2: Выделен из monolithic api.ts.
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface Site {
  id: string;
  name: string;
  address: string;
  city: string;
  status: 'active' | 'inactive' | 'maintenance';
  last_sync: string;
  created_at: string;
  updated_at: string;
}

export interface Ticket {
  id: string;
  title: string;
  description: string;
  device_id?: string;
  priority: string;
  status: string;
  assignee?: string;
  created_at: string;
  updated_at: string;
  comments?: TicketComment[];
}

export interface TicketComment {
  id: string;
  ticket_id: string;
  user_id?: string;
  user_name?: string;
  content: string;
  created_at: string;
}

// ─── Sites API ──────────────────────────────────────────────────────

export const sitesApi = {
  getSites(): Promise<Site[]> {
    return request<Site[]>('/sites');
  },

  getSite(siteId: string): Promise<Site> {
    return request<Site>(`/sites/${siteId}`);
  },

  createSite(site: Partial<Site>): Promise<Site> {
    return request<Site>('/sites', {
      method: 'POST',
      body: JSON.stringify(site),
    });
  },

  updateSite(siteId: string, updates: Partial<Site>): Promise<Site> {
    return request<Site>(`/sites/${siteId}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  },

  deleteSite(siteId: string): Promise<void> {
    return request<void>(`/sites/${siteId}`, {
      method: 'DELETE',
    });
  },
};

// ─── Tickets API ────────────────────────────────────────────────────

export const ticketsApi = {
  getTickets(): Promise<Ticket[]> {
    return request<Ticket[]>('/tickets');
  },

  getTicket(ticketId: string): Promise<Ticket> {
    return request<Ticket>(`/tickets/${ticketId}`);
  },

  createTicket(ticket: Partial<Ticket>): Promise<Ticket> {
    return request<Ticket>('/tickets', {
      method: 'POST',
      body: JSON.stringify(ticket),
    });
  },

  updateTicket(ticketId: string, updates: Partial<Ticket>): Promise<Ticket> {
    return request<Ticket>(`/tickets/${ticketId}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  },

  deleteTicket(ticketId: string): Promise<void> {
    return request<void>(`/tickets/${ticketId}`, {
      method: 'DELETE',
    });
  },

  addTicketComment(ticketId: string, content: string): Promise<TicketComment> {
    return request<TicketComment>(`/tickets/${ticketId}/comments`, {
      method: 'POST',
      body: JSON.stringify({ content }),
    });
  },
};
