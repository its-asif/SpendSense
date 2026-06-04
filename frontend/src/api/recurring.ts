import client from './client';

export type RecurringPayment = {
  id: string;
  wallet_id: string;
  wallet_name: string;
  category_id: string;
  category_name: string;
  category_color: string;
  category_icon: string;
  title: string;
  amount: number;
  currency: string;
  interval: string;
  start_date: string;
  deadline: string;
  alert_rule: string;
  end_date?: string | null;
  status: 'unpaid' | 'paid' | 'inactive';
  created_at: string;
  updated_at: string;
};

export type CreateRecurringRequest = {
  wallet_id: string;
  category_id: string;
  title: string;
  amount: number;
  currency: string;
  interval: string;
  start_date: string;
  deadline: string;
  alert_rule: string;
  end_date?: string | null;
};

export type PayRecurringRequest = {
  payment_date: string;
  fine: number;
};

export type ListRecurringResponse = {
  recurring_payments: RecurringPayment[];
};

export async function listRecurringPayments(): Promise<ListRecurringResponse> {
  const response = await client.get<ListRecurringResponse>('/api/v1/recurring-payments');
  return response.data;
}

export async function createRecurringPayment(data: CreateRecurringRequest): Promise<RecurringPayment> {
  const response = await client.post<RecurringPayment>('/api/v1/recurring-payments', data);
  return response.data;
}

export async function updateRecurringPayment(id: string, data: CreateRecurringRequest): Promise<RecurringPayment> {
  const response = await client.put<RecurringPayment>(`/api/v1/recurring-payments/${id}`, data);
  return response.data;
}

export async function deleteRecurringPayment(id: string): Promise<void> {
  await client.delete(`/api/v1/recurring-payments/${id}`);
}

export async function payRecurringPayment(id: string, data: PayRecurringRequest): Promise<RecurringPayment> {
  const response = await client.post<RecurringPayment>(`/api/v1/recurring-payments/${id}/pay`, data);
  return response.data;
}
