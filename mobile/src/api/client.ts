import axios, { AxiosInstance, AxiosError, InternalAxiosRequestConfig, AxiosResponse } from 'axios';
import { storage } from '../utils/storage';
import { useAuthStore } from '../store/authStore';
import { LoginResponse } from '../types';

declare const process: { env: Record<string, string | undefined> };

const API_BASE_URL: string = process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
    // P1-SEC.1: Мобильные клиенты получают токены в response body
    // для хранения в secure storage (AsyncStorage/Keychain).
    'X-Client-Type': 'mobile',
  },
});

let refreshPromise: Promise<LoginResponse> | null = null;

apiClient.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    const token = await storage.getToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error: AxiosError) => Promise.reject(error),
);

apiClient.interceptors.response.use(
  (response: AxiosResponse) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

    if (error.response?.status !== 401 || originalRequest._retry) {
      return Promise.reject(error);
    }

    const refreshToken = await storage.getRefreshToken();
    if (!refreshToken) {
      await useAuthStore.getState().logout();
      return Promise.reject(error);
    }

    originalRequest._retry = true;
    try {
      refreshPromise = refreshPromise ?? axios
        .post<LoginResponse>(`${API_BASE_URL}/auth/refresh`, { refresh_token: refreshToken }, { timeout: 15000 })
        .then((response) => response.data)
        .finally(() => {
          refreshPromise = null;
        });

      const refreshed = await refreshPromise;
      await storage.setToken(refreshed.token);
      await storage.setRefreshToken(refreshed.refresh_token);
      await storage.setUser(JSON.stringify(refreshed.user));
      useAuthStore.setState({
        token: refreshed.token,
        refreshToken: refreshed.refresh_token,
        user: refreshed.user,
        isAuthenticated: true,
        isLoading: false,
      });

      originalRequest.headers.Authorization = `Bearer ${refreshed.token}`;
      return apiClient(originalRequest);
    } catch (refreshError) {
      await useAuthStore.getState().logout();
      return Promise.reject(refreshError);
    }
  },
);
