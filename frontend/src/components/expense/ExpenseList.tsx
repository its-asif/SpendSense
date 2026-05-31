import { Card } from '../common/Card';
import type { Expense, ExpenseCategory, Wallet } from '../../types';
import { ExpenseItem } from './ExpenseItem';

type ExpenseListProps = {
  expenses: Expense[];
  categories: ExpenseCategory[];
  wallets: Wallet[];
  isLoading?: boolean;
  deletingExpenseId?: string | null;
  onEdit: (expense: Expense) => void;
  onRequestDelete: (expense: Expense) => void;
  showActions?: boolean;
  action?: React.ReactNode;
};

export function ExpenseList({ expenses, categories, wallets, isLoading, deletingExpenseId, onEdit, onRequestDelete, showActions = true, action }: ExpenseListProps) {
  return (
    <Card title="Recent expenses" subtitle="Latest activity from the API" action={action}>
      {isLoading ? (
        <p className="text-sm text-text-secondary">Loading expenses...</p>
      ) : expenses.length === 0 ? (
        <p className="text-sm text-text-secondary">No expenses yet.</p>
      ) : (
        <div className="space-y-0.5">
          {expenses.map((expense) => (
            <ExpenseItem
              key={expense.id}
              expense={expense}
              categories={categories}
              wallets={wallets}
              onEdit={onEdit}
              onRequestDelete={onRequestDelete}
              deleting={deletingExpenseId === expense.id}
              showActions={showActions}
            />
          ))}
        </div>
      )}
    </Card>
  );
}