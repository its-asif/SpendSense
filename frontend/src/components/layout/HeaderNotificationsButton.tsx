import { useCallback, useEffect, useState } from 'react';
import { dismissNotification, listNotifications, markAllNotificationsRead, markNotificationRead } from '../../api/notifications';
import { postRecurringExpense } from '../../api/expenses';
import type { AppNotification } from '../../types';

function formatRelativeTime(iso: string): string {
  const created = new Date(iso);
  const diffMs = Date.now() - created.getTime();
  const minutes = Math.floor(diffMs / 60000);
  if (minutes < 1) {
    return 'Just now';
  }
  if (minutes < 60) {
    return `${minutes}m ago`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `${hours}h ago`;
  }
  const days = Math.floor(hours / 24);
  return days === 1 ? 'Yesterday' : `${days}d ago`;
}

export function HeaderNotificationsButton() {
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<AppNotification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [isLoading, setIsLoading] = useState(false);

  const refresh = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await listNotifications(30);
      setItems(response.notifications);
      setUnreadCount(response.unread_count);
    } catch {
      setItems([]);
      setUnreadCount(0);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    if (open) {
      void refresh();
    }
  }, [open, refresh]);

  useEffect(() => {
    const handleRefresh = () => {
      // 800ms delay to allow async backend checks to complete
      const timer = setTimeout(() => {
        void refresh();
      }, 800);
      return () => clearTimeout(timer);
    };

    window.addEventListener('spendsense-refresh-notifications', handleRefresh);
    return () => {
      window.removeEventListener('spendsense-refresh-notifications', handleRefresh);
    };
  }, [refresh]);

  return (
    <div className="relative">
      <button
        type="button"
        className="relative inline-flex h-9 w-9 items-center justify-center rounded-full transition-colors hover:bg-dark-elevated/50"
        aria-label={`Notifications, ${unreadCount} unread`}
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-4 w-4">
          <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9" />
          <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" />
        </svg>
        {unreadCount > 0 && (
          <div className="absolute -right-1 -top-1 flex h-5 min-w-5 items-center justify-center rounded-full border border-transparent bg-destructive px-1 text-xs font-semibold text-destructive-foreground">
            {unreadCount > 9 ? '9+' : unreadCount}
          </div>
        )}
      </button>

      {open && (
        <div className="absolute right-0 top-11 z-50 w-80 overflow-hidden rounded-3xl border border-dark-elevated bg-dark-bg shadow-[0_22px_50px_rgba(15,23,42,0.35)]">
          <div className="flex items-center justify-between border-b border-dark-elevated px-4 py-3">
            <div>
              <p className="text-sm font-semibold text-text-primary">Notifications</p>
              <p className="text-xs text-text-muted">Budget alerts, recurring & loan reminders</p>
            </div>
            {unreadCount > 0 && (
              <button
                type="button"
                className="text-xs font-semibold text-accent-blue hover:underline"
                onClick={async () => {
                  await markAllNotificationsRead();
                  await refresh();
                }}
              >
                Mark all read
              </button>
            )}
          </div>

          <div className="max-h-80 divide-y divide-dark-elevated overflow-auto">
            {isLoading ? (
              <p className="px-4 py-6 text-sm text-text-muted">Loading...</p>
            ) : items.length === 0 ? (
              <p className="px-4 py-6 text-sm text-text-muted">No notifications yet.</p>
            ) : (
              items.map((notification) => (
                <div
                  key={notification.id}
                  className={`px-4 py-3 ${notification.is_read ? 'opacity-80' : 'bg-dark-elevated/20'}`}
                >
                  <div className="flex items-start gap-3">
                    <span className={`mt-1 h-2.5 w-2.5 shrink-0 rounded-full ${notification.is_read ? 'bg-dark-elevated' : 'bg-accent-blue'}`} />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center justify-between gap-3">
                        <p className="truncate text-sm font-semibold text-text-primary">{notification.title}</p>
                        <span className="shrink-0 text-xs text-text-muted">{formatRelativeTime(notification.created_at)}</span>
                      </div>
                      <p className="mt-1 text-sm leading-5 text-text-secondary">{notification.body}</p>
                      <div className="mt-2 flex gap-2">
                        {notification.type === 'recurring_due' && (
                          <button
                            type="button"
                            className="text-xs font-semibold text-accent-green hover:underline"
                            onClick={() => {
                              window.location.href = '/recurring';
                            }}
                          >
                            Manage & Pay
                          </button>
                        )}
                        {!notification.is_read && (
                          <button
                            type="button"
                            className="text-xs font-semibold text-accent-blue hover:underline"
                            onClick={async () => {
                              await markNotificationRead(notification.id);
                              await refresh();
                            }}
                          >
                            Mark read
                          </button>
                        )}
                        <button
                          type="button"
                          className="text-xs font-semibold text-text-muted hover:underline"
                          onClick={async () => {
                            await dismissNotification(notification.id);
                            await refresh();
                          }}
                        >
                          Dismiss
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}
