import { Card } from '../common/Card';
import type { ExpenseCategory, Income, Wallet } from '../../types';
import { IncomeItem } from './IncomeItem';

type IncomeListProps = {
  incomes: Income[];
  categories: ExpenseCategory[];
  wallets: Wallet[];
  isLoading?: boolean;
  deletingIncomeId?: string | null;
  onEdit: (income: Income) => void;
  onRequestDelete: (income: Income) => void;
  showActions?: boolean;
  action?: React.ReactNode;
};

export function IncomeList({ incomes, categories, wallets, isLoading, deletingIncomeId, onEdit, onRequestDelete, showActions = true, action }: IncomeListProps) {
  return (
    <Card title="Recent incomes" subtitle="Latest income activity from the API" action={action}>
      {isLoading ? (
        <p className="text-sm text-text-secondary">Loading incomes...</p>
      ) : incomes.length === 0 ? (
        <p className="text-sm text-text-secondary">No incomes yet.</p>
      ) : (
        <div className="space-y-0.5">
          {incomes.map((income) => (
            <IncomeItem
              key={income.id}
              income={income}
              categories={categories}
              wallets={wallets}
              onEdit={onEdit}
              onRequestDelete={onRequestDelete}
              deleting={deletingIncomeId === income.id}
              showActions={showActions}
            />
          ))}
        </div>
      )}
    </Card>
  );
}