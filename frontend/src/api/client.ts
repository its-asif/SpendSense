import axios, { AxiosError, type AxiosInstance, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios';
import { clearSession, readSession, saveSession } from '../lib/storage';
import type { ApiTokenResponse } from '../types';

type RetriableRequestConfig = InternalAxiosRequestConfig & {
  _retry?: boolean;
};

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

const client: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const session = readSession();
  if (session?.accessToken) {
    config.headers.Authorization = `Bearer ${session.accessToken}`;
  }

  return config;
});

client.interceptors.response.use(
  (response: AxiosResponse) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as RetriableRequestConfig | undefined;
    // Don't trigger token refresh or redirect for auth endpoints themselves
    const url = originalRequest?.url ?? '';
    if (url.includes('/auth/login') || url.includes('/auth/register') || url.includes('/auth/refresh')) {
      return Promise.reject(error);
    }

    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      originalRequest._retry = true;

      const session = readSession();
      if (!session?.refreshToken) {
        clearSession();
        window.location.href = '/auth';
        return Promise.reject(error);
      }

      try {
        const response = await axios.post<ApiTokenResponse>(`${API_BASE_URL}/auth/refresh`, {
          email: session.user.email,
          refresh_token: session.refreshToken,
        });

        const nextSession = {
          ...session,
          accessToken: response.data.access_token,
        };
        saveSession(nextSession);

        if (originalRequest.headers) {
          originalRequest.headers.Authorization = `Bearer ${response.data.access_token}`;
        }

        return client(originalRequest);
      } catch (refreshError) {
        clearSession();
        window.location.href = '/auth';
        return Promise.reject(refreshError);
      }
    }

    return Promise.reject(error);
  }
);

export default client;