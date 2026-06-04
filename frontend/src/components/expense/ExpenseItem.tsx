import { Badge } from '../common/Badge';
import { Button } from '../common/Button';
import { useUserSettings } from '../../hooks/useUserSettings';
import { formatCurrency } from '../../lib/userSettings';
import type { Expense, ExpenseCategory, Wallet } from '../../types';

type ExpenseItemProps = {
  expense: Expense;
  categories: ExpenseCategory[];
  wallets: Wallet[];
  onEdit: (expense: Expense) => void;
  // Request delete (open confirmation modal) — the actual delete happens elsewhere
  onRequestDelete: (expense: Expense) => void;
  deleting?: boolean;
  showActions?: boolean;
};

export function ExpenseItem({ expense, categories, wallets, onEdit, onRequestDelete, deleting, showActions = true }: ExpenseItemProps) {
  const settings = useUserSettings();
  const category = categories.find((item) => item.id === expense.category_id);
  const wallet = wallets.find((item) => item.id === expense.wallet_id);

  return (
    <div className="flex items-center justify-between gap-4 border-b border-dark-elevated py-3 last:border-b-0">
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-3">
          <Badge variant="expense">{category?.name ?? 'Uncategorized'}</Badge>
          <p className="truncate font-semibold text-text-primary">{expense.merchant || 'Unnamed expense'}</p>
          {expense.receipt_url && (
            <a
              href={`${import.meta.env.VITE_API_URL || 'http://localhost:8080'}${expense.receipt_url}`}
              target="_blank"
              rel="noreferrer"
              title="View Receipt"
              className="text-accent-blue hover:underline text-xs inline-flex items-center gap-1 font-semibold border border-dark-elevated bg-dark-bg/50 px-2 py-0.5 rounded-full"
              onClick={(e) => e.stopPropagation()}
            >
              📎 Receipt
            </a>
          )}
        </div>
        <p className="mt-1 text-xs text-text-muted">
          {expense.date} · {wallet?.name ?? 'Unknown wallet'}
        </p>
      </div>
      <div className="text-right">
        <p className="font-mono text-lg font-semibold text-accent-amber">-{formatCurrency(Number(expense.amount), expense.currency, settings.locale)}</p>
        <p className="mt-1 text-xs text-text-muted">FX {expense.fx_rate_to_base.toFixed(2)}</p>
        {showActions && (
          <div className="mt-3 flex items-center justify-end gap-2">
            <Button variant="secondary" type="button" onClick={() => onEdit(expense)}>
              Edit
            </Button>
            <Button variant="secondary" type="button" onClick={() => onRequestDelete(expense)} disabled={deleting}>
              {deleting ? 'Deleting...' : 'Delete'}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}