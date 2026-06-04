import React from 'react';
import { Button } from '../common/Button';
import type { RecurringPayment } from '../../api/recurring';

type RecurringPaymentListProps = {
  payments: RecurringPayment[];
  isLoading: boolean;
  onEdit: (payment: RecurringPayment) => void;
  onDelete: (payment: RecurringPayment) => void;
  onPay: (payment: RecurringPayment) => void;
};

export function RecurringPaymentList({
  payments,
  isLoading,
  onEdit,
  onDelete,
  onPay,
}: RecurringPaymentListProps) {
  if (isLoading) {
    return (
      <div className="flex h-32 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-accent-blue border-t-transparent" />
      </div>
    );
  }

  if (payments.length === 0) {
    return (
      <div className="flex h-32 items-center justify-center text-sm text-text-muted">
        No recurring payments found. Add one on the left to get started!
      </div>
    );
  }

  const todayStr = new Date().toISOString().slice(0, 10);

  const getStatusBadge = (payment: RecurringPayment) => {
    if (payment.status === 'inactive') {
      return (
        <span className="inline-flex items-center rounded-full bg-dark-elevated px-2.5 py-0.5 text-xs font-semibold text-text-muted">
          Inactive
        </span>
      );
    }

    if (todayStr > payment.deadline) {
      return (
        <span className="inline-flex items-center rounded-full bg-accent-red/10 px-2.5 py-0.5 text-xs font-semibold text-accent-red border border-accent-red/20 animate-pulse">
          ⚠️ Overdue
        </span>
      );
    }

    if (todayStr >= payment.start_date) {
      return (
        <span className="inline-flex items-center rounded-full bg-accent-amber/10 px-2.5 py-0.5 text-xs font-semibold text-accent-amber border border-accent-amber/20">
          Due
        </span>
      );
    }

    return (
      <span className="inline-flex items-center rounded-full bg-accent-green/10 px-2.5 py-0.5 text-xs font-semibold text-accent-green border border-accent-green/20">
        Upcoming
      </span>
    );
  };

  const formatAlertRule = (rule: string) => {
    switch (rule) {
      case 'start':
        return 'On start date';
      case '1h':
        return '1 Hour before';
      case '12h':
        return '12 Hours before';
      case '1d':
        return '1 Day before';
      case '7d':
        return '7 Days before';
      default:
        return rule;
    }
  };

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left border-collapse">
        <thead>
          <tr className="border-b border-dark-elevated text-xs font-semibold uppercase tracking-wider text-text-secondary">
            <th className="px-4 py-3">Title / Category</th>
            <th className="px-4 py-3">Amount</th>
            <th className="px-4 py-3">Wallet</th>
            <th className="px-4 py-3">Cycle Dates</th>
            <th className="px-4 py-3">Interval</th>
            <th className="px-4 py-3">Alert</th>
            <th className="px-4 py-3">Status</th>
            <th className="px-4 py-3 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-dark-elevated">
          {payments.map((payment) => {
            const hasEnded = payment.status === 'inactive';
            return (
              <tr key={payment.id} className="hover:bg-dark-bg/10 transition-colors">
                <td className="px-4 py-4">
                  <div className="flex items-center gap-3">
                    {payment.category_icon && (
                      <span
                        className="flex h-8 w-8 items-center justify-center rounded-lg text-sm"
                        style={{
                          backgroundColor: payment.category_color ? `${payment.category_color}20` : '#3B82F620',
                          color: payment.category_color || '#3B82F6',
                        }}
                      >
                        {payment.category_icon}
                      </span>
                    )}
                    <div>
                      <div className="font-semibold text-text-primary">{payment.title}</div>
                      <div className="text-xs text-text-muted">{payment.category_name}</div>
                    </div>
                  </div>
                </td>
                <td className="px-4 py-4 font-mono font-semibold text-text-primary">
                  {new Intl.NumberFormat('en-US', {
                    style: 'currency',
                    currency: payment.currency,
                  }).format(payment.amount)}
                </td>
                <td className="px-4 py-4 text-sm text-text-secondary">
                  {payment.wallet_name}
                </td>
                <td className="px-4 py-4 text-xs text-text-secondary">
                  <div>Start: {payment.start_date}</div>
                  <div className="font-semibold text-text-primary">Due: {payment.deadline}</div>
                  {payment.end_date && <div className="text-text-muted">Ends: {payment.end_date}</div>}
                </td>
                <td className="px-4 py-4 text-xs capitalize text-text-secondary">
                  {payment.interval}
                </td>
                <td className="px-4 py-4 text-xs text-text-muted">
                  {formatAlertRule(payment.alert_rule)}
                </td>
                <td className="px-4 py-4">
                  {getStatusBadge(payment)}
                </td>
                <td className="px-4 py-4 text-right">
                  <div className="flex items-center justify-end gap-2">
                    {!hasEnded && (
                      <Button
                        onClick={() => onPay(payment)}
                        className="py-1.5 px-3 text-xs bg-accent-blue hover:bg-accent-blue/90"
                      >
                        Pay
                      </Button>
                    )}
                    <button
                      onClick={() => onEdit(payment)}
                      className="p-1.5 text-text-muted hover:text-text-primary transition-colors rounded-lg hover:bg-dark-elevated"
                      title="Edit"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M12 20h9" />
                        <path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z" />
                      </svg>
                    </button>
                    <button
                      onClick={() => onDelete(payment)}
                      className="p-1.5 text-text-muted hover:text-accent-red transition-colors rounded-lg hover:bg-dark-elevated"
                      title="Delete"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M3 6h18" />
                        <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" />
                        <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
                      </svg>
                    </button>
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
