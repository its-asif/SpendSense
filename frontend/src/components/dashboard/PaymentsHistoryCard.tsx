import { Card } from '../common/Card';

type PaymentStatus = 'Paid' | 'Due' | 'Cancel';

type PaymentHistoryItem = {
  id: string;
  title: string;
  dateLabel: string;
  amountLabel: string;
  status?: PaymentStatus;
};

type PaymentsHistoryCardProps = {
  items: PaymentHistoryItem[];
  onSeeMore?: () => void;
  className?: string;
};

function statusClass(status: PaymentStatus) {
  if (status === 'Paid') {
    return 'bg-emerald-500/10 text-emerald-500';
  }
  if (status === 'Due') {
    return 'bg-yellow-500/10 text-yellow-500';
  }
  return 'bg-red-500/10 text-red-500';
}

export function PaymentsHistoryCard({ items, onSeeMore, className }: PaymentsHistoryCardProps) {
  return (
    <Card
      className={className}
      title="Payments History"
      action={(
        <button
          type="button"
          className="text-sm text-accent-blue hover:underline"
          onClick={onSeeMore}
        >
          See more
        </button>
      )}
    >
      <div className="grid gap-4">
        {items.length > 0 ? items.map((item) => (
          <div key={item.id} className="grid grid-cols-[1fr,auto] items-center gap-4">
            <div className="flex flex-col gap-0.5">
              <span className="font-medium text-text-primary">{item.title}</span>
              <span className="text-xs text-text-muted">{item.dateLabel}</span>
            </div>
            <div className="text-right">
              <div className="font-medium text-text-primary">{item.amountLabel}</div>
              {item.status ? (
                <span className={`mt-1 inline-flex rounded-md px-2.5 py-0.5 text-xs font-normal ${statusClass(item.status)}`}>
                  {item.status}
                </span>
              ) : null}
            </div>
          </div>
        )) : (
          <p className="text-sm text-text-muted">No payment activity yet.</p>
        )}
      </div>
    </Card>
  );
}
