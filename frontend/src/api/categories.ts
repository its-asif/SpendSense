import client from './client';
import type { CategoryKind, CreateCategoryRequest, ExpenseCategory, UpdateCategoryRequest } from '../types';

export async function listCategories(kind?: CategoryKind): Promise<ExpenseCategory[]> {
  const response = await client.get<{ categories: ExpenseCategory[] }>('/api/v1/categories', {
    params: kind ? { kind } : undefined,
  });
  return response.data.categories;
}

export async function createCategory(data: CreateCategoryRequest): Promise<ExpenseCategory> {
  const response = await client.post<ExpenseCategory>('/api/v1/categories', data);
  return response.data;
}

export async function updateCategory(id: string, data: UpdateCategoryRequest): Promise<ExpenseCategory> {
  const response = await client.put<ExpenseCategory>(`/api/v1/categories/${id}`, data);
  return response.data;
}

export async function deleteCategory(id: string): Promise<void> {
  await client.delete(`/api/v1/categories/${id}`);
}
