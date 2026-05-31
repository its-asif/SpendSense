import type { Wallet } from '../../types';

type WalletSelectorProps = {
  wallets: Wallet[];
  value: string;
  onChange: (walletId: string) => void;
  disabled?: boolean;
};

export function WalletSelector({ wallets, value, onChange, disabled }: WalletSelectorProps) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Wallet</span>
      <select
        className="input"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={disabled}
      >
        <option value="">Select wallet</option>
        {wallets.map((wallet) => (
          <option key={wallet.id} value={wallet.id}>
            {wallet.name} · {wallet.currency}
          </option>
        ))}
      </select>
    </label>
  );
}