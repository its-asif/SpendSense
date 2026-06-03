import client from './client';
import type { Budget, CreateBudgetRequest, UpdateBudgetRequest } from '../types';

export async function listBudgets(period?: string): Promise<Budget[]> {
  const response = await client.get<{ budgets: Budget[] }>('/api/v1/budgets', {
    params: period ? { period } : undefined,
  });
  return response.data.budgets;
}

export async function createBudget(data: CreateBudgetRequest): Promise<Budget> {
  const response = await client.post<Budget>('/api/v1/budgets', data);
  return response.data;
}

export async function updateBudget(id: string, data: UpdateBudgetRequest): Promise<Budget> {
  const response = await client.put<Budget>(`/api/v1/budgets/${id}`, data);
  return response.data;
}

export async function deleteBudget(id: string): Promise<void> {
  await client.delete(`/api/v1/budgets/${id}`);
}
