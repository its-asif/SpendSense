import client from './client';
import type {
  CreateIncomeRequest,
  Income,
  ListIncomesResponse,
  UpdateIncomeRequest,
} from '../types';

export async function listIncomes(limit = 20, pagination?: string): Promise<ListIncomesResponse> {
  const response = await client.get<ListIncomesResponse>('/api/v1/incomes', {
    params: { limit, pagination },
  });

  return response.data;
}

export async function createIncome(data: CreateIncomeRequest): Promise<Income> {
  const response = await client.post<Income>('/api/v1/incomes', {
    ...data,
  });

  return response.data;
}

export async function updateIncome(id: string, data: UpdateIncomeRequest): Promise<Income> {
  const response = await client.put<Income>(`/api/v1/incomes/${id}`, {
    ...data,
  });

  return response.data;
}

export async function deleteIncome(id: string): Promise<void> {
  await client.delete(`/api/v1/incomes/${id}`);
}