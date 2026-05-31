import { useCallback, useEffect, useState } from 'react';
import { listCategories, listWallets } from '../api/expenses';
import type { ExpenseCategory, Wallet } from '../types';

export function useDashboardMeta() {
  const [categories, setCategories] = useState<ExpenseCategory[]>([]);
  const [wallets, setWallets] = useState<Wallet[]>([]);
  const [isLoadingMeta, setIsLoadingMeta] = useState(true);
  const [metaError, setMetaError] = useState<string | null>(null);

  const refreshMeta = useCallback(async () => {
    setMetaError(null);
    setIsLoadingMeta(true);

    try {
      const [nextCategories, nextWallets] = await Promise.all([listCategories(), listWallets()]);
      setCategories(nextCategories);
      setWallets(nextWallets);
    } catch {
      setMetaError('Failed to refresh categories or wallets.');
    } finally {
      setIsLoadingMeta(false);
    }
  }, []);

  const syncWallets = useCallback(async () => {
    try {
      const nextWallets = await listWallets();
      setWallets(nextWallets);
    } catch {
      setMetaError('Failed to refresh wallets.');
    }
  }, []);

  useEffect(() => {
    void refreshMeta();
  }, [refreshMeta]);

  return {
    categories,
    wallets,
    isLoadingMeta,
    metaError,
    refreshMeta,
    syncWallets,
    setCategories,
    setWallets,
  };
}