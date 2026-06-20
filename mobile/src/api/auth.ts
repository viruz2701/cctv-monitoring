import { apiClient } from './client';
import { LoginResponse } from '../types';
import { storage } from '../utils/storage';

export const authApi = {
  login: async (username: string, password: string): Promise<LoginResponse> => {
    const response = await apiClient.post<LoginResponse>('/auth/login', {
      username,
      password,
    });
    await storage.setToken(response.data.token);
    await storage.setUser(JSON.stringify(response.data.user));
    return response.data;
  },

  logout: async (): Promise<void> => {
    await storage.removeToken();
    await storage.removeUser();
  },

  getCurrentUser: async () => {
    const response = await apiClient.get('/users/me');
    return response.data;
  },

  registerPushToken: async (token: string): Promise<void> => {
    await apiClient.post('/mobile/register-token', { token });
  },
};