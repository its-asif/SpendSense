import { Button } from '../common/Button';
import { Card } from '../common/Card';
import { formatCurrency } from '../../lib/userSettings';
import { useUserSettings } from '../../hooks/useUserSettings';
import { WalletTypeIcon } from './WalletTypeIcon';
import type { Wallet } from '../../types';

type WalletOverviewSummaryProps = {
  selectedWallet: Wallet | null;
  selectedWalletBalance: number;
  totalBalance: number;
  personalFunds: number;
  creditExposure: number;
  currencyCode: string;
  onEditSelectedWallet: () => void;
  onTransferSelectedWallet: () => void;
  onDeleteSelectedWallet: () => void;
};

function maskAccountNumber(value?: string | null) {
  if (!value) {
    return 'No account number';
  }

  const digits = value.replace(/\s+/g, '');
  if (digits.length <= 4) {
    return digits;
  }

  return `**** **** **** ${digits.slice(-4)}`;
}

export function WalletOverviewSummary({
  selectedWallet,
  selectedWalletBalance,
  totalBalance,
  personalFunds,
  creditExposure,
  currencyCode,
  onEditSelectedWallet,
  onTransferSelectedWallet,
  onDeleteSelectedWallet,
}: WalletOverviewSummaryProps) {
  const settings = useUserSettings();

  const selectedWalletDelta = selectedWallet
    ? selectedWallet.current_balance - selectedWallet.opening_balance
    : 0;
  const selectedWalletDeltaLabel = selectedWallet
    ? `${selectedWalletDelta >= 0 ? '+' : '-'}${formatCurrency(Math.abs(selectedWalletDelta), selectedWallet.currency, settings.locale)} since opening`
    : 'No wallet selected';

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <Card className="overflow-hidden">
        <div className="p-4 sm:p-6">
          <div className="text-sm text-text-muted mb-1">Total Balance</div>
          <div className="mb-4 text-2xl sm:text-3xl font-bold text-text-primary">
            {formatCurrency(totalBalance, currencyCode, settings.locale)}
          </div>
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div className="text-sm text-text-muted">Personal Funds</div>
              <div className="font-medium text-text-primary">{formatCurrency(personalFunds, currencyCode, settings.locale)}</div>
            </div>
            <div className="flex items-center justify-between">
              <div className="text-sm text-text-muted">Credit Exposure</div>
              <div className="font-medium text-text-primary">{formatCurrency(creditExposure, currencyCode, settings.locale)}</div>
            </div>
          </div>
        </div>
      </Card>

      <Card className="overflow-hidden bg-gradient-to-br from-accent-blue to-accent-purple text-white">
        <div className="relative p-4 sm:p-6">
          {selectedWallet ? (
            <>
              <div className="absolute right-4 top-4">
                <span className="font-bold opacity-80">{selectedWallet.wallet_type}</span>
              </div>
              <div className="mb-6 mt-8 text-lg tracking-widest font-mono sm:text-2xl">
                {maskAccountNumber(selectedWallet.account_number)}
              </div>
              <div className="flex items-center justify-between gap-4">
                <div>
                  <div className="font-medium">{selectedWallet.account_name ?? selectedWallet.name}</div>
                  <div className="text-sm opacity-80">{selectedWallet.provider ?? selectedWallet.currency}</div>
                </div>
                <div className="rounded-full bg-white/15 p-3">
                  <WalletTypeIcon walletType={selectedWallet.wallet_type} className="h-5 w-5 text-white" />
                </div>
              </div>
              <div className="mt-5 border-t border-white/15 pt-4">
                <div className="text-sm opacity-80">Selected Balance</div>
                <div className="mt-1 text-xl font-bold sm:text-2xl">
                  {formatCurrency(selectedWalletBalance, selectedWallet.currency, settings.locale)}
                </div>
                <div className="mt-2 text-xs opacity-80">{selectedWalletDeltaLabel}</div>
              </div>
              <div className="mt-5 flex flex-wrap gap-2">
                <Button variant="secondary" type="button" onClick={onEditSelectedWallet}>Edit</Button>
                <Button variant="secondary" type="button" onClick={onTransferSelectedWallet}>Transfer</Button>
                <Button variant="secondary" type="button" onClick={onDeleteSelectedWallet}>Delete</Button>
              </div>
            </>
          ) : (
            <div className="py-8 text-sm opacity-80">No wallet selected.</div>
          )}
        </div>
      </Card>
    </div>
  );
}