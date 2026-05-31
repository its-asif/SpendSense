import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Layout } from '../components/layout/Layout';
import { Button } from '../components/common/Button';
import Modal from '../components/common/Modal';
import WalletForm from '../components/wallet/WalletForm';
import WalletTransferForm from '../components/wallet/WalletTransferForm';
import { WalletOverviewHeader } from '../components/wallet/WalletOverviewHeader';
import { WalletRail } from '../components/wallet/WalletRail';
import { WalletOverviewSummary } from '../components/wallet/WalletOverviewSummary';
import { WalletBalanceChartCard, type WalletBalancePoint } from '../components/wallet/WalletBalanceChartCard';
import { TransactionHistoryCard } from '../components/dashboard/TransactionHistoryCard';
import { useDashboardMeta } from '../hooks/useDashboardMeta';
import { useUserSettings } from '../hooks/useUserSettings';
import { createWallet, createWalletTransfer, deleteWallet, updateWallet } from '../api/wallets';
import { convertCurrency, listCurrencies } from '../api/currencies';
import { listExpenses } from '../api/expenses';
import { listIncomes } from '../api/incomes';
import type { AuthUser, Wallet, CurrencyOption, Expense, Income, ExpenseCategory } from '../types';

type WalletsPageProps = {
  user: AuthUser;
  onLogout: () => void;
};

type WalletActivityRow = {
  id: string;
  category: string;
  categoryColor: string;
  description: string;
  dateLabel: string;
  amount: number;
  currency: string;
  type: 'income' | 'expense';
  status?: 'Paid' | 'Due' | 'Cancel';
};

function formatAxisValue(value: number) {
  if (Math.abs(value) >= 1000) {
    return `${(value / 1000).toFixed(1)}k`;
  }

  return `${Math.round(value)}`;
}

