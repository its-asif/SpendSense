import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { Card } from '../common/Card';
import { formatCurrency } from '../../lib/userSettings';
import type { DashboardMonthlyCashFlowPoint } from '../../types';

type MonthlyIncomeExpensesChartCardProps = {
  isLoading: boolean;
  points: DashboardMonthlyCashFlowPoint[];
  currencyCode: string;
  locale: string;
  formatAxisValue: (value: number) => string;
  className?: string;
};

export function MonthlyIncomeExpensesChartCard({
  isLoading,
  points,
  currencyCode,
  locale,
  formatAxisValue,
  className,
}: MonthlyIncomeExpensesChartCardProps) {
  return (
    <Card className={className} title="Monthly Income vs Expenses" subtitle="Current-year totals from your existing transactions">
      {isLoading ? (
        <div className="flex h-[350px] items-center justify-center rounded-2xl border border-dashed border-dark-elevated text-sm text-text-muted">
          Loading chart...
        </div>
      ) : points.length > 0 ? (
        <div className="h-[350px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={points} margin={{ top: 8, right: 0, left: 0, bottom: 0 }} barCategoryGap="28%">
              <CartesianGrid stroke="#E5E7EB" strokeDasharray="3 3" vertical={false} />
              <XAxis
                dataKey="label"
                tick={{ fill: '#6B7280', fontSize: 12 }}
                axisLine={false}
                tickLine={false}
                interval="preserveStartEnd"
              />
              <YAxis
                tick={{ fill: '#6B7280', fontSize: 12 }}
                axisLine={false}
                tickLine={false}
                width={52}
                tickFormatter={formatAxisValue}
              />
              <Tooltip
                formatter={(value, name) => [
                  formatCurrency(typeof value === 'number' ? value : 0, currencyCode, locale),
                  name === 'income' ? 'Income' : 'Expenses',
                ]}
                labelStyle={{ color: '#111827' }}
                contentStyle={{
                  background: 'rgba(17, 24, 39, 0.96)',
                  border: '1px solid rgba(148, 163, 184, 0.2)',
                  borderRadius: '16px',
                  color: '#F9FAFB',
                  boxShadow: '0 20px 40px rgba(15, 23, 42, 0.3)',
                }}
              />
              <Bar dataKey="income" fill="#4F46E5" radius={[4, 4, 0, 0]} barSize={10} />
              <Bar dataKey="expenses" fill="#E5E7EB" radius={[4, 4, 0, 0]} barSize={10} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      ) : (
        <div className="flex h-[350px] items-center justify-center rounded-2xl border border-dashed border-dark-elevated text-sm text-text-muted">
          No monthly cash flow data yet
        </div>
      )}
    </Card>
  );
}
