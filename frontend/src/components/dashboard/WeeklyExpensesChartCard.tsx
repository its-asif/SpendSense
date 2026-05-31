import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { Card } from '../common/Card';

type WeeklyExpensesPoint = {
  label: string;
  cash: number;
  bank: number;
  digital: number;
};

type WeeklyExpensesChartCardProps = {
  isLoading: boolean;
  points: WeeklyExpensesPoint[];
  formatAxisValue: (value: number) => string;
  className?: string;
};

export function WeeklyExpensesChartCard({ isLoading, points, formatAxisValue, className }: WeeklyExpensesChartCardProps) {
  return (
    <Card className={className} title="Monthly Expenses">
      {isLoading ? (
        <div className="flex h-[300px] items-center justify-center rounded-2xl border border-dashed border-dark-elevated text-sm text-text-muted">
          Loading monthly expenses...
        </div>
      ) : points.length > 0 ? (
        <div className="h-[300px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={points} margin={{ top: 5, right: 0, left: 0, bottom: 0 }} barCategoryGap="18%">
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
                formatter={(value, name) => [typeof value === 'number' ? value.toFixed(2) : '0.00', name]}
                labelStyle={{ color: '#111827' }}
                contentStyle={{
                  background: 'rgba(17, 24, 39, 0.96)',
                  border: '1px solid rgba(148, 163, 184, 0.2)',
                  borderRadius: '16px',
                  color: '#F9FAFB',
                  boxShadow: '0 20px 40px rgba(15, 23, 42, 0.3)',
                }}
              />
              <Bar dataKey="cash" stackId="expenses" fill="#818CF8" radius={[4, 4, 0, 0]} />
              <Bar dataKey="bank" stackId="expenses" fill="#6366F1" radius={[4, 4, 0, 0]} />
              <Bar dataKey="digital" stackId="expenses" fill="#4F46E5" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      ) : (
        <div className="flex h-[300px] items-center justify-center rounded-2xl border border-dashed border-dark-elevated text-sm text-text-muted">
          No monthly expenses yet
        </div>
      )}
    </Card>
  );
}
