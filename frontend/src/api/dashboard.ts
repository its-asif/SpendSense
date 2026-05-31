import client from './client';
import type { DashboardSummary, DashboardWidgetsResponse } from '../types';

export async function getDashboardSummary(defaultCurrency?: string): Promise<DashboardSummary> {
  const response = await client.get<DashboardSummary>('/api/v1/dashboard/summary', {
    params: defaultCurrency ? { default_currency: defaultCurrency } : undefined,
  });
  return response.data;
}

export async function getDashboardWidgets(defaultCurrency?: string): Promise<DashboardWidgetsResponse> {
  const response = await client.get<DashboardWidgetsResponse>('/api/v1/dashboard/widgets', {
    params: defaultCurrency ? { default_currency: defaultCurrency } : undefined,
  });
  return response.data;
}