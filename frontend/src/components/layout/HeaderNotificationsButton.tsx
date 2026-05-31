import { useState } from 'react';

type HeaderNotificationsButtonProps = {
  count?: number;
};

const demoNotifications = [
  {
    id: '1',
    title: 'Budget warning',
    message: "Dining expenses reached 82% of this month's limit.",
    time: '2m ago',
  },
  {
    id: '2',
    title: 'Salary received',
    message: 'Your monthly income has been added to Savings Wallet.',
    time: '1h ago',
  },
  {
    id: '3',
    title: 'Transfer completed',
    message: 'BDT 3,000 moved from Main Wallet to Emergency Fund.',
    time: 'Yesterday',
  },
];

export function HeaderNotificationsButton({ count = 3 }: HeaderNotificationsButtonProps) {
  const [open, setOpen] = useState(false);

  return (
    <div className="relative">
      <button
        type="button"
        className="relative inline-flex h-9 w-9 items-center justify-center rounded-full transition-colors hover:bg-transparent"
        aria-label={`Notifications, ${count} unread`}
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-4 w-4">
          <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9" />
          <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" />
        </svg>
        <div className="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full border border-transparent bg-destructive text-xs font-semibold text-destructive-foreground">
          {count}
        </div>
      </button>

      {open && (
        <div className="absolute right-0 top-11 z-50 w-80 overflow-hidden rounded-3xl border border-dark-elevated bg-dark-bg shadow-[0_22px_50px_rgba(15,23,42,0.35)]">
          <div className="flex items-center justify-between border-b border-dark-elevated px-4 py-3">
            <div>
              <p className="text-sm font-semibold text-text-primary">Notifications</p>
              <p className="text-xs text-text-muted">Demo activity feed</p>
            </div>
            <span className="rounded-full bg-dark-elevated px-2 py-1 text-xs font-semibold text-text-primary">
              {count} new
            </span>
          </div>

          <div className="max-h-80 divide-y divide-dark-elevated overflow-auto">
            {demoNotifications.map((notification) => (
              <button key={notification.id} type="button" className="block w-full px-4 py-3 text-left transition-colors hover:bg-dark-elevated/40">
                <div className="flex items-start gap-3">
                  <span className="mt-1 h-2.5 w-2.5 shrink-0 rounded-full bg-accent-blue" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center justify-between gap-3">
                      <p className="truncate text-sm font-semibold text-text-primary">{notification.title}</p>
                      <span className="shrink-0 text-xs text-text-muted">{notification.time}</span>
                    </div>
                    <p className="mt-1 text-sm leading-5 text-text-secondary">{notification.message}</p>
                  </div>
                </div>
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}