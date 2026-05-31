import { Button } from '../common/Button';

type WalletOverviewHeaderProps = {
  title: string;
  breadcrumbLabel: string;
  onRefresh: () => void;
  isRefreshing?: boolean;
};

export function WalletOverviewHeader({ title, breadcrumbLabel, onRefresh, isRefreshing = false }: WalletOverviewHeaderProps) {
  return (
    <div className="py-4">
      <div className="flex flex-col gap-1 md:flex-row md:items-center md:justify-between">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-bold tracking-tight text-text-primary">{title}</h1>
        </div>
        <nav className="flex items-center space-x-1 text-sm text-text-muted">
          <div className="flex items-center">
            <a className="hover:text-text-primary hover:underline" href="/">Home</a>
          </div>
          <div className="flex items-center">
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-4 w-4">
              <path d="m9 18 6-6-6-6" />
            </svg>
            <span className="pointer-events-none text-text-primary">{breadcrumbLabel}</span>
          </div>
          <Button variant="secondary" className="ml-3" onClick={onRefresh} disabled={isRefreshing}>
            {isRefreshing ? 'Refreshing...' : 'Refresh'}
          </Button>
        </nav>
      </div>
    </div>
  );
}