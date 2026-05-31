import { Bar, BarChart, CartesianGrid, Cell, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { Card } from '../common/Card';
import { formatCurrency } from '../../lib/userSettings';
import { useUserSettings } from '../../hooks/useUserSettings';

export type WalletBalancePoint = {
  id: string;
  name: string;
  balance: number;
  color: string;
};

type WalletBalanceChartCardProps = {
  isLoading: boolean;
  points: WalletBalancePoint[];
  currencyCode: string;
  formatAxisValue: (value: number) => string;
  className?: string;
};

export function WalletBalanceChartCard({ isLoading, points, currencyCode, formatAxisValue, className }: WalletBalanceChartCardProps) {
  const settings = useUserSettings();

  return (
    <Card className={className} title="Balance Overview" subtitle="Current wallet balances converted to your default currency">
      {isLoading ? (
        <div className="flex h-[320px] items-center justify-center rounded-2xl border border-dashed border-dark-elevated text-sm text-text-muted">
          Loading balance snapshot...
        </div>
      ) : points.length > 0 ? (
        <div className="h-[320px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={points} margin={{ top: 8, right: 0, left: 0, bottom: 0 }} barCategoryGap="22%">
              <CartesianGrid stroke="#E5E7EB" strokeDasharray="3 3" vertical={false} />
              <XAxis dataKey="name" tick={{ fill: '#6B7280', fontSize: 12 }} axisLine={false} tickLine={false} interval="preserveStartEnd" />
              <YAxis tick={{ fill: '#6B7280', fontSize: 12 }} axisLine={false} tickLine={false} width={52} tickFormatter={formatAxisValue} />
              <Tooltip
                formatter={(value) => [formatCurrency(typeof value === 'number' ? value : 0, currencyCode, settings.locale), 'Balance']}
                labelStyle={{ color: '#111827' }}
                contentStyle={{
                  background: 'rgba(255, 255, 255, 0.72)',
                  border: '1px solid rgba(148, 163, 184, 0.2)',
                  borderRadius: '16px',
                  color: '#111827',
                  boxShadow: '0 20px 40px rgba(15, 23, 42, 0.3)',
                }}
              />
              <Bar dataKey="balance" radius={[6, 6, 0, 0]}>
                {points.map((point) => (
                  <Cell key={point.id} fill={point.color} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      ) : (
        <div className="flex h-[320px] items-center justify-center rounded-2xl border border-dashed border-dark-elevated text-sm text-text-muted">
          No wallets to chart yet
        </div>
      )}
    </Card>
  );
}