import client from './client';
import type {
  CategoryListResponse,
  CreateExpenseRequest,
  Expense,
  ListExpensesResponse,
  UpdateExpenseRequest,
  WalletListResponse,
} from '../types';

export async function listExpenses(limit = 20, pagination?: string): Promise<ListExpensesResponse> {
  const response = await client.get<ListExpensesResponse>('/api/v1/expenses', {
    params: { limit, pagination },
  });

  return response.data;
}

export async function createExpense(data: CreateExpenseRequest): Promise<Expense> {
  const response = await client.post<Expense>('/api/v1/expenses', {
    ...data,
    fx_rate_to_base: data.fx_rate_to_base ?? 1,
    is_recurring: data.is_recurring ?? false,
  });

  return response.data;
}

export async function updateExpense(id: string, data: UpdateExpenseRequest): Promise<Expense> {
  const response = await client.put<Expense>(`/api/v1/expenses/${id}`, {
    ...data,
    fx_rate_to_base: data.fx_rate_to_base ?? 1,
    is_recurring: data.is_recurring ?? false,
  });

  return response.data;
}

export async function deleteExpense(id: string): Promise<void> {
  await client.delete(`/api/v1/expenses/${id}`);
}

export async function listCategories(): Promise<CategoryListResponse['categories']> {
  const response = await client.get<CategoryListResponse>('/api/v1/categories');
  return response.data.categories;
}

export async function listWallets(): Promise<WalletListResponse['wallets']> {
  const response = await client.get<WalletListResponse>('/api/v1/wallets');
  return response.data.wallets;
}