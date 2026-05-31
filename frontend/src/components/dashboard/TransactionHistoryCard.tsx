import { Badge } from '../common/Badge';
import { Card } from '../common/Card';
import { useUserSettings } from '../../hooks/useUserSettings';
import { formatCurrency } from '../../lib/userSettings';

type TransactionHistoryItem = {
  id: string;
  category: string;
  categoryColor: string;
  description: string;
  dateLabel: string;
  amount: number;
  currency: string;
  type: 'income' | 'expense';
  status?: 'Paid' | 'Due' | 'Cancel';
};

type TransactionHistoryCardProps = {
  transactions: TransactionHistoryItem[];
  onSeeMore?: () => void;
  className?: string;
};

export function TransactionHistoryCard({ transactions, onSeeMore, className }: TransactionHistoryCardProps) {
  const settings = useUserSettings();

  return (
    <Card
      className={className}
      title="Transaction History"
      subtitle="Recent income and expense activity"
      action={onSeeMore ? (
        <button type="button" onClick={onSeeMore} className="text-sm font-medium text-accent-blue hover:underline">
          See more
        </button>
      ) : undefined}
    >
      {transactions.length > 0 ? (
        <div className="overflow-hidden rounded-3xl border border-dark-elevated bg-dark-bg">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-dark-elevated text-left">
              <thead className="bg-dark-bg/80">
                <tr className="text-xs uppercase tracking-[0.14em] text-text-muted">
                  <th className="px-4 py-3 font-medium">Category</th>
                  <th className="px-4 py-3 font-medium">Date</th>
                  <th className="px-4 py-3 font-medium">Description</th>
                  <th className="px-4 py-3 font-medium text-right">Amount</th>
                  <th className="px-4 py-3 font-medium text-right">Currency</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-dark-elevated">
                {transactions.map((transaction) => {
                  const amountClass = transaction.type === 'income' ? 'text-accent-green' : 'text-accent-amber';

                  return (
                    <tr key={transaction.id} className="bg-dark-bg/40 text-sm text-text-primary">
                      <td className="px-4 py-4">
                        <div className="flex items-center gap-3">
                          <span
                            className="flex h-9 w-9 items-center justify-center rounded-full text-xs font-semibold text-white"
                            style={{ backgroundColor: transaction.categoryColor }}
                          >
                            {transaction.category.slice(0, 1).toUpperCase()}
                          </span>
                          <div className="min-w-0">
                            <p className="font-semibold text-text-primary">{transaction.category}</p>
                            {transaction.status ? (
                              <Badge variant={transaction.status === 'Paid' ? 'income' : transaction.status === 'Due' ? 'info' : 'expense'}>
                                {transaction.status}
                              </Badge>
                            ) : null}
                          </div>
                        </div>
                      </td>
                      <td className="px-4 py-4 text-text-muted">{transaction.dateLabel}</td>
                      <td className="px-4 py-4 text-text-secondary">{transaction.description}</td>
                      <td className={`px-4 py-4 text-right font-mono font-semibold ${amountClass}`}>
                        {transaction.type === 'income' ? '+' : '-'}{formatCurrency(transaction.amount, transaction.currency, settings.locale)}
                      </td>
                      <td className="px-4 py-4 text-right text-text-muted">{transaction.currency}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <div className="flex h-56 items-center justify-center rounded-3xl border border-dashed border-dark-elevated text-sm text-text-muted">
          No transactions found.
        </div>
      )}
    </Card>
  );
}
