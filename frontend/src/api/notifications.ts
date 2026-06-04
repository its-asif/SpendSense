import client from './client';
import type { NotificationListResponse } from '../types';

export async function listNotifications(limit = 30): Promise<NotificationListResponse> {
  const response = await client.get<NotificationListResponse>('/api/v1/notifications', { params: { limit } });
  return response.data;
}

export async function markNotificationRead(id: string): Promise<void> {
  await client.post(`/api/v1/notifications/${id}/read`);
}

export async function dismissNotification(id: string): Promise<void> {
  await client.post(`/api/v1/notifications/${id}/dismiss`);
}

export async function markAllNotificationsRead(): Promise<void> {
  await client.post('/api/v1/notifications/read-all');
}
