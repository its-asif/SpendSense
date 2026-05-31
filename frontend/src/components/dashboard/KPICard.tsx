import { Card } from '../common/Card';

type KPICardProps = {
  label: string;
  value: string;
  trend: string;
  direction: 'up' | 'down';
  tone?: 'blue' | 'green' | 'amber' | 'purple';
};

export function KPICard({ label, value, trend, direction, tone = 'blue' }: KPICardProps) {
  const trendClass = direction === 'up' ? 'text-accent-green' : 'text-accent-amber';
  const toneClass =
    tone === 'green'
      ? 'from-accent-green/20 to-transparent'
      : tone === 'amber'
        ? 'from-accent-amber/20 to-transparent'
        : tone === 'purple'
          ? 'from-accent-purple/20 to-transparent'
          : 'from-accent-blue/20 to-transparent';

  return (
    <Card className="overflow-hidden">
      <div className={`-mx-4 -mt-4 mb-4 h-2 bg-gradient-to-r ${toneClass}`} />
      <p className="text-xs font-semibold uppercase tracking-[0.14em] text-text-muted">{label}</p>
      <p className="mt-3 font-mono text-3xl font-semibold text-text-primary">{value}</p>
      <p className={`mt-2 text-sm font-semibold ${trendClass}`}>{direction === 'up' ? '↑' : '↓'} {trend}</p>
    </Card>
  );
}