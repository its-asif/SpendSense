import client from './client';
import type { UpdateWalletRequest, CreateWalletRequest } from '../types';

export async function updateWallet(id: string, data: UpdateWalletRequest) {
  const response = await client.put(`/api/v1/wallets/${id}`, data);
  return response.data;
}

export async function getWallet(id: string) {
  const response = await client.get(`/api/v1/wallets/${id}`);
  return response.data;
}

export async function createWallet(data: CreateWalletRequest) {
  const response = await client.post(`/api/v1/wallets`, data);
  return response.data;
}

export async function deleteWallet(id: string) {
  await client.delete(`/api/v1/wallets/${id}`);
}

export async function createWalletTransfer(data: { from_wallet_id: string; to_wallet_id: string; amount: number; currency: string; notes?: string; transfer_date?: string; exchange_rate?: number; fee_amount?: number; }) {
  const response = await client.post(`/api/v1/wallets/transfer`, data);
  return response.data;
}
