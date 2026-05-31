import { useEffect, useMemo, useState } from 'react';
import { Header } from '../components/layout/Header';
import { Layout } from '../components/layout/Layout';
import { Card } from '../components/common/Card';
import { listExpenses } from '../api/expenses';
import { listIncomes } from '../api/incomes';
import { useUserSettings } from '../hooks/useUserSettings';
import { formatCurrency } from '../lib/userSettings';
import type { AuthUser, Expense, Income } from '../types';

type ReportsPageProps = {
  user: AuthUser;
  onLogout: () => void;
};

export function ReportsPage({ user, onLogout }: ReportsPageProps) {
  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [incomes, setIncomes] = useState<Income[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const settings = useUserSettings();

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setIsLoading(true);
      try {
        const [expenseResponse, incomeResponse] = await Promise.all([listExpenses(), listIncomes()]);
        if (!cancelled) {
          setExpenses(expenseResponse.expenses);
          setIncomes(incomeResponse.incomes);
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, []);

  const now = new Date();
  const currentMonth = now.getMonth();
  const currentYear = now.getFullYear();

  const totals = useMemo(() => {
    const thisMonthIncome = incomes
      .filter((income) => {
        const date = new Date(income.income_date);
        return date.getMonth() === currentMonth && date.getFullYear() === currentYear;
      })
      .reduce((sum, income) => sum + Number(income.amount || 0), 0);

    const thisMonthExpense = expenses
      .filter((expense) => {
        const date = new Date(expense.date);
        return date.getMonth() === currentMonth && date.getFullYear() === currentYear;
      })
      .reduce((sum, expense) => sum + Number(expense.amount || 0), 0);

    return {
      income: thisMonthIncome,
      expense: thisMonthExpense,
      net: thisMonthIncome - thisMonthExpense,
    };
  }, [expenses, incomes, currentMonth, currentYear]);

  return (
    <Layout>
      <Header user={user} onLogout={onLogout} />

      <section className="grid gap-4 md:grid-cols-3">
        <Card title="Monthly income" subtitle="Current month total">
          <p className="font-mono text-2xl font-semibold text-accent-green">{isLoading ? '...' : formatCurrency(totals.income, settings.defaultCurrency, settings.locale)}</p>
        </Card>
        <Card title="Monthly expenses" subtitle="Current month total">
          <p className="font-mono text-2xl font-semibold text-accent-amber">{isLoading ? '...' : formatCurrency(totals.expense, settings.defaultCurrency, settings.locale)}</p>
        </Card>
        <Card title="Net cash flow" subtitle="Income minus expenses">
          <p className="font-mono text-2xl font-semibold text-accent-blue">{isLoading ? '...' : formatCurrency(totals.net, settings.defaultCurrency, settings.locale)}</p>
        </Card>
      </section>
    </Layout>
  );
}
