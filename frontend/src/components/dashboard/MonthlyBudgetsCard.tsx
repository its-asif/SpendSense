import { Card } from '../common/Card';
import { formatCurrency } from '../../lib/userSettings';
import type { DashboardBudgetRow } from '../../types';

type MonthlyBudgetsCardProps = {
  isLoading: boolean;
  rows: DashboardBudgetRow[];
  locale: string;
  className?: string;
};

const budgetAccentColors = ['#10B981', '#06B6D4', '#3B82F6', '#8B5CF6', '#6366F1'];

export function MonthlyBudgetsCard({ isLoading, rows, locale, className }: MonthlyBudgetsCardProps) {
  return (
    <Card className={className} title="Monthly Budgets" subtitle="Current month budget usage">
      {isLoading ? (
        <p className="text-sm text-text-muted">Loading budgets...</p>
      ) : rows.length > 0 ? (
        <div className="space-y-5">
          {rows.map((budget, index) => {
            const accentColor = budget.category_color ?? budgetAccentColors[index % budgetAccentColors.length];
            const progress = Math.min(Math.max(budget.usage_percent, 0), 100);
            const badgeLabel = (budget.category_icon?.trim()?.[0] ?? budget.category_name.trim()[0] ?? '?').toUpperCase();

            return (
              <div key={budget.id} className="space-y-3">
                <div className="flex items-center gap-3">
                  <div className="flex h-9 w-9 items-center justify-center rounded-full text-xs font-bold text-white shadow-sm" style={{ backgroundColor: accentColor }}>
                    {badgeLabel}
                  </div>
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-text-primary">{budget.category_name}</p>
                    <p className="text-xs text-text-muted">
                      {formatCurrency(budget.spent, budget.currency, locale)} / {formatCurrency(budget.limit, budget.currency, locale)}
                    </p>
                  </div>
                  <span className="ml-auto text-sm text-text-muted">{Math.round(progress)}%</span>
                </div>
                <div className="h-2 w-full overflow-hidden rounded-full bg-dark-bg">
                  <div
                    className={`h-full rounded-full transition-all ${progress >= 100 ? 'bg-rose-500' : ''}`}
                    style={{ width: `${progress}%`, backgroundColor: progress >= 100 ? undefined : accentColor }}
                  />
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        <p className="text-sm text-text-muted">No monthly budgets found.</p>
      )}
    </Card>
  );
}