export function WalletsPage({ user, onLogout }: WalletsPageProps) {
  const {
    categories,
    wallets,
    refreshMeta,
    syncWallets,
    isLoadingMeta,
  } = useDashboardMeta();
  const settings = useUserSettings();
  const [selectedWalletId, setSelectedWalletId] = useState<string | null>(null);
  const [creatingWallet, setCreatingWallet] = useState(false);
  const [editingWallet, setEditingWallet] = useState<Wallet | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Wallet | null>(null);
  const [deletingWalletId, setDeletingWalletId] = useState<string | null>(null);
  const [transferFromWallet, setTransferFromWallet] = useState<Wallet | null>(null);
  const [currencies, setCurrencies] = useState<CurrencyOption[]>([]);
  const [isLoadingCurrencies, setIsLoadingCurrencies] = useState(false);
  const [walletBalancePoints, setWalletBalancePoints] = useState<WalletBalancePoint[]>([]);
  const [isLoadingWalletBalances, setIsLoadingWalletBalances] = useState(true);
  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [incomes, setIncomes] = useState<Income[]>([]);
  const [isLoadingTransactions, setIsLoadingTransactions] = useState(true);

  useEffect(() => {
    let cancelled = false;
    if (!transferFromWallet) {
      return;
    }

    setIsLoadingCurrencies(true);
    void listCurrencies()
      .then((items) => {
        if (!cancelled) {
          setCurrencies(items);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setCurrencies([]);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoadingCurrencies(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [transferFromWallet]);

  useEffect(() => {
    let cancelled = false;

    async function loadTransactions() {
      setIsLoadingTransactions(true);

      try {
        const [expenseResponse, incomeResponse] = await Promise.all([listExpenses(), listIncomes()]);

        if (!cancelled) {
          setExpenses(expenseResponse.expenses);
          setIncomes(incomeResponse.incomes);
        }
      } catch {
        if (!cancelled) {
          setExpenses([]);
          setIncomes([]);
        }
      } finally {
        if (!cancelled) {
          setIsLoadingTransactions(false);
        }
      }
    }

    void loadTransactions();

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (wallets.length === 0) {
      setSelectedWalletId(null);
      return;
    }

    if (!selectedWalletId || !wallets.some((wallet) => wallet.id === selectedWalletId)) {
      setSelectedWalletId(wallets[0].id);
    }
  }, [wallets, selectedWalletId]);

  useEffect(() => {
    let cancelled = false;

    async function loadWalletBalances() {
      setIsLoadingWalletBalances(true);
      const selectedCurrency = settings.defaultCurrency;
      const rateCache = new Map<string, number>();

      async function convertToSelectedCurrency(amount: number, fromCurrency: string) {
        const normalizedFrom = fromCurrency.trim().toUpperCase();
        const normalizedTo = selectedCurrency.trim().toUpperCase();

        if (!normalizedFrom || !normalizedTo) {
          return amount;
        }

        if (normalizedFrom === normalizedTo) {
          return amount;
        }

        const cacheKey = `${normalizedFrom}->${normalizedTo}`;
        const cachedRate = rateCache.get(cacheKey);
        if (cachedRate !== undefined) {
          return amount * cachedRate;
        }

        const conversion = await convertCurrency(1, normalizedFrom, normalizedTo);
        const rate = conversion.converted_amount || conversion.exchange_rate || 1;
        rateCache.set(cacheKey, rate);
        return amount * rate;
      }

      const palette = ['#6366F1', '#10B981', '#F59E0B', '#A855F7', '#EF4444', '#06B6D4'];
      const points: WalletBalancePoint[] = [];

      for (const [index, wallet] of wallets.entries()) {
        const balance = await convertToSelectedCurrency(wallet.current_balance, wallet.currency);
        points.push({
          id: wallet.id,
          name: wallet.name,
          balance: Number(balance.toFixed(2)),
          color: wallet.wallet_type?.toUpperCase() === 'CARD' ? '#EF4444' : palette[index % palette.length],
        });
      }

      if (!cancelled) {
        setWalletBalancePoints(points);
        setIsLoadingWalletBalances(false);
      }
    }

    void loadWalletBalances();

    return () => {
      cancelled = true;
    };
  }, [wallets, settings.defaultCurrency]);

  const selectedWallet = wallets.find((wallet) => wallet.id === selectedWalletId) ?? wallets[0] ?? null;
  const totalBalance = walletBalancePoints.reduce((sum, point) => sum + point.balance, 0);
  const personalFunds = walletBalancePoints
    .filter((point) => wallets.find((wallet) => wallet.id === point.id)?.wallet_type?.toUpperCase() !== 'CARD')
    .reduce((sum, point) => sum + point.balance, 0);
  const creditExposure = walletBalancePoints
    .filter((point) => wallets.find((wallet) => wallet.id === point.id)?.wallet_type?.toUpperCase() === 'CARD')
    .reduce((sum, point) => sum + Math.abs(point.balance), 0);

  const categoryNameById = new Map(categories.map((category: ExpenseCategory) => [category.id, category.name]));
  const categoryColorById = new Map(categories.map((category: ExpenseCategory, index) => [category.id, category.color ?? ['#6366F1', '#10B981', '#F59E0B', '#A855F7'][index % 4]]));

  const selectedWalletTransactions: WalletActivityRow[] = [
    ...expenses
      .filter((expense) => expense.wallet_id === selectedWallet?.id)
      .map((expense) => {
        const date = new Date(expense.date);

        return {
          id: `expense-${expense.id}`,
          category: categoryNameById.get(expense.category_id) || 'Expense',
          categoryColor: categoryColorById.get(expense.category_id) || '#6366F1',
          description: expense.merchant?.trim() || 'Expense payment',
          dateLabel: Number.isNaN(date.getTime())
            ? expense.date
            : new Intl.DateTimeFormat('en-US', { day: 'numeric', month: 'long', year: 'numeric' }).format(date),
          amount: expense.amount,
          currency: expense.currency,
          type: 'expense' as const,
          status: expense.notes?.toLowerCase().includes('cancel')
            ? 'Cancel' as const
            : date > new Date()
              ? 'Due' as const
              : 'Paid' as const,
          sortDate: date.getTime(),
        };
      }),
    ...incomes
      .filter((income) => income.wallet_id === selectedWallet?.id)
      .map((income) => {
        const date = new Date(income.income_date);

        return {
          id: `income-${income.id}`,
          category: income.category_id ? categoryNameById.get(income.category_id) || 'Income' : 'Income',
          categoryColor: income.category_id ? categoryColorById.get(income.category_id) || '#10B981' : '#10B981',
          description: income.source_name,
          dateLabel: Number.isNaN(date.getTime())
            ? income.income_date
            : new Intl.DateTimeFormat('en-US', { day: 'numeric', month: 'long', year: 'numeric' }).format(date),
          amount: income.amount,
          currency: income.currency,
          type: 'income' as const,
          status: 'Paid' as const,
          sortDate: date.getTime(),
        };
      }),
  ]
    .sort((a, b) => b.sortDate - a.sortDate)
    .slice(0, 6)
    .map(({ sortDate, ...transaction }) => transaction);

  return (
    <Layout>
      <div className="container mx-auto px-4">
        <WalletOverviewHeader
          title="Financial Overview"
          breadcrumbLabel="Financial Overview"
          onRefresh={() => void refreshMeta()}
          isRefreshing={isLoadingMeta || isLoadingWalletBalances}
        />

        <div className="py-4 sm:py-6">
          <div className="grid grid-cols-1 gap-4 sm:gap-6 lg:grid-cols-12">
            <div className="lg:col-span-3">
              <WalletRail
                wallets={wallets}
                selectedWalletId={selectedWalletId}
                onSelectWallet={setSelectedWalletId}
                onAddWallet={() => setCreatingWallet(true)}
              />
            </div>

            <div className="lg:col-span-9 space-y-4 sm:space-y-6">
              <WalletOverviewSummary
                selectedWallet={selectedWallet}
                selectedWalletBalance={selectedWallet?.current_balance ?? 0}
                totalBalance={totalBalance}
                personalFunds={personalFunds}
                creditExposure={creditExposure}
                currencyCode={settings.defaultCurrency}
                onEditSelectedWallet={() => selectedWallet && setEditingWallet(selectedWallet)}
                onTransferSelectedWallet={() => selectedWallet && setTransferFromWallet(selectedWallet)}
                onDeleteSelectedWallet={() => selectedWallet && setPendingDelete(selectedWallet)}
              />

              <section className="grid gap-4 sm:gap-6">
                <WalletBalanceChartCard
                  className="w-full"
                  isLoading={isLoadingWalletBalances}
                  points={walletBalancePoints}
                  currencyCode={settings.defaultCurrency}
                  formatAxisValue={formatAxisValue}
                />
              </section>

              <section className="grid gap-4 sm:gap-6">
                <TransactionHistoryCard
                  className="w-full"
                  transactions={selectedWalletTransactions}
                />
              </section>
            </div>
          </div>
        </div>
      </div>

      {creatingWallet && (
        <Modal title="Add wallet" onClose={() => setCreatingWallet(false)}>
          <WalletForm
            onSubmit={async (data) => {
              try {
                await createWallet(data);
                toast.success('Wallet created');
                await refreshMeta();
                setCreatingWallet(false);
              } catch {
                toast.error('Failed to create wallet');
              }
            }}
            onCancel={() => setCreatingWallet(false)}
          />
        </Modal>
      )}

      {editingWallet && (
        <Modal title="Edit wallet" onClose={() => setEditingWallet(null)}>
          <WalletForm
            initial={editingWallet}
            onSubmit={async (data) => {
              try {
                await updateWallet(editingWallet.id, data);
                toast.success('Wallet updated');
                await refreshMeta();
                setEditingWallet(null);
              } catch {
                toast.error('Failed to update wallet');
              }
            }}
            onCancel={() => setEditingWallet(null)}
          />
        </Modal>
      )}

      {transferFromWallet && (
        <Modal title={`Transfer from ${transferFromWallet.name}`} onClose={() => setTransferFromWallet(null)}>
          <WalletTransferForm
            wallets={wallets}
            currencies={currencies}
            loadingCurrencies={isLoadingCurrencies}
            from={transferFromWallet}
            onSubmit={async (data) => {
              try {
                await createWalletTransfer({ ...data, transfer_date: new Date().toISOString() });
                toast.success('Transfer created');
                await refreshMeta();
                void syncWallets();
                setTransferFromWallet(null);
              } catch {
                toast.error('Failed to create transfer');
              }
            }}
            onCancel={() => setTransferFromWallet(null)}
          />
        </Modal>
      )}

      {pendingDelete && (
        <Modal title="Confirm delete" onClose={() => setPendingDelete(null)}>
          <p className="text-sm text-text-secondary">Are you sure you want to delete this wallet?</p>
          <div className="mt-4 flex items-center gap-3">
            <Button variant="secondary" onClick={() => setPendingDelete(null)}>Cancel</Button>
            <Button
              onClick={async () => {
                setDeletingWalletId(pendingDelete.id);
                try {
                  await deleteWallet(pendingDelete.id);
                  toast.success('Wallet deleted');
                  await refreshMeta();
                  setPendingDelete(null);
                } catch {
                  toast.error('Failed to delete wallet');
                } finally {
                  setDeletingWalletId(null);
                }
              }}
            >
              {deletingWalletId === pendingDelete.id ? 'Deleting...' : 'Delete'}
            </Button>
          </div>
        </Modal>
      )}
    </Layout>
  );
}
