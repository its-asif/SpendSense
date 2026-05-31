import { Badge } from '../common/Badge';
import { useUserSettings } from '../../hooks/useUserSettings';
import { formatCurrency } from '../../lib/userSettings';
import type { Transaction } from '../../types';

type TransactionItemProps = {
  transaction: Transaction;
};

export function TransactionItem({ transaction }: TransactionItemProps) {
  const settings = useUserSettings();
  const amountClass = transaction.type === 'income' ? 'text-accent-green' : 'text-accent-amber';

  return (
    <div className="flex items-center justify-between gap-4 border-b border-dark-elevated py-3 last:border-b-0">
      <div className="min-w-0">
        <div className="flex items-center gap-3">
          <Badge variant={transaction.type === 'income' ? 'income' : 'expense'}>{transaction.category}</Badge>
          <p className="truncate font-semibold text-text-primary">{transaction.description}</p>
        </div>
        <p className="mt-1 text-xs text-text-muted">{transaction.date}</p>
      </div>
      <p className={`font-mono text-lg font-semibold ${amountClass}`}>
        {transaction.type === 'income' ? '+' : '-'}{formatCurrency(transaction.amount / 100, settings.defaultCurrency, settings.locale)}
      </p>
    </div>
  );
}