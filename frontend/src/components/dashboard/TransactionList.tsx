import type { Transaction } from '../../types';
import { Card } from '../common/Card';
import { TransactionItem } from './TransactionItem';

type TransactionListProps = {
  transactions: Transaction[];
};

export function TransactionList({ transactions }: TransactionListProps) {
  return (
    <Card title="Recent Transactions" subtitle="Latest activity from your demo account">
      <div className="space-y-0.5">
        {transactions.map((transaction) => (
          <TransactionItem key={transaction.id} transaction={transaction} />
        ))}
      </div>
    </Card>
  );
}