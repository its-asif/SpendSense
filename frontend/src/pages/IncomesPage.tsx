import { useEffect, useMemo, useState } from 'react';
import toast from 'react-hot-toast';
import { Header } from '../components/layout/Header';
import { Layout } from '../components/layout/Layout';
import { Card } from '../components/common/Card';
import { Button } from '../components/common/Button';
import { Input } from '../components/common/Input';
import Modal from '../components/common/Modal';
import { IncomeForm } from '../components/income/IncomeForm';
import { IncomeList } from '../components/income/IncomeList';
import { createIncome, deleteIncome, listIncomes, updateIncome } from '../api/incomes';
import { useDashboardMeta } from '../hooks/useDashboardMeta';
import type { AuthUser, CreateIncomeRequest, Income } from '../types';

const ITEMS_PER_PAGE = 8;

type IncomesPageProps = {
  user: AuthUser;
  onLogout: () => void;
};

export function IncomesPage({ user, onLogout }: IncomesPageProps) {
  const [incomes, setIncomes] = useState<Income[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingIncome, setEditingIncome] = useState<Income | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Income | null>(null);
  const [deletingIncomeId, setDeletingIncomeId] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const { categories, wallets, syncWallets } = useDashboardMeta();

  const refreshIncomes = async () => {
    setError(null);
    setIsLoading(true);

    try {
      const response = await listIncomes(500);
      setIncomes(response.incomes);
    } catch {
      setError('Failed to load incomes.');
    } finally {
      setIsLoading(false);
    }
  };

  const filteredIncomes = useMemo(() => {
    const query = searchTerm.trim().toLowerCase();
    if (!query) {
      return incomes;
    }

    return incomes.filter((income) => {
      const category = categories.find((item) => item.id === income.category_id);
      const wallet = wallets.find((item) => item.id === income.wallet_id);
      return [income.source_name, income.notes, income.income_date, category?.name, wallet?.name, income.currency]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(query));
    });
  }, [incomes, searchTerm, categories, wallets]);

  const totalPages = Math.max(Math.ceil(filteredIncomes.length / ITEMS_PER_PAGE), 1);
  const pageIncomes = filteredIncomes.slice((currentPage - 1) * ITEMS_PER_PAGE, currentPage * ITEMS_PER_PAGE);

  useEffect(() => {
    void refreshIncomes();
  }, []);

  const handleCreateIncome = async (data: CreateIncomeRequest) => {
    const created = await createIncome(data);
    setIncomes((current) => [created, ...current]);
    setCurrentPage(1);
    void syncWallets();
    toast.success('Income created');
  };

  const handleUpdateIncome = async (data: CreateIncomeRequest) => {
    if (!editingIncome) {
      return;
    }

    const updated = await updateIncome(editingIncome.id, data);
    setIncomes((current) => current.map((income) => (income.id === updated.id ? updated : income)));
    setEditingIncome(null);
    void syncWallets();
    toast.success('Income updated');
  };

  const handleDeleteIncome = async (income: Income) => {
    setDeletingIncomeId(income.id);

    try {
      await deleteIncome(income.id);
      setIncomes((current) => current.filter((item) => item.id !== income.id));
      setPendingDelete(null);
      if (editingIncome?.id === income.id) {
        setEditingIncome(null);
      }
      void syncWallets();
      toast.success('Income deleted');
    } catch {
      setError('Failed to delete income.');
      toast.error('Failed to delete income');
    } finally {
      setDeletingIncomeId(null);
    }
  };

  return (
    <Layout>
      <Header user={user} onLogout={onLogout} />

      <section className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        <Card title="Add income" subtitle="Record a new income transaction">
          {error && <p className="mb-4 text-sm text-accent-red">{error}</p>}
          <IncomeForm categories={categories} wallets={wallets} onSubmit={handleCreateIncome} />
        </Card>

        <Card
          title="Incomes"
          subtitle={`${filteredIncomes.length} result${filteredIncomes.length === 1 ? '' : 's'}`}
          action={<Button variant="secondary" onClick={() => void refreshIncomes()}>Refresh</Button>}
        >
          <div className="mb-4 grid gap-3 md:grid-cols-[1fr_auto] md:items-end">
            <Input
              label="Search"
              value={searchTerm}
              onChange={(event) => {
                setSearchTerm(event.target.value);
                setCurrentPage(1);
              }}
              placeholder="Search source, category, wallet..."
            />
            <p className="text-xs text-text-muted">Page {currentPage} of {totalPages}</p>
          </div>

          <IncomeList
            incomes={pageIncomes}
            categories={categories}
            wallets={wallets}
            isLoading={isLoading}
            deletingIncomeId={deletingIncomeId}
            onEdit={(income) => setEditingIncome(income)}
            onRequestDelete={(income) => setPendingDelete(income)}
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
        <Modal title="Confirm delete" onClose={() => setPendingDelete(null)}>
          <p className="text-sm text-text-secondary">Are you sure you want to delete this income?</p>
          <div className="mt-4 flex items-center gap-3">
            <Button variant="secondary" onClick={() => setPendingDelete(null)}>Cancel</Button>
            <Button onClick={() => void handleDeleteIncome(pendingDelete)}>
              {deletingIncomeId === pendingDelete.id ? 'Deleting...' : 'Delete'}
            </Button>
          </div>
        </Modal>
      )}
    </Layout>
  );
}
