// ═══════════════════════════════════════════════════════════════════════
// Users & Auth API
// ARCH.2: Выделен из monolithic api.ts.
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface User {
  id: string;
  username: string;
  name: string;
  role: 'admin' | 'support' | 'owner' | 'manager' | 'technician' | 'viewer';
  owner_id?: string | null;
  created_at: string;
  avatar?: string;
  sites?: string[];
  email: string;
  status?: 'active' | 'inactive';
  lastLogin?: string;
}

export interface TechnicianSiteAssignment {
  id: string;
  technician_id: string;
  site_id: string;
  is_primary: boolean;
  assigned_at: string;
  assigned_by: string;
  technician_name?: string;
  site_name?: string;
}

// ─── Auth Methods ───────────────────────────────────────────────────

export const authApi = {
  async login(username: string, password: string): Promise<{ token?: string; user: User }> {
    const { setAuthToken } = await import('./client');
    const data = await request<{ token?: string; user: User }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
    if (data.token) {
      setAuthToken(data.token);
    }
    return data;
  },

  getCurrentUser(): Promise<User> {
    return request<User>('/users/me');
  },

  async logout(): Promise<void> {
    try {
      await request<void>('/auth/logout', { method: 'POST' });
    } catch {
      // Ignore errors
    }
    const { setAuthToken } = await import('./client');
    setAuthToken(null);
  },

  login2FA(sessionToken: string, code: string): Promise<{ token?: string; user: User }> {
    return request<{ token?: string; user: User }>('/auth/login/2fa', {
      method: 'POST',
      body: JSON.stringify({ session_token: sessionToken, code }),
    });
  },
};

// ─── User CRUD ──────────────────────────────────────────────────────

export const usersApi = {
  getUsers(): Promise<User[]> {
    return request<User[]>('/users');
  },

  getUser(userId: string): Promise<User> {
    return request<User>(`/users/${userId}`);
  },

  createUser(user: { username: string; password: string; role: string; email?: string }): Promise<User> {
    return request<User>('/users', {
      method: 'POST',
      body: JSON.stringify(user),
    });
  },

  updateUser(userId: string, updates: Partial<User>): Promise<User> {
    return request<User>(`/users/${userId}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  },

  deleteUser(userId: string): Promise<void> {
    return request<void>(`/users/${userId}`, {
      method: 'DELETE',
    });
  },

  changePassword(currentPassword: string, newPassword: string): Promise<void> {
    return request<void>('/users/me/password', {
      method: 'PUT',
      body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
    });
  },

  resetUserPassword(userId: string, newPassword: string): Promise<void> {
    return request<void>(`/users/${userId}/reset-password`, {
      method: 'PUT',
      body: JSON.stringify({ new_password: newPassword }),
    });
  },

  // 2FA
  setup2FA(): Promise<{ secret: string; uri: string }> {
    return request<{ secret: string; uri: string }>('/users/me/2fa/setup', {
      method: 'POST',
    });
  },

  verify2FA(code: string): Promise<void> {
    return request<void>('/users/me/2fa/verify', {
      method: 'POST',
      body: JSON.stringify({ code }),
    });
  },

  disable2FA(password: string): Promise<void> {
    return request<void>('/users/me/2fa/disable', {
      method: 'POST',
      body: JSON.stringify({ password }),
    });
  },

  // Sessions
  getSessions(): Promise<any[]> {
    return request<any[]>('/sessions');
  },

  revokeSession(sessionId: string): Promise<void> {
    return request<void>(`/sessions/${sessionId}`, {
      method: 'DELETE',
    });
  },

  revokeAllOtherSessions(currentSessionId: string): Promise<void> {
    return request<void>('/sessions/revoke-all', {
      method: 'POST',
      body: JSON.stringify({ current_session_id: currentSessionId }),
    });
  },

  // Telegram
  generateTelegramLink(): Promise<{ token: string; expires_at: string }> {
    return request<{ token: string; expires_at: string }>('/users/me/telegram/generate-link', {
      method: 'POST',
    });
  },

  getTelegramStatus(): Promise<{ linked: boolean; alerts: boolean; tfa: boolean }> {
    return request<{ linked: boolean; alerts: boolean; tfa: boolean }>('/users/me/telegram/status');
  },

  updateTelegramSettings(settings: { alerts: boolean; tfa: boolean }): Promise<void> {
    return request<void>('/users/me/telegram/settings', {
      method: 'POST',
      body: JSON.stringify(settings),
    });
  },

  requestTelegramLoginCode(username: string): Promise<{ message: string; code: string }> {
    return request<{ message: string; code: string }>('/auth/telegram/request-code', {
      method: 'POST',
      body: JSON.stringify({ username }),
    });
  },

  verifyTelegramLogin(username: string, code: string): Promise<{ token: string; user: any }> {
    return request<{ token: string; user: any }>('/auth/telegram/verify', {
      method: 'POST',
      body: JSON.stringify({ username, code }),
    });
  },

  // API Keys
  getAPIKeys(): Promise<any[]> {
    return request<any[]>('/api-keys');
  },

  createAPIKey(data: { name: string; permissions: string[]; expires_at?: string }): Promise<any> {
    return request<any>('/api-keys', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  revokeAPIKey(id: string): Promise<void> {
    return request<void>(`/api-keys/${id}`, {
      method: 'DELETE',
    });
  },

  // Technician Site Assignments
  getTechnicianSiteAssignments(filters?: { technician_id?: string; site_id?: string; is_primary?: boolean }): Promise<TechnicianSiteAssignment[]> {
    const params = new URLSearchParams();
    if (filters?.technician_id) params.append('technician_id', filters.technician_id);
    if (filters?.site_id) params.append('site_id', filters.site_id);
    if (filters?.is_primary !== undefined) params.append('is_primary', filters.is_primary.toString());
    const query = params.toString() ? `?${params.toString()}` : '';
    return request<TechnicianSiteAssignment[]>(`/technician-assignments${query}`);
  },

  createTechnicianSiteAssignment(data: { technician_id: string; site_id: string; is_primary?: boolean }): Promise<TechnicianSiteAssignment> {
    return request<TechnicianSiteAssignment>('/technician-assignments', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  updateTechnicianSiteAssignment(id: string, data: { is_primary?: boolean }): Promise<void> {
    return request<void>(`/technician-assignments/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  deleteTechnicianSiteAssignment(id: string): Promise<void> {
    return request<void>(`/technician-assignments/${id}`, {
      method: 'DELETE',
    });
  },
};
