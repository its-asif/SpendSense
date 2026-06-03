import { useEffect, useMemo, useState } from 'react';
import toast from 'react-hot-toast';
import { createBudget, deleteBudget, listBudgets, updateBudget } from '../api/budgets';
import { listCategories } from '../api/categories';
import { CategorySelector } from '../components/expense/CategorySelector';
import { Button } from '../components/common/Button';
import { Card } from '../components/common/Card';
import { Input } from '../components/common/Input';
import Modal from '../components/common/Modal';
import { Header } from '../components/layout/Header';
import { Layout } from '../components/layout/Layout';
import { useUserSettings } from '../hooks/useUserSettings';
import { formatCurrency } from '../lib/userSettings';
import type { AuthUser, Budget, ExpenseCategory } from '../types';

type BudgetsPageProps = {
  user: AuthUser;
  onLogout: () => void;
};

function firstDayOfMonthISO(): string {
  const now = new Date();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  return `${now.getFullYear()}-${month}-01`;
}

function apiErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error !== null && 'response' in error) {
    const response = (error as { response?: { data?: { message?: string } } }).response;
    if (response?.data?.message) {
      return response.data.message;
    }
  }
  return fallback;
}

export function BudgetsPage({ user, onLogout }: BudgetsPageProps) {
  const settings = useUserSettings();
  const [budgets, setBudgets] = useState<Budget[]>([]);
  const [categories, setCategories] = useState<ExpenseCategory[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [categoryId, setCategoryId] = useState('');
  const [amount, setAmount] = useState('');
  const [editingBudget, setEditingBudget] = useState<Budget | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Budget | null>(null);

  const refresh = async () => {
    setIsLoading(true);
    try {
      const [budgetItems, categoryItems] = await Promise.all([listBudgets('MONTHLY'), listCategories()]);
      setBudgets(budgetItems);
      setCategories(categoryItems);
    } catch {
      toast.error('Failed to load budgets');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    void refresh();
  }, []);

  const budgetedCategoryIds = useMemo(() => new Set(budgets.map((budget) => budget.category_id)), [budgets]);

  const availableCategories = useMemo(
    () => categories.filter((category) => !budgetedCategoryIds.has(category.id)),
    [categories, budgetedCategoryIds],
  );

  const submitCreate = async (event: React.FormEvent) => {
    event.preventDefault();
    const parsedAmount = Number(amount);
    if (!categoryId || !Number.isFinite(parsedAmount) || parsedAmount <= 0) {
      toast.error('Choose a category and enter a valid amount');
      return;
    }

    try {
      const created = await createBudget({
        category_id: categoryId,
        amount: parsedAmount,
        currency: settings.defaultCurrency || user.baseCurrency || 'USD',
        period: 'MONTHLY',
        start_date: firstDayOfMonthISO(),
      });
      setBudgets((current) => [...current, created].sort((a, b) => a.category_name.localeCompare(b.category_name)));
      setCategoryId('');
      setAmount('');
      toast.success('Monthly budget created');
    } catch (error) {
      toast.error(apiErrorMessage(error, 'Failed to create budget'));
    }
  };

  return (
    <Layout>
      <Header user={user} onLogout={onLogout} />

      <section className="grid gap-4 xl:grid-cols-[1fr_1.2fr]">
        <Card title="Add monthly budget" subtitle="Set a spending limit per category for the current month">
          <form className="space-y-4" onSubmit={submitCreate}>
            <CategorySelector
              categories={availableCategories}
              value={categoryId}
              onChange={setCategoryId}
              disabled={availableCategories.length === 0}
            />
            {availableCategories.length === 0 && (
              <p className="text-sm text-text-muted">Every category already has a monthly budget, or no categories are available.</p>
            )}
            <Input
              label="Monthly limit"
              type="number"
              min="0.01"
              step="0.01"
              value={amount}
              onChange={(event) => setAmount(event.target.value)}
              required
            />
            <p className="text-xs text-text-muted">
              Currency: {settings.defaultCurrency || user.baseCurrency || 'USD'} · Period starts {firstDayOfMonthISO()}
            </p>
            <Button type="submit" disabled={availableCategories.length === 0}>
              Add budget
            </Button>
          </form>
        </Card>

        <Card title="Monthly budgets" subtitle="Limits shown on your dashboard">
          {isLoading ? (
            <p className="text-sm text-text-secondary">Loading budgets...</p>
          ) : budgets.length === 0 ? (
            <p className="text-sm text-text-secondary">No monthly budgets yet. Add one to track category spending.</p>
          ) : (
            <div className="space-y-3">
              {budgets.map((budget) => {
                const accent = budget.category_color ?? '#6366F1';
                const badge = (budget.category_icon?.trim()?.[0] ?? budget.category_name.trim()[0] ?? '?').toUpperCase();

                return (
                  <div key={budget.id} className="flex items-center justify-between gap-3 rounded-2xl border border-dark-elevated bg-dark-bg px-4 py-3">
                    <div className="flex min-w-0 items-center gap-3">
                      <div
                        className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white"
                        style={{ backgroundColor: accent }}
                      >
                        {badge}
                      </div>
                      <div className="min-w-0">
                        <p className="truncate font-semibold text-text-primary">{budget.category_name}</p>
                        <p className="mt-1 text-xs text-text-muted">
                          {formatCurrency(budget.amount, budget.currency, settings.locale)} / month
                        </p>
                      </div>
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                      <Button variant="secondary" onClick={() => setEditingBudget(budget)}>
                        Edit
                      </Button>
                      <Button variant="secondary" onClick={() => setPendingDelete(budget)}>
                        Delete
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </Card>
      </section>

      {editingBudget && (
        <Modal title="Edit monthly budget" onClose={() => setEditingBudget(null)}>
          <form
            className="space-y-4"
            onSubmit={async (event) => {
              event.preventDefault();
              const parsedAmount = Number(editingBudget.amount);
              if (!Number.isFinite(parsedAmount) || parsedAmount <= 0) {
                toast.error('Enter a valid amount');
                return;
              }

              try {
                const updated = await updateBudget(editingBudget.id, {
                  category_id: editingBudget.category_id,
                  amount: parsedAmount,
                  currency: editingBudget.currency,
                  period: 'MONTHLY',
                  start_date: editingBudget.start_date,
                  rollover_enabled: editingBudget.rollover_enabled,
                });
                setBudgets((current) =>
                  current
                    .map((item) => (item.id === updated.id ? updated : item))
                    .sort((a, b) => a.category_name.localeCompare(b.category_name)),
                );
                setEditingBudget(null);
                toast.success('Budget updated');
              } catch (error) {
                toast.error(apiErrorMessage(error, 'Failed to update budget'));
              }
            }}
          >
            <p className="text-sm text-text-secondary">Category: {editingBudget.category_name}</p>
            <Input
              label="Monthly limit"
              type="number"
              min="0.01"
              step="0.01"
              value={String(editingBudget.amount)}
              onChange={(event) => setEditingBudget({ ...editingBudget, amount: Number(event.target.value) })}
              required
            />
            <div className="flex items-center gap-3">
              <Button type="submit">Save</Button>
              <Button type="button" variant="secondary" onClick={() => setEditingBudget(null)}>
                Cancel
              </Button>
            </div>
          </form>
        </Modal>
      )}

      {pendingDelete && (
        <Modal title="Confirm delete" onClose={() => setPendingDelete(null)}>
          <p className="text-sm text-text-secondary">
            Delete the monthly budget for &quot;{pendingDelete.category_name}&quot;?
          </p>
          <div className="mt-4 flex items-center gap-3">
            <Button variant="secondary" onClick={() => setPendingDelete(null)}>
              Cancel
            </Button>
            <Button
              onClick={async () => {
                try {
                  await deleteBudget(pendingDelete.id);
                  setBudgets((current) => current.filter((item) => item.id !== pendingDelete.id));
                  setPendingDelete(null);
                  toast.success('Budget deleted');
                } catch {
                  toast.error('Failed to delete budget');
                }
              }}
            >
              Delete
            </Button>
          </div>
        </Modal>
      )}
    </Layout>
  );
}
