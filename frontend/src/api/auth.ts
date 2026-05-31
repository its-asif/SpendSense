import client from './client';
import type { ApiAuthResponse, ApiUser, AuthSession, LoginRequest, RegisterRequest, ApiListSessionsResponse } from '../types';
import { toAuthUser } from '../lib/storage';

function toAuthSession(response: ApiAuthResponse): AuthSession {
  return {
    accessToken: response.access_token,
    refreshToken: response.refresh_token,
    user: toAuthUser(response.user.email, response.user.display_name ?? undefined, response.user.base_currency),
  };
}

export async function login(data: LoginRequest): Promise<AuthSession> {
  const response = await client.post<ApiAuthResponse>('/auth/login', data);
  return toAuthSession(response.data);
}

export async function register(data: RegisterRequest): Promise<AuthSession> {
  const response = await client.post<ApiAuthResponse>('/auth/register', data);
  return toAuthSession(response.data);
}

export async function logout(refreshToken: string) {
  await client.post('/auth/logout', { refresh_token: refreshToken });
}

export async function logoutAll() {
  await client.post('/auth/logout-all');
}

export async function logoutOtherSessions() {
  await client.post('/auth/logout-other');
}

export type UpdateUserPreferencesRequest = {
  baseCurrency: string;
  timezone: string;
  locale: string;
};

export type UpdateUserProfileRequest = {
  displayName: string;
  avatarUrl: string;
};

export async function updateUserProfile(data: UpdateUserProfileRequest): Promise<ApiUser> {
  const response = await client.put<ApiUser>('/auth/me/profile', {
    display_name: data.displayName,
    avatar_url: data.avatarUrl,
  });

  return response.data;
}

export type ChangePasswordRequest = {
  oldPassword: string;
  newPassword: string;
};

export async function changePassword(data: ChangePasswordRequest): Promise<{ ok: boolean }> {
  const response = await client.put<{ ok: boolean }>('/auth/me/password', {
    old_password: data.oldPassword,
    new_password: data.newPassword,
  });

  return response.data;
}

export async function listSessions(): Promise<ApiListSessionsResponse> {
  const response = await client.get<ApiListSessionsResponse>('/auth/me/sessions');
  return response.data;
}

export async function revokeSession(id: string): Promise<{ ok: boolean }> {
  const response = await client.delete<{ ok: boolean }>(`/auth/me/sessions/${id}`);
  return response.data;
}

export async function updateUserPreferences(data: UpdateUserPreferencesRequest): Promise<ApiUser> {
  const response = await client.put<ApiUser>('/auth/me/preferences', {
    base_currency: data.baseCurrency,
    timezone: data.timezone,
    locale: data.locale,
  });

  return response.data;
}

export async function me(): Promise<{ user_id: string; email: string; session_id: string; totp_enabled: boolean }> {
  const response = await client.get('/auth/me');
  return response.data;
}

export async function twoFASetup(): Promise<{ secret: string; otp_auth_url: string; qr_data_url: string }> {
  const response = await client.get('/auth/me/2fa/setup');
  return response.data;
}

export async function twoFAConfirm(code: string, secret?: string): Promise<{ ok: boolean }> {
  const response = await client.post('/auth/me/2fa/confirm', { code, secret });
  return response.data;
}

export async function twoFADisable(): Promise<{ ok: boolean }> {
  const response = await client.post('/auth/me/2fa/disable');
  return response.data;
}