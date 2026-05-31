import client from './client';
import type { CurrencyConvertResponse, CurrencyListResponse, CurrencyOption } from '../types';

export async function listCurrencies(defaultCurrency?: string): Promise<CurrencyOption[]> {
  const response = await client.get<CurrencyListResponse>('/api/v1/currencies', {
    params: defaultCurrency ? { default_currency: defaultCurrency } : undefined,
  });

  return response.data.currencies;
}

export async function convertCurrency(amount: number, fromCurrency: string, toCurrency: string): Promise<CurrencyConvertResponse> {
  const response = await client.get<CurrencyConvertResponse>('/api/v1/currencies/convert', {
    params: { amount, from: fromCurrency, to: toCurrency },
  });

  return response.data;
}
