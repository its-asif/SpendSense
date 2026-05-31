import { useEffect, useMemo, useState } from 'react';
import toast from 'react-hot-toast';
import { Header } from '../components/layout/Header';
import { Layout } from '../components/layout/Layout';
import { Card } from '../components/common/Card';
import { Button } from '../components/common/Button';
import { Input } from '../components/common/Input';
import Modal from '../components/common/Modal';
import { ExpenseForm } from '../components/expense/ExpenseForm';
import { ExpenseList } from '../components/expense/ExpenseList';
import { createExpense, deleteExpense, listExpenses, updateExpense } from '../api/expenses';
import { useDashboardMeta } from '../hooks/useDashboardMeta';
import type { AuthUser, CreateExpenseRequest, Expense } from '../types';

const ITEMS_PER_PAGE = 8;

type ExpensesPageProps = {
  user: AuthUser;
  onLogout: () => void;
};

export function ExpensesPage({ user, onLogout }: ExpensesPageProps) {
  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingExpense, setEditingExpense] = useState<Expense | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Expense | null>(null);
  const [deletingExpenseId, setDeletingExpenseId] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const { categories, wallets, syncWallets } = useDashboardMeta();

  const refreshExpenses = async () => {
    setError(null);
    setIsLoading(true);

    try {
      const response = await listExpenses(500);
      setExpenses(response.expenses);
    } catch {
      setError('Failed to load expenses.');
    } finally {
      setIsLoading(false);
    }
  };

  const filteredExpenses = useMemo(() => {
    const query = searchTerm.trim().toLowerCase();
    if (!query) {
      return expenses;
    }

    return expenses.filter((expense) => {
      const category = categories.find((item) => item.id === expense.category_id);
      const wallet = wallets.find((item) => item.id === expense.wallet_id);
      return [expense.merchant, expense.notes, expense.date, category?.name, wallet?.name, expense.currency]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(query));
    });
  }, [expenses, searchTerm, categories, wallets]);

  const totalPages = Math.max(Math.ceil(filteredExpenses.length / ITEMS_PER_PAGE), 1);
  const pageExpenses = filteredExpenses.slice((currentPage - 1) * ITEMS_PER_PAGE, currentPage * ITEMS_PER_PAGE);

  useEffect(() => {
    void refreshExpenses();
  }, []);

  const handleCreateExpense = async (data: CreateExpenseRequest) => {
    const created = await createExpense(data);
    setExpenses((current) => [created, ...current]);
    setCurrentPage(1);
    void syncWallets();
    toast.success('Expense created');
  };

  const handleUpdateExpense = async (data: CreateExpenseRequest) => {
    if (!editingExpense) {
      return;
    }

    const updated = await updateExpense(editingExpense.id, data);
    setExpenses((current) => current.map((expense) => (expense.id === updated.id ? updated : expense)));
    setEditingExpense(null);
    void syncWallets();
    toast.success('Expense updated');
  };

  const handleDeleteExpense = async (expense: Expense) => {
    setDeletingExpenseId(expense.id);

    try {
      await deleteExpense(expense.id);
      setExpenses((current) => current.filter((item) => item.id !== expense.id));
      setPendingDelete(null);
      if (editingExpense?.id === expense.id) {
        setEditingExpense(null);
      }
      void syncWallets();
      toast.success('Expense deleted');
    } catch {
      setError('Failed to delete expense.');
      toast.error('Failed to delete expense');
    } finally {
      setDeletingExpenseId(null);
    }
  };

  return (
    <Layout>
      <Header user={user} onLogout={onLogout} />

      <section className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        <Card title="Add expense" subtitle="Create a new transaction">
          {error && <p className="mb-4 text-sm text-accent-red">{error}</p>}
          <ExpenseForm categories={categories} wallets={wallets} onSubmit={handleCreateExpense} />
        </Card>

        <Card
          title="Expenses"
          subtitle={`${filteredExpenses.length} result${filteredExpenses.length === 1 ? '' : 's'}`}
          action={<Button variant="secondary" onClick={() => void refreshExpenses()}>Refresh</Button>}
        >
          <div className="mb-4 grid gap-3 md:grid-cols-[1fr_auto] md:items-end">
            <Input
              label="Search"
              value={searchTerm}
              onChange={(event) => {
                setSearchTerm(event.target.value);
                setCurrentPage(1);
              }}
              placeholder="Search merchant, category, wallet..."
            />
            <p className="text-xs text-text-muted">Page {currentPage} of {totalPages}</p>
          </div>

          <ExpenseList
            expenses={pageExpenses}
            categories={categories}
            wallets={wallets}
            isLoading={isLoading}
            deletingExpenseId={deletingExpenseId}
            onEdit={(expense) => setEditingExpense(expense)}
            onRequestDelete={(expense) => setPendingDelete(expense)}
          />

          <div className="mt-4 flex items-center justify-between gap-3">
            <Button variant="secondary" onClick={() => setCurrentPage((current) => Math.max(current - 1, 1))} disabled={currentPage === 1}>
              Previous
            </Button>
            <Button variant="secondary" onClick={() => setCurrentPage((current) => Math.min(current + 1, totalPages))} disabled={currentPage >= totalPages}>
              Next
            </Button>
          </div>
        </Card>
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

      {pendingDelete && (
        <Modal title="Confirm delete" onClose={() => setPendingDelete(null)}>
          <p className="text-sm text-text-secondary">Are you sure you want to delete this expense?</p>
          <div className="mt-4 flex items-center gap-3">
            <Button variant="secondary" onClick={() => setPendingDelete(null)}>Cancel</Button>
            <Button onClick={() => void handleDeleteExpense(pendingDelete)}>
              {deletingExpenseId === pendingDelete.id ? 'Deleting...' : 'Delete'}
            </Button>
          </div>
        </Modal>
      )}
    </Layout>
  );
}
