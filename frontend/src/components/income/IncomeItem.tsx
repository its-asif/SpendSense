import { Badge } from '../common/Badge';
import { Button } from '../common/Button';
import { useUserSettings } from '../../hooks/useUserSettings';
import { formatCurrency } from '../../lib/userSettings';
import type { ExpenseCategory, Income, Wallet } from '../../types';

type IncomeItemProps = {
  income: Income;
  categories: ExpenseCategory[];
  wallets: Wallet[];
  onEdit: (income: Income) => void;
  onRequestDelete: (income: Income) => void;
  deleting?: boolean;
  showActions?: boolean;
};

export function IncomeItem({ income, categories, wallets, onEdit, onRequestDelete, deleting, showActions = true }: IncomeItemProps) {
  const settings = useUserSettings();
  const category = categories.find((item) => item.id === income.category_id);
  const wallet = wallets.find((item) => item.id === income.wallet_id);

  return (
    <div className="flex items-center justify-between gap-4 border-b border-dark-elevated py-3 last:border-b-0">
      <div className="min-w-0">
        <div className="flex items-center gap-3">
          <Badge variant="income">{category?.name ?? 'Uncategorized'}</Badge>
          <p className="truncate font-semibold text-text-primary">{income.source_name || 'Unnamed income'}</p>
        </div>
        <p className="mt-1 text-xs text-text-muted">
          {income.income_date} · {wallet?.name ?? 'Unknown wallet'}
        </p>
      </div>
      <div className="text-right">
        <p className="font-mono text-lg font-semibold text-accent-green">+{formatCurrency(Number(income.amount), income.currency, settings.locale)}</p>
        {showActions && (
          <div className="mt-3 flex items-center justify-end gap-2">
            <Button variant="secondary" type="button" onClick={() => onEdit(income)}>
              Edit
            </Button>
            <Button variant="secondary" type="button" onClick={() => onRequestDelete(income)} disabled={deleting}>
              {deleting ? 'Deleting...' : 'Delete'}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}