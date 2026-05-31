import type { ReactNode } from 'react';

type BadgeProps = {
  children: ReactNode;
  variant?: 'income' | 'expense' | 'info';
};

export function Badge({ children, variant = 'info' }: BadgeProps) {
  const variantClass =
    variant === 'income' ? 'badge-income' : variant === 'expense' ? 'badge-expense' : 'badge-info';

  return <span className={`badge ${variantClass}`}>{children}</span>;
}