import type { Wallet } from '../../types';
import { formatCurrency } from '../../lib/userSettings';
import { useUserSettings } from '../../hooks/useUserSettings';
import { WalletTypeIcon } from './WalletTypeIcon';

type WalletRailProps = {
  wallets: Wallet[];
  selectedWalletId: string | null;
  onSelectWallet: (walletId: string) => void;
  onAddWallet: () => void;
};

export function WalletRail({ wallets, selectedWalletId, onSelectWallet, onAddWallet }: WalletRailProps) {
  const settings = useUserSettings();

  return (
    <aside className="lg:space-y-3">
      <div className="flex gap-3 overflow-x-auto pb-2 lg:flex-col lg:overflow-visible lg:pb-0">
        {wallets.map((wallet) => {
          const active = wallet.id === selectedWalletId;

          return (
            <button
              key={wallet.id}
              type="button"
              onClick={() => onSelectWallet(wallet.id)}
              className={[
                'flex-shrink-0 w-[200px] sm:w-[220px] lg:w-full overflow-hidden rounded-3xl border text-left shadow-sm transition-all duration-200',
                active
                  ? 'border-white/20 bg-gradient-to-br from-accent-blue via-accent-purple to-accent-blue text-white shadow-lg shadow-accent-blue/25 ring-1 ring-white/20'
                  : 'border-dark-elevated bg-card text-text-primary hover:border-accent-blue/30 hover:bg-dark-bg hover:shadow-md',
              ].join(' ')}
            >
              <div className="p-4">
                <div className="flex items-center gap-3">
                  <div className={[
                    'rounded-full p-3 transition-colors',
                    active ? 'bg-white/15' : 'bg-dark-bg',
                  ].join(' ')}>
                    <WalletTypeIcon
                      walletType={wallet.wallet_type}
                      className={['h-5 w-5', active ? 'text-white' : 'text-accent-blue'].join(' ')}
                    />
                  </div>
                  <div className="min-w-0">
                    <div className={['truncate text-sm font-medium', active ? 'text-white' : 'text-text-primary'].join(' ')}>{wallet.name}</div>
                    <div className={['font-mono text-base font-bold', active ? 'text-white' : 'text-text-primary'].join(' ')}>
                      {formatCurrency(wallet.current_balance, wallet.currency, settings.locale)}
                    </div>
                  </div>
                </div>
              </div>
            </button>
          );
        })}

        <button
          type="button"
          onClick={onAddWallet}
          className="flex-shrink-0 w-[200px] sm:w-[220px] lg:w-full rounded-2xl border border-dashed border-dark-elevated bg-card p-4 text-center transition-colors hover:border-accent-blue hover:bg-dark-bg flex items-center justify-center gap-2"
        >
          <span className="text-sm font-medium text-text-primary">Add new wallet</span>
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-4 w-4">
            <path d="M5 12h14" />
            <path d="M12 5v14" />
          </svg>
        </button>
      </div>
    </aside>
  );
}