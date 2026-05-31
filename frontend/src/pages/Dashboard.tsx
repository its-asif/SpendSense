import { Header } from '../components/layout/Header';
import { Layout } from '../components/layout/Layout';
import { KPICard } from '../components/dashboard/KPICard';
import { MonthlyBudgetsCard } from '../components/dashboard/MonthlyBudgetsCard';
import { MonthlyIncomeExpensesChartCard } from '../components/dashboard/MonthlyIncomeExpensesChartCard';
import { SavingGoalsCard } from '../components/dashboard/SavingGoalsCard';
import { TransactionHistoryCard } from '../components/dashboard/TransactionHistoryCard';
import { Button } from '../components/common/Button';
import { Card } from '../components/common/Card';
import { ExpenseForm } from '../components/expense/ExpenseForm';
import { createExpense, deleteExpense, listExpenses, updateExpense } from '../api/expenses';
import { createIncome, deleteIncome, listIncomes, updateIncome } from '../api/incomes';
import { useEffect, useState } from 'react';
import Modal from '../components/common/Modal';
import toast from 'react-hot-toast';
import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import type {
  AuthUser,
  DashboardSummary,
  DashboardWidgetsResponse,
  CreateExpenseRequest,
  CreateIncomeRequest,
  Expense,
  ExpenseCategory,
  Income,
  Wallet,
} from '../types';
import { getDashboardSummary, getDashboardWidgets } from '../api/dashboard';
import { createWallet, updateWallet, deleteWallet, createWalletTransfer } from '../api/wallets';
import { convertCurrency } from '../api/currencies';
import WalletForm from '../components/wallet/WalletForm';
import WalletTransferForm from '../components/wallet/WalletTransferForm';
import { listCurrencies } from '../api/currencies';
import { IncomeForm } from '../components/income/IncomeForm';
import { useDashboardMeta } from '../hooks/useDashboardMeta';
import { useUserSettings } from '../hooks/useUserSettings';
import { formatCurrency } from '../lib/userSettings';

type DashboardProps = {
  user: AuthUser;
  onLogout: () => void;
};

type TrendPoint = {
  value: number;
  label: string;
};

type TrendRange = 'weekly' | 'monthly' | 'quarterly' | 'yearly';

type TrendRangeOption = {
  value: TrendRange;
  label: string;
  days: number;
};

const TREND_RANGE_OPTIONS: TrendRangeOption[] = [
  { value: 'weekly', label: 'Weekly', days: 7 },
  { value: 'monthly', label: 'Monthly', days: 30 },
  { value: 'quarterly', label: 'Quarterly', days: 90 },
  { value: 'yearly', label: 'Yearly', days: 365 },
];

function startOfDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function addDays(date: Date, days: number) {
  const next = new Date(date);
  next.setDate(next.getDate() + days);
  return next;
}

function toDateKey(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function toMonthKey(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  return `${year}-${month}`;
}

function formatTrendLabel(date: Date, bucket: 'day' | 'month') {
  return bucket === 'day'
    ? new Intl.DateTimeFormat('en-US', { day: 'numeric', month: 'short' }).format(date)
    : new Intl.DateTimeFormat('en-US', { month: 'short' }).format(date);
}

function formatTrendAxisValue(value: number) {
  if (Math.abs(value) >= 1000) {
    return `${(value / 1000).toFixed(1)}k`;
  }

  return `${Math.round(value)}`;
}

function buildRangeBuckets(range: TrendRange, now = new Date()) {
  const option = TREND_RANGE_OPTIONS.find((item) => item.value === range) ?? TREND_RANGE_OPTIONS[0];
  const current = startOfDay(now);
  const buckets: Array<{ key: string; label: string; start: Date; end: Date }> = [];

  const start = addDays(current, -(option.days - 1));
  for (let index = 0; index < option.days; index += 1) {
    const bucketStart = addDays(start, index);
    buckets.push({
      key: toDateKey(bucketStart),
      label: formatTrendLabel(bucketStart, 'day'),
      start: bucketStart,
      end: addDays(bucketStart, 1),
    });
  }

  return buckets;
}

function buildMonthBuckets(year: number) {
  return Array.from({ length: 12 }, (_, index) => {
    const monthDate = new Date(year, index, 1);
    return {
      key: toMonthKey(monthDate),
      label: monthDate.toLocaleDateString('en-US', { month: 'short' }),
      cash: 0,
      bank: 0,
      digital: 0,
      income: 0,
      expenses: 0,
    };
  });
}

export function Dashboard({ user, onLogout }: DashboardProps) {
  const settings = useUserSettings();
  const [trendRange, setTrendRange] = useState<TrendRange>('weekly');
  const [trendPoints, setTrendPoints] = useState<TrendPoint[]>([]);
  const [weeklyExpensePoints, setWeeklyExpensePoints] = useState<Array<{ label: string; cash: number; bank: number; digital: number }>>([]);
  const [monthlyCashFlowPoints, setMonthlyCashFlowPoints] = useState<DashboardWidgetsResponse['monthly_cash_flow']>([]);
  const [dashboardWidgets, setDashboardWidgets] = useState<DashboardWidgetsResponse | null>(null);
  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [incomes, setIncomes] = useState<Income[]>([]);
  const [dashboardSummary, setDashboardSummary] = useState<DashboardSummary | null>(null);
  const [isLoadingExpenses, setIsLoadingExpenses] = useState(true);
  const [isLoadingIncomes, setIsLoadingIncomes] = useState(true);
  const [isLoadingSummary, setIsLoadingSummary] = useState(true);
  const [isLoadingWidgets, setIsLoadingWidgets] = useState(true);
  const [isLoadingWeeklyExpenses, setIsLoadingWeeklyExpenses] = useState(true);
  const [isLoadingMonthlyCashFlow, setIsLoadingMonthlyCashFlow] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [summaryError, setSummaryError] = useState<string | null>(null);
  const [widgetsError, setWidgetsError] = useState<string | null>(null);
  const [editingExpense, setEditingExpense] = useState<Expense | null>(null);
  const [editingIncome, setEditingIncome] = useState<Income | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Expense | null>(null);
  const [pendingIncomeDelete, setPendingIncomeDelete] = useState<Income | null>(null);
  const [deletingExpenseId, setDeletingExpenseId] = useState<string | null>(null);
  const [deletingIncomeId, setDeletingIncomeId] = useState<string | null>(null);
  const [editingWallet, setEditingWallet] = useState<Wallet | null>(null);
  const [creatingWallet, setCreatingWallet] = useState(false);
  const [pendingWalletDelete, setPendingWalletDelete] = useState<Wallet | null>(null);
  const [deletingWalletId, setDeletingWalletId] = useState<string | null>(null);
  const [transferFromWallet, setTransferFromWallet] = useState<Wallet | null>(null);
  const [currenciesForTransfer, setCurrenciesForTransfer] = useState<any[]>([]);
  const [loadingCurrenciesForTransfer, setLoadingCurrenciesForTransfer] = useState(false);
  const [manageWalletsOpen, setManageWalletsOpen] = useState(false);
  const [manageExpensesOpen, setManageExpensesOpen] = useState(false);
  const [manageIncomesOpen, setManageIncomesOpen] = useState(false);

  const openCreateWallet = () => {
    setManageWalletsOpen(false);
    setCreatingWallet(true);
  };

  const openEditWallet = (wallet: Wallet) => {
    setManageWalletsOpen(false);
    setEditingWallet(wallet);
  };

  const openTransferWallet = (wallet: Wallet) => {
    setManageWalletsOpen(false);
    setTransferFromWallet(wallet);
  };

  const openDeleteWallet = (wallet: Wallet) => {
    setManageWalletsOpen(false);
    setPendingWalletDelete(wallet);
  };

  const openEditExpense = (expense: Expense) => {
    setManageExpensesOpen(false);
    setEditingExpense(expense);
  };

  const openDeleteExpense = (expense: Expense) => {
    setManageExpensesOpen(false);
    handleRequestDelete(expense);
  };

  const openEditIncome = (income: Income) => {
    setManageIncomesOpen(false);
    setEditingIncome(income);
  };

  const openDeleteIncome = (income: Income) => {
    setManageIncomesOpen(false);
    handleRequestIncomeDelete(income);
  };

  // load currencies when opening transfer modal
  useEffect(() => {
    let cancelled = false;
    if (!transferFromWallet) return;
    setLoadingCurrenciesForTransfer(true);
    void listCurrencies()
      .then((list) => {
        if (!cancelled) setCurrenciesForTransfer(list);
      })
      .catch(() => {
        if (!cancelled) setCurrenciesForTransfer([]);
      })
      .finally(() => {
        if (!cancelled) setLoadingCurrenciesForTransfer(false);
      });

    return () => {
      cancelled = true;
    };
  }, [transferFromWallet]);
  const {
    categories,
    wallets,
    isLoadingMeta,
    metaError,
    refreshMeta: refreshDashboardMeta,
    syncWallets: syncDashboardWallets,
  } = useDashboardMeta();

  useEffect(() => {
    let cancelled = false;

    async function loadExpenses() {
      setIsLoadingExpenses(true);

      try {
        const response = await listExpenses();
        if (!cancelled) {
          setExpenses(response.expenses);
        }
      } catch {
        if (!cancelled) {
          setError('Failed to load expenses.');
        }
      } finally {
        if (!cancelled) {
          setIsLoadingExpenses(false);
        }
      }
    }

    async function loadIncomes() {
      setIsLoadingIncomes(true);

      try {
        const response = await listIncomes();
        if (!cancelled) {
          setIncomes(response.incomes);
        }
      } catch {
        if (!cancelled) {
          setError('Failed to load incomes.');
        }
      } finally {
        if (!cancelled) {
          setIsLoadingIncomes(false);
        }
      }
    }

    void loadExpenses();
    void loadIncomes();

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function loadSummary() {
      setIsLoadingSummary(true);

      try {
        const response = await getDashboardSummary(settings.defaultCurrency);
        if (!cancelled) {
          setDashboardSummary(response);
        }
      } catch {
        if (!cancelled) {
          setSummaryError('Failed to load dashboard summary.');
        }
      } finally {
        if (!cancelled) {
          setIsLoadingSummary(false);
        }
      }
    }

    void loadSummary();

    return () => {
      cancelled = true;
    };
  }, [settings.defaultCurrency]);

  useEffect(() => {
    let cancelled = false;

    async function loadWidgets() {
      setIsLoadingWidgets(true);

      try {
        const response = await getDashboardWidgets(settings.defaultCurrency);
        if (!cancelled) {
          setDashboardWidgets(response);
        }
      } catch {
        if (!cancelled) {
          setWidgetsError('Failed to load dashboard widgets.');
        }
      } finally {
        if (!cancelled) {
          setIsLoadingWidgets(false);
        }
      }
    }

    void loadWidgets();

    return () => {
      cancelled = true;
    };
  }, [settings.defaultCurrency]);

  useEffect(() => {
    let cancelled = false;

    async function loadTrend() {
      const option = TREND_RANGE_OPTIONS.find((item) => item.value === trendRange) ?? TREND_RANGE_OPTIONS[0];
      const buckets = buildRangeBuckets(trendRange);
      const selectedCurrency = settings.defaultCurrency;
      const rateCache = new Map<string, number>();
      const bucketTotals = new Map<string, number>();

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

      for (const expense of expenses) {
        const itemDate = new Date(expense.date);
        if (Number.isNaN(itemDate.getTime())) {
          continue;
        }

        const bucket = buckets.find(({ start, end }) => itemDate >= start && itemDate < end);
        if (!bucket) {
          continue;
        }

        const converted = await convertToSelectedCurrency(expense.amount, expense.currency);
        bucketTotals.set(bucket.key, (bucketTotals.get(bucket.key) ?? 0) - converted);
      }

      for (const income of incomes) {
        const itemDate = new Date(income.income_date);
        if (Number.isNaN(itemDate.getTime())) {
          continue;
        }

        const bucket = buckets.find(({ start, end }) => itemDate >= start && itemDate < end);
        if (!bucket) {
          continue;
        }

        const converted = await convertToSelectedCurrency(income.amount, income.currency);
        bucketTotals.set(bucket.key, (bucketTotals.get(bucket.key) ?? 0) + converted);
      }

      let cumulative = 0;
      const rawPoints = buckets.map((bucket) => {
        cumulative += bucketTotals.get(bucket.key) ?? 0;
        return {
          value: cumulative,
          label: bucket.label,
        };
      });

      if (!cancelled) {
        setTrendPoints(rawPoints);
      }
    }

    async function loadWeeklyExpenses() {
      setIsLoadingWeeklyExpenses(true);
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

      const now = new Date();
      const currentYear = now.getFullYear();
      const monthBuckets = buildMonthBuckets(currentYear);
      const bucketByKey = new Map(monthBuckets.map((item) => [item.key, item]));
      const walletById = new Map(wallets.map((wallet) => [wallet.id, wallet]));

      for (const expense of expenses) {
        const itemDate = new Date(expense.date);
        if (Number.isNaN(itemDate.getTime())) {
          continue;
        }

        const bucketKey = toMonthKey(itemDate);
        const bucket = bucketByKey.get(bucketKey);
        if (!bucket) {
          continue;
        }

        const converted = await convertToSelectedCurrency(expense.amount, expense.currency);
        const wallet = walletById.get(expense.wallet_id);
        const walletType = wallet?.wallet_type?.toUpperCase() ?? '';

        if (walletType === 'BANK') {
          bucket.bank += converted;
        } else if (walletType === 'CARD' || walletType === 'MOBILE_WALLET') {
          bucket.digital += converted;
        } else {
          bucket.cash += converted;
        }
      }

      const normalized = monthBuckets.map((item) => ({
        label: item.label,
        cash: Number(item.cash.toFixed(2)),
        bank: Number(item.bank.toFixed(2)),
        digital: Number(item.digital.toFixed(2)),
      }));

      if (!cancelled) {
        setWeeklyExpensePoints(normalized);
        setIsLoadingWeeklyExpenses(false);
      }
    }

    async function loadMonthlyCashFlow() {
      setIsLoadingMonthlyCashFlow(true);
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

      const currentYear = new Date().getFullYear();
      const monthBuckets = buildMonthBuckets(currentYear);
      const bucketByKey = new Map(monthBuckets.map((item) => [item.key, item]));

      for (const expense of expenses) {
        const itemDate = new Date(expense.date);
        if (Number.isNaN(itemDate.getTime())) {
          continue;
        }

        const bucket = bucketByKey.get(toMonthKey(itemDate));
        if (!bucket) {
          continue;
        }

        const converted = await convertToSelectedCurrency(expense.amount, expense.currency);
        bucket.expenses += converted;
      }

      for (const income of incomes) {
        const itemDate = new Date(income.income_date);
        if (Number.isNaN(itemDate.getTime())) {
          continue;
        }

        const bucket = bucketByKey.get(toMonthKey(itemDate));
        if (!bucket) {
          continue;
        }

        const converted = await convertToSelectedCurrency(income.amount, income.currency);
        bucket.income += converted;
      }

      const normalized = monthBuckets.map((item) => ({
        month: item.key,
        label: item.label,
        income: Number(item.income.toFixed(2)),
        expenses: Number(item.expenses.toFixed(2)),
      }));

      if (!cancelled) {
        setMonthlyCashFlowPoints(normalized);
        setIsLoadingMonthlyCashFlow(false);
      }
    }

    void Promise.all([loadTrend(), loadWeeklyExpenses(), loadMonthlyCashFlow()]);

    return () => {
      cancelled = true;
    };
  }, [trendRange, expenses, incomes, wallets, settings.defaultCurrency]);

  const refreshMeta = async () => {
    await refreshDashboardMeta();
    await refreshSummary();
    // Clear any active edit when meta is refreshed to avoid stale selections
    setEditingExpense(null);
    setEditingIncome(null);
  };

  const refreshSummary = async () => {
    setSummaryError(null);
    setIsLoadingSummary(true);

    try {
      const response = await getDashboardSummary(settings.defaultCurrency);
      setDashboardSummary(response);
    } catch {
      setSummaryError('Failed to refresh dashboard summary.');
    } finally {
      setIsLoadingSummary(false);
    }
  };

  const refreshExpenses = async () => {
    setError(null);
    setIsLoadingExpenses(true);

    try {
      const response = await listExpenses();
      setExpenses(response.expenses);
    } catch {
      setError('Failed to refresh expenses.');
    } finally {
      setIsLoadingExpenses(false);
    }
  };

  const refreshIncomes = async () => {
    setError(null);
    setIsLoadingIncomes(true);

    try {
      const response = await listIncomes();
      setIncomes(response.incomes);
    } catch {
      setError('Failed to refresh incomes.');
    } finally {
      setIsLoadingIncomes(false);
    }
  };

  const showUndoToast = (
    title: string,
    subtitle: string,
    onUndo: () => Promise<void>,
    undoSuccessMessage: string,
    undoErrorMessage: string,
  ) => {
    toast(
      (t) => (
        <div className="flex items-center gap-3">
          <div className="min-w-0">
            <p className="text-sm font-semibold text-text-primary">{title}</p>
            <p className="truncate text-xs text-text-muted">{subtitle}</p>
          </div>
          <div className="ml-4 flex-shrink-0">
            <Button
              variant="secondary"
              type="button"
              onClick={async () => {
                toast.dismiss(t.id);
                try {
                  await onUndo();
                  toast.success(undoSuccessMessage);
                } catch {
                  toast.error(undoErrorMessage);
                }
              }}
            >
              Undo
            </Button>
          </div>
        </div>
      ),
      { duration: 5000 },
    );
  };

  const handleCreateExpense = async (data: CreateExpenseRequest) => {
    const created = await createExpense(data);
    setExpenses((current) => [created, ...current]);
    setEditingExpense(null);
    setManageExpensesOpen(false);
    void syncDashboardWallets();
    void refreshSummary();
    toast.success('Expense created');
  };

  const handleUpdateExpense = async (data: CreateExpenseRequest) => {
    if (!editingExpense) {
      return;
    }

    const updated = await updateExpense(editingExpense.id, data);
    setExpenses((current) => current.map((expense) => (expense.id === updated.id ? updated : expense)));
    setEditingExpense(null);
    void syncDashboardWallets();
    void refreshSummary();
    toast.success('Expense updated');
  };

  const handleDeleteExpense = async (expense: Expense) => {
    setDeletingExpenseId(expense.id);
    setError(null);

    try {
      await deleteExpense(expense.id);
      setExpenses((current) => current.filter((item) => item.id !== expense.id));
      if (editingExpense?.id === expense.id) {
        setEditingExpense(null);
      }
      void syncDashboardWallets();
      void refreshSummary();
      return true;
    } catch {
      setError('Failed to delete expense.');
      return false;
    } finally {
      setDeletingExpenseId(null);
    }
  };

  const handleCreateIncome = async (data: CreateIncomeRequest) => {
    const created = await createIncome(data);
    setIncomes((current) => [created, ...current]);
    setEditingIncome(null);
    setManageIncomesOpen(false);
    await syncDashboardWallets();
    await refreshSummary();
    toast.success('Income created');
  };

  const handleUpdateIncome = async (data: CreateIncomeRequest) => {
    if (!editingIncome) {
      return;
    }

    const updated = await updateIncome(editingIncome.id, data);
    setIncomes((current) => current.map((income) => (income.id === updated.id ? updated : income)));
    setEditingIncome(null);
    await syncDashboardWallets();
    await refreshSummary();
    toast.success('Income updated');
  };

  const handleDeleteIncome = async (income: Income) => {
    setDeletingIncomeId(income.id);
    setError(null);

    try {
      await deleteIncome(income.id);
      setIncomes((current) => current.filter((item) => item.id !== income.id));
      if (editingIncome?.id === income.id) {
        setEditingIncome(null);
      }
      await syncDashboardWallets();
      await refreshSummary();
      return true;
    } catch {
      setError('Failed to delete income.');
      return false;
    } finally {
      setDeletingIncomeId(null);
    }
  };

  const handleRequestDelete = (expense: Expense) => {
    setPendingDelete(expense);
  };

  const handleRequestIncomeDelete = (income: Income) => {
    setPendingIncomeDelete(income);
  };

  const confirmPendingDelete = async () => {
    if (!pendingDelete) return;
    const toDelete = pendingDelete;
    setPendingDelete(null);

    const ok = await handleDeleteExpense(toDelete);
    if (!ok) {
      return;
    }

    showUndoToast(
      'Expense deleted',
      toDelete.merchant ?? 'Unnamed expense',
      async () => {
        const restored = await createExpense({
          wallet_id: toDelete.wallet_id,
          amount: toDelete.amount,
          currency: toDelete.currency,
          category_id: toDelete.category_id,
          merchant: toDelete.merchant ?? undefined,
          date: toDelete.date,
          notes: toDelete.notes ?? undefined,
          fx_rate_to_base: toDelete.fx_rate_to_base ?? 1,
          is_recurring: toDelete.is_recurring ?? false,
          recurring_rule: toDelete.recurring_rule ?? undefined,
        });

        setExpenses((cur) => [restored, ...cur]);
        void syncDashboardWallets();
      },
      'Restored expense',
      'Failed to restore expense',
    );
  };

  const confirmPendingIncomeDelete = async () => {
    if (!pendingIncomeDelete) return;
    const toDelete = pendingIncomeDelete;
    setPendingIncomeDelete(null);

    const ok = await handleDeleteIncome(toDelete);
    if (!ok) {
      return;
    }

    showUndoToast(
      'Income deleted',
      toDelete.source_name ?? 'Unnamed income',
      async () => {
        const restored = await createIncome({
          wallet_id: toDelete.wallet_id,
          amount: toDelete.amount,
          currency: toDelete.currency,
          category_id: toDelete.category_id ?? undefined,
          source_name: toDelete.source_name,
          income_date: toDelete.income_date,
          notes: toDelete.notes ?? undefined,
        });

        setIncomes((cur) => [restored, ...cur]);
        void syncDashboardWallets();
      },
      'Restored income',
      'Failed to restore income',
    );
  };

  const cancelPendingDelete = () => setPendingDelete(null);

  const currencyForFormatting = dashboardSummary?.base_currency ?? settings.defaultCurrency;
  const formatCurrencyByCode = (amount: number, currencyCode: string) => {
    try {
      return new Intl.NumberFormat(settings.locale, { style: 'currency', currency: currencyCode }).format(amount);
    } catch {
      return `${amount.toFixed(2)} ${currencyCode}`;
    }
  };

  const totalBalance = dashboardSummary?.total_balance ?? 0;
  const monthlyExpenses = dashboardSummary?.monthly_expenses ?? 0;
  const monthlyIncomes = dashboardSummary?.monthly_income ?? 0;
  const monthlySavings = dashboardSummary?.monthly_savings ?? Math.max(monthlyIncomes - monthlyExpenses, 0);
  const netThisMonth = dashboardSummary?.net_this_month ?? (monthlyIncomes - monthlyExpenses);
  const safeToSpendValue = dashboardSummary?.safe_to_spend ?? (totalBalance + netThisMonth);
  const monthlySpendingPercentChange = dashboardSummary?.monthly_spending_percent_change ?? null;

  const kpiTotalBalance = formatCurrency(totalBalance, currencyForFormatting, settings.locale);
  const kpiTotalBalanceTrend = `Net this month ${formatCurrency(netThisMonth, currencyForFormatting, settings.locale)}`;
  const kpiTotalBalanceDirection = netThisMonth >= 0 ? 'up' as const : 'down' as const;

  const kpiMonthlySpending = formatCurrency(monthlyExpenses, currencyForFormatting, settings.locale);
  const kpiMonthlySpendingTrend = monthlySpendingPercentChange === null ? '—' : `${monthlySpendingPercentChange >= 0 ? '+' : ''}${monthlySpendingPercentChange.toFixed(1)}% from last month`;
  const kpiMonthlySpendingDirection = monthlySpendingPercentChange === null ? 'down' as const : (monthlySpendingPercentChange > 0 ? 'down' as const : 'up' as const);

  const kpiSafeToSpend = formatCurrency(safeToSpendValue, currencyForFormatting, settings.locale);
  const kpiSafeToSpendTrend = `Net this month ${formatCurrency(netThisMonth, currencyForFormatting, settings.locale)}`;
  const kpiSafeToSpendDirection = netThisMonth >= 0 ? 'up' as const : 'down' as const;

  const spendingByDay = dashboardSummary?.daily_spending ?? [];

  const maxDailySpending = Math.max(...spendingByDay.map((entry) => entry.total), 1);
  const spendingBars = spendingByDay.map((entry) => ({
    label: entry.label,
    value: Math.max(Math.round((entry.total / maxDailySpending) * 100), entry.total > 0 ? 12 : 8),
  }));

  const walletRows = wallets.slice(0, 3).map((wallet) => {
    const opening = Number(wallet.opening_balance ?? 0);
    const current = Number(wallet.current_balance ?? 0);
    const delta = current - opening;
    const deltaPercent = opening === 0 ? null : (delta / Math.abs(opening)) * 100;
    const deltaLabel = deltaPercent === null
      ? `${formatCurrencyByCode(delta, wallet.currency)} since opening`
      : `${delta >= 0 ? '+' : ''}${deltaPercent.toFixed(1)}% since opening`;

    return {
      name: wallet.name,
      balance: formatCurrencyByCode(current, wallet.currency),
      change: deltaLabel,
    };
  });

  const categoryBreakdown = (dashboardSummary?.category_breakdown ?? [])
    .slice()
    .sort((a, b) => b.total - a.total)
    .slice(0, 4);
  const maxCategoryTotal = Math.max(...categoryBreakdown.map((item) => item.total), 1);
  const budgetRows = categoryBreakdown.length > 0
    ? categoryBreakdown.map((item, index) => ({
        name: item.name,
        spent: Math.max(Math.round((item.total / maxCategoryTotal) * 100), 10),
        color: ['bg-accent-blue', 'bg-accent-green', 'bg-accent-amber', 'bg-accent-purple'][index % 4],
      }))
    : [
        { name: 'No spending yet', spent: 0, color: 'bg-accent-blue' },
      ];

  const nearlyCompleteGoalsDynamic = budgetRows.filter((row) => row.spent >= 75).length;
  const activeGoalsCount = budgetRows.length;
  const nearlyCompleteGoals = nearlyCompleteGoalsDynamic;
  const kpiActiveGoals = String(activeGoalsCount);
  const kpiActiveGoalsTrend = `${nearlyCompleteGoals} goals nearly complete`;
  const kpiActiveGoalsDirection = nearlyCompleteGoals > 0 ? 'up' as const : 'up' as const;

  const latestTrendPoint = trendPoints[trendPoints.length - 1] ?? null;
  const latestTrendValue = latestTrendPoint?.value ?? 0;
  const latestTrendLabel = latestTrendPoint?.label ?? 'Today';
  const previousTrendValue = trendPoints[trendPoints.length - 2]?.value ?? null;
  const latestTrendChangePercent = previousTrendValue && previousTrendValue !== 0
    ? ((latestTrendValue - previousTrendValue) / Math.abs(previousTrendValue)) * 100
    : null;
  const latestTrendDirection = latestTrendChangePercent === null
    ? 'up'
    : latestTrendChangePercent >= 0
      ? 'up'
      : 'down';
  const selectedTrendOption = TREND_RANGE_OPTIONS.find((item) => item.value === trendRange) ?? TREND_RANGE_OPTIONS[0];
  const trendPrimaryValue = trendPoints.length > 0 ? latestTrendValue : totalBalance;
  const trendValueDelta = latestTrendChangePercent === null ? null : `${latestTrendChangePercent >= 0 ? '+' : ''}${latestTrendChangePercent.toFixed(2)}%`;
  const trendHeaderLabel = selectedTrendOption.label;
  const trendArrowDirection = latestTrendDirection === 'up' ? 'up' : 'down';
  const monthlyBudgetRows = (dashboardWidgets?.budgets ?? []).slice(0, 5);
  const categoryMetaByID = new Map(
    categories.map((category, index) => [category.id, { name: category.name, color: category.color ?? ['#6366F1', '#10B981', '#F59E0B', '#A855F7'][index % 4] }]),
  );
  const savingGoalRows = monthlyBudgetRows.slice(0, 4).map((budget, index) => ({
    id: budget.id,
    title: budget.category_name,
    progress: Math.min(Math.max(budget.usage_percent, 0), 100),
    color: budget.category_color ?? ['#6366F1', '#10B981', '#F59E0B', '#A855F7'][index % 4],
  }));
  const transactionHistoryRows = [
    ...expenses.map((expense) => {
      const date = new Date(expense.date);
      const category = categoryMetaByID.get(expense.category_id);

      return {
        id: `expense-${expense.id}`,
        category: category?.name || 'Expense',
        categoryColor: category?.color || '#6366F1',
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
    ...incomes.map((income) => {
      const date = new Date(income.income_date);
      const category = income.category_id ? categoryMetaByID.get(income.category_id) : undefined;

      return {
        id: `income-${income.id}`,
        category: category?.name || 'Income',
        categoryColor: category?.color || '#10B981',
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

  const isLoadingDashboard = isLoadingSummary && !dashboardSummary;
  const displayError = error ?? metaError ?? summaryError ?? widgetsError;

  return (
    <Layout>
      <Header user={user} onLogout={onLogout} />

    {/* KPI Cards */}
      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <KPICard label="Total balance" value={kpiTotalBalance} trend={kpiTotalBalanceTrend} direction={kpiTotalBalanceDirection} tone="blue" />
        <KPICard label="Monthly spending" value={kpiMonthlySpending} trend={kpiMonthlySpendingTrend} direction={kpiMonthlySpendingDirection} tone="amber" />
        <KPICard label="Safe to spend" value={kpiSafeToSpend} trend={kpiSafeToSpendTrend} direction={kpiSafeToSpendDirection} tone="green" />
        <KPICard label="Active goals" value={kpiActiveGoals} trend={kpiActiveGoalsTrend} direction={kpiActiveGoalsDirection} tone="purple" />
      </section>

    {/* Cash Flow Overview */}
      <section className="grid gap-4 xl:grid-cols-[1.35fr_0.85fr]">
        <Card
          title="Cash flow overview"
          subtitle="Monthly inflow versus outflow"
          action={<Button variant="secondary">This month</Button>}
          footer={<p className="text-sm text-text-secondary">Monthly spending stays within target when incomes outpace expenses.</p>}
        >
          <div className="grid gap-5 lg:grid-cols-[1.2fr_0.8fr]">
            <div>
              <div className="flex h-64 items-end gap-3 rounded-3xl border border-dark-elevated bg-dark-bg p-4">
                {spendingBars.map((bar) => (
                  <div key={bar.label} className="flex h-full flex-1 flex-col items-center gap-2">
                    <div className="flex h-full w-full items-end justify-center">
                      <div
                        className="w-full max-w-10 rounded-t-2xl bg-gradient-to-t from-accent-blue to-accent-green shadow-lg shadow-accent-blue/10"
                        style={{ height: `${bar.value}%` }}
                      />
                    </div>
                    <p className="text-xs text-text-muted">{bar.label}</p>
                  </div>
                ))}
              </div>
              <div className="mt-4 grid gap-3 sm:grid-cols-2">
                <div className="rounded-2xl border border-dark-elevated bg-dark-bg p-4">
                  <p className="text-xs font-semibold uppercase tracking-[0.14em] text-text-muted">Income</p>
                  <p className="mt-2 font-mono text-xl font-semibold text-accent-green">{formatCurrency(monthlyIncomes, currencyForFormatting, settings.locale)}</p>
                </div>
                <div className="rounded-2xl border border-dark-elevated bg-dark-bg p-4">
                  <p className="text-xs font-semibold uppercase tracking-[0.14em] text-text-muted">Spent</p>
                  <p className="mt-2 font-mono text-xl font-semibold text-accent-amber">{formatCurrency(monthlyExpenses, currencyForFormatting, settings.locale)}</p>
                </div>
                <div className="rounded-2xl border border-dark-elevated bg-dark-bg p-4 sm:col-span-2">
                  <p className="text-xs font-semibold uppercase tracking-[0.14em] text-text-muted">Saved</p>
                  <p className="mt-2 font-mono text-xl font-semibold text-accent-blue">{formatCurrency(monthlySavings, currencyForFormatting, settings.locale)}</p>
                </div>
              </div>
            </div>

            <div className="space-y-3">
              {budgetRows.map((budget) => (
                <div key={budget.name} className="rounded-2xl border border-dark-elevated bg-dark-bg p-4">
                  <div className="flex items-center justify-between gap-3 text-sm">
                    <p className="font-semibold text-text-primary">{budget.name}</p>
                    <p className="text-text-muted">{budget.spent}% used</p>
                  </div>
                  <div className="mt-3 h-2 overflow-hidden rounded-full bg-dark-elevated">
                    <div className={`h-full rounded-full ${budget.color}`} style={{ width: `${budget.spent}%` }} />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </Card>

        <div className="grid gap-4">
          <Card title="Quick actions" subtitle="Common money tasks">
            <div className="grid gap-3">
              <Button onClick={() => setManageExpensesOpen(true)}>+ Add expense</Button>
              <Button variant="secondary" onClick={() => setManageIncomesOpen(true)}>+ Add income</Button>
              <Button variant="secondary" onClick={() => {
                if (wallets.length === 0) {
                  toast.error('Create a wallet first');
                  return;
                }

                setTransferFromWallet(wallets[0]);
              }}>Send transfer</Button>
            </div>
          </Card>

          
          <Card title="Wallets" subtitle="Current balances and alerts">
            <div className="space-y-3">
              {walletRows.map((account) => (
                <div key={account.name} className="flex items-center justify-between gap-4 rounded-2xl border border-dark-elevated bg-dark-bg px-4 py-3">
                  <div>
                    <p className="font-semibold text-text-primary">{account.name}</p>
                    <p className="mt-1 text-xs text-text-muted">{account.change}</p>
                  </div>
                  <p className="font-mono text-lg font-semibold text-text-primary">{account.balance}</p>
                </div>
              ))}
            </div>
          </Card>

        </div>
      </section>

      
      <section className="grid gap-4 lg:grid-cols-4">
        <Card className="overflow-hidden lg:col-span-3">
          <div className="flex flex-col gap-5 p-6">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div className="space-y-1">
                <p className="text-lg font-medium leading-none tracking-tight text-text-primary">Balance Trends</p>
                <p className="text-sm text-text-muted">{isLoadingDashboard ? 'Loading trend data' : `${trendHeaderLabel} in ${currencyForFormatting}`}</p>
              </div>

              <div className="flex flex-col items-start gap-3 sm:items-end">
                <div className="flex items-center overflow-hidden rounded-full border border-dark-elevated bg-dark-bg p-1">
                  {TREND_RANGE_OPTIONS.map((option) => {
                    const active = option.value === trendRange;

                    return (
                      <button
                        key={option.value}
                        type="button"
                        onClick={() => setTrendRange(option.value)}
                        className={[
                          'rounded-full px-3 py-1.5 text-sm font-medium transition',
                          active
                            ? 'bg-accent-blue text-white shadow-sm'
                            : 'text-text-muted hover:bg-dark-elevated hover:text-text-primary',
                        ].join(' ')}
                        aria-pressed={active}
                      >
                        {option.label}
                      </button>
                    );
                  })}
                </div>

                <div className={`flex items-center ${latestTrendDirection === 'up' ? 'text-emerald-600' : 'text-rose-500'}`}>
                  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-4 w-4">
                    {trendArrowDirection === 'up' ? (
                      <><path d="m5 12 7-7 7 7" /><path d="M12 19V5" /></>
                    ) : (
                      <><path d="m5 12 7 7 7-7" /><path d="M12 5v14" /></>
                    )}
                  </svg>
                  <span className="text-sm font-medium">{trendValueDelta ?? '0.00%'}</span>
                </div>
              </div>
            </div>

            <div>
              <p className="text-2xl font-bold text-text-primary">{isLoadingDashboard ? '...' : formatCurrency(trendPrimaryValue, currencyForFormatting, settings.locale)}</p>
            </div>

            <div className="h-[330px] w-full pt-4">
              {trendPoints.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={trendPoints} margin={{ top: 5, right: 0, left: 0, bottom: 0 }}>
                    <defs>
                      <linearGradient id="colorValue" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="rgb(99, 102, 241)" stopOpacity={0.3} />
                        <stop offset="95%" stopColor="rgb(99, 102, 241)" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid stroke="#E5E7EB" strokeDasharray="3 3" vertical={false} />
                    <XAxis
                      dataKey="label"
                      tick={{ fill: '#6B7280', fontSize: 12 }}
                      axisLine={false}
                      tickLine={false}
                      interval="preserveStartEnd"
                    />
                    <YAxis
                      tick={{ fill: '#6B7280', fontSize: 12 }}
                      axisLine={false}
                      tickLine={false}
                      width={52}
                      tickFormatter={formatTrendAxisValue}
                    />
                    <Tooltip
                      formatter={(value) => [formatCurrency(typeof value === 'number' ? value : 0, currencyForFormatting, settings.locale), 'Balance']}
                      labelStyle={{ color: '#111827' }}
                      contentStyle={{
                        background: 'rgba(17, 24, 39, 0.96)',
                        border: '1px solid rgba(148, 163, 184, 0.2)',
                        borderRadius: '16px',
                        color: '#F9FAFB',
                        boxShadow: '0 20px 40px rgba(15, 23, 42, 0.3)',
                      }}
                    />
                    <Area
                      type="monotone"
                      dataKey="value"
                      stroke="rgb(99, 102, 241)"
                      strokeWidth={2}
                      fill="url(#colorValue)"
                      dot={false}
                      activeDot={{ r: 4, strokeWidth: 2, stroke: 'rgb(99, 102, 241)', fill: '#fff' }}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              ) : (
                <div className="flex h-full items-center justify-center rounded-2xl border border-dashed border-dark-elevated text-sm text-text-muted">
                  No spending data yet
                </div>
              )}
            </div>

            <div className="flex flex-wrap items-center gap-3 text-sm text-text-muted">
              <span>Latest point: {latestTrendLabel}</span>
              <span className="h-1 w-1 rounded-full bg-text-muted/50" />
              <span>{formatCurrency(latestTrendValue, currencyForFormatting, settings.locale)}</span>
            </div>
          </div>
        </Card>

        <Card className="overflow-hidden lg:col-span-1">
          <div className="flex h-full flex-col gap-4 p-1 sm:p-2 md:p-0">
            <div>
              <p className="text-lg font-medium leading-none tracking-tight text-text-primary">Monthly Expenses</p>
              <p className="mt-1 text-sm text-text-muted">{isLoadingDashboard ? 'Loading categories' : 'Category split for the current month'}</p>
            </div>

            <div className="h-2 w-full overflow-hidden rounded-full bg-dark-bg">
              {categoryBreakdown.length > 0 ? (
                categoryBreakdown.map((item, index) => (
                  <div
                    key={item.category_id || item.name}
                    className={['h-full', ['bg-accent-purple', 'bg-accent-blue', 'bg-accent-green', 'bg-accent-amber'][index % 4]].join(' ')}
                    style={{ width: `${Math.max((item.total / Math.max(monthlyExpenses, 1)) * 100, 6)}%` }}
                  />
                ))
              ) : (
                <div className="h-full w-full bg-dark-elevated" />
              )}
            </div>

            <div className="space-y-0">
              {categoryBreakdown.length > 0 ? categoryBreakdown.map((item, index) => (
                <div key={item.category_id || item.name} className="flex items-center justify-between border-b border-dark-elevated py-3 last:border-b-0">
                  <div className="flex items-center gap-2">
                    <div className={['h-3 w-3 rounded-full', ['bg-accent-purple', 'bg-accent-blue', 'bg-accent-green', 'bg-accent-amber'][index % 4]].join(' ')} />
                    <span className="text-sm text-text-muted">{item.name}</span>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="text-sm font-medium text-text-primary">{formatCurrency(item.total, currencyForFormatting, settings.locale)}</span>
                    <span className="w-10 text-right text-sm text-text-muted">{monthlyExpenses > 0 ? `${Math.round((item.total / monthlyExpenses) * 100)}%` : '0%'}</span>
                  </div>
                </div>
              )) : (
                <p className="py-4 text-sm text-text-muted">No expenses recorded yet.</p>
              )}
            </div>
          </div>
        </Card>
      </section>

      <section className="grid gap-4 lg:grid-cols-4">
        <MonthlyBudgetsCard
          className="lg:col-span-1"
          isLoading={isLoadingWidgets}
          rows={monthlyBudgetRows}
          locale={settings.locale}
        />

        <MonthlyIncomeExpensesChartCard
          className="lg:col-span-3"
          isLoading={isLoadingMonthlyCashFlow}
          points={monthlyCashFlowPoints}
          currencyCode={currencyForFormatting}
          locale={settings.locale}
          formatAxisValue={formatTrendAxisValue}
        />
      </section>

      <section className="grid gap-4 lg:grid-cols-[0.9fr_1.1fr]">
        <SavingGoalsCard
          className="h-full"
          goals={savingGoalRows}
        />

        <TransactionHistoryCard
          transactions={transactionHistoryRows}
          onSeeMore={() => setManageExpensesOpen(true)}
        />
      </section>


      {editingExpense && (
        <Modal title="Edit expense" onClose={() => setEditingExpense(null)}>
          <ExpenseForm
            categories={categories}
            wallets={wallets}
            onSubmit={handleUpdateExpense}
            initialExpense={editingExpense}
            onCancel={() => setEditingExpense(null)}
          />
        </Modal>
      )}

      {editingIncome && (
        <Modal title="Edit income" onClose={() => setEditingIncome(null)}>
          <IncomeForm
            categories={categories}
            wallets={wallets}
            onSubmit={handleUpdateIncome}
            initialIncome={editingIncome}
            onCancel={() => setEditingIncome(null)}
          />
        </Modal>
      )}

      {pendingDelete && (
        <Modal title="Confirm delete" onClose={cancelPendingDelete}>
          <p className="text-sm text-text-secondary">Are you sure you want to delete this expense? This action cannot be undone.</p>
          <div className="mt-4 flex items-center gap-3">
            <Button variant="secondary" onClick={cancelPendingDelete}>Cancel</Button>
            <Button onClick={() => void confirmPendingDelete()}>{deletingExpenseId === pendingDelete.id ? 'Deleting...' : 'Delete'}</Button>
          </div>
        </Modal>
      )}

      {pendingIncomeDelete && (
        <Modal title="Confirm delete" onClose={() => setPendingIncomeDelete(null)}>
          <p className="text-sm text-text-secondary">Are you sure you want to delete this income? This action cannot be undone.</p>
          <div className="mt-4 flex items-center gap-3">
            <Button variant="secondary" onClick={() => setPendingIncomeDelete(null)}>Cancel</Button>
            <Button onClick={() => void confirmPendingIncomeDelete()}>{deletingIncomeId === pendingIncomeDelete.id ? 'Deleting...' : 'Delete'}</Button>
          </div>
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
                void refreshMeta();
                setEditingWallet(null);
              } catch {
                toast.error('Failed to update wallet');
              }
            }}
            onCancel={() => setEditingWallet(null)}
          />
        </Modal>
      )}

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

      {pendingWalletDelete && (
        <Modal title="Confirm delete" onClose={() => setPendingWalletDelete(null)}>
          <p className="text-sm text-text-secondary">Are you sure you want to delete this wallet? This will remove the wallet and may affect balances.</p>
          <div className="mt-4 flex items-center gap-3">
            <Button variant="secondary" onClick={() => setPendingWalletDelete(null)}>Cancel</Button>
            <Button
              onClick={async () => {
                if (!pendingWalletDelete) return;
                setDeletingWalletId(pendingWalletDelete.id);
                try {
                  await deleteWallet(pendingWalletDelete.id);
                  toast.success('Wallet deleted');
                  void refreshMeta();
                } catch {
                  toast.error('Failed to delete wallet');
                } finally {
                  setDeletingWalletId(null);
                  setPendingWalletDelete(null);
                }
              }}
            >
              {deletingWalletId === pendingWalletDelete.id ? 'Deleting...' : 'Delete'}
            </Button>
          </div>
        </Modal>
      )}

      {transferFromWallet && (
        <Modal title={`Transfer from ${transferFromWallet.name}`} onClose={() => setTransferFromWallet(null)}>
          <WalletTransferForm
            wallets={wallets}
            currencies={currenciesForTransfer}
            loadingCurrencies={loadingCurrenciesForTransfer}
            from={transferFromWallet}
            onSubmit={async (data) => {
              try {
                await createWalletTransfer({ ...data, transfer_date: new Date().toISOString() });

                const fromW = wallets.find((w) => w.id === data.from_wallet_id);
                const toW = wallets.find((w) => w.id === data.to_wallet_id);
                const subtitle = `${fromW?.name ?? 'Wallet'} → ${toW?.name ?? 'Wallet'}`;

                showUndoToast(
                  'Transfer created',
                  subtitle,
                  async () => {
                    // create reverse transfer
                    try {
                      let reverseAmount = data.amount;
                      const reverseCurrency = toW?.currency ?? data.currency;
                      if (fromW && toW && fromW.currency !== toW.currency) {
                        const conv = await convertCurrency(data.amount, data.currency, toW.currency);
                        reverseAmount = conv.converted_amount;
                      }

                      await createWalletTransfer({
                        from_wallet_id: data.to_wallet_id,
                        to_wallet_id: data.from_wallet_id,
                        amount: reverseAmount,
                        currency: reverseCurrency,
                        fee_amount: 0,
                        transfer_date: new Date().toISOString(),
                      });

                      void refreshMeta();
                      void syncDashboardWallets();
                    } catch (e) {
                      throw e;
                    }
                  },
                  'Transfer undone',
                  'Failed to undo transfer',
                );

                void refreshMeta();
                void syncDashboardWallets();
                setTransferFromWallet(null);
              } catch {
                toast.error('Failed to create transfer');
              }
            }}
            onCancel={() => setTransferFromWallet(null)}
          />
        </Modal>
      )}

      {manageWalletsOpen && (
        <Modal title="Manage wallets" onClose={() => setManageWalletsOpen(false)}>
          <div className="space-y-4">
            <div className="flex items-center justify-between gap-3 border-b border-dark-elevated pb-4">
              <div>
                <p className="text-sm font-semibold text-text-primary">Wallet actions</p>
                <p className="text-xs text-text-muted">Create a new wallet or manage existing ones here.</p>
              </div>
                <Button variant="primary" type="button" onClick={openCreateWallet}>
                Add wallet
              </Button>
            </div>

            <div className="space-y-3">
            {wallets.length === 0 ? (
              <p className="text-sm text-text-secondary">No wallets to manage.</p>
            ) : (
              wallets.map((w) => (
                <div key={w.id} className="flex items-center justify-between gap-3 rounded-2xl border border-dark-elevated bg-dark-bg px-4 py-3">
                  <div>
                    <p className="font-semibold text-text-primary">{w.name}</p>
                    <p className="mt-1 text-xs text-text-muted">{w.wallet_type} · {w.currency}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <p className="font-mono text-sm font-semibold text-text-primary">{formatCurrencyByCode(w.current_balance, w.currency)}</p>
                    <Button variant="secondary" onClick={() => openEditWallet(w)}>Edit</Button>
                    <Button variant="secondary" onClick={() => openTransferWallet(w)}>Transfer</Button>
                    <Button variant="secondary" onClick={() => openDeleteWallet(w)}>Delete</Button>
                  </div>
                </div>
              ))
            )}
            </div>
          </div>
        </Modal>
      )}

      {manageIncomesOpen && (
        <Modal title="Add income" onClose={() => setManageIncomesOpen(false)}>
          <IncomeForm
            categories={categories}
            wallets={wallets}
            onSubmit={handleCreateIncome}
            onCancel={() => setManageIncomesOpen(false)}
          />
        </Modal>
      )}

      {manageExpensesOpen && (
        <Modal title="Add expense" onClose={() => setManageExpensesOpen(false)}>
          <ExpenseForm
            categories={categories}
            wallets={wallets}
            onSubmit={handleCreateExpense}
            onCancel={() => setManageExpensesOpen(false)}
          />
        </Modal>
      )}

      {/* toasts rendered by ToastProvider */}
    </Layout>
  );
}