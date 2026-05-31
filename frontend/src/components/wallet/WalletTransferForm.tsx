import { useEffect, useState } from 'react';
import { Button } from '../common/Button';
import { Input } from '../common/Input';
import { convertCurrency } from '../../api/currencies';
import { useUserSettings } from '../../hooks/useUserSettings';
import { formatCurrency } from '../../lib/userSettings';
import type { Wallet, CurrencyOption } from '../../types';

type TransferFormProps = {
  wallets: Wallet[];
  currencies: CurrencyOption[];
  loadingCurrencies?: boolean;
  from?: Wallet | null;
  onSubmit: (data: { from_wallet_id: string; to_wallet_id: string; amount: number; currency: string; fee_amount: number; notes?: string }) => Promise<void> | void;
  onCancel?: () => void;
};

export function WalletTransferForm({ wallets, currencies, loadingCurrencies, from, onSubmit, onCancel }: TransferFormProps) {
  const settings = useUserSettings();
  const [fromId, setFromId] = useState(from?.id ?? (wallets[0]?.id ?? ''));
  const [toId, setToId] = useState(wallets.find((w) => w.id !== from?.id)?.id ?? wallets[0]?.id ?? '');
  const [amount, setAmount] = useState<number>(0);
  const [feeAmount, setFeeAmount] = useState<number>(0);
  // currency is derived from the selected "from" wallet
  const [currency, setCurrency] = useState<string>(from?.currency ?? wallets.find((w) => w.id === from?.id)?.currency ?? wallets[0]?.currency ?? settings.defaultCurrency);
  const [converted, setConverted] = useState<number | null>(null);
  const [notes, setNotes] = useState('');

  const submit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!fromId || !toId) return;
    await onSubmit({
      from_wallet_id: fromId,
      to_wallet_id: toId,
      amount: Number(amount),
      currency,
      fee_amount: Number(feeAmount),
      notes: notes || undefined,
    });
  };

  useEffect(() => {
    // keep currency in sync with selected from wallet
    const fromWallet = wallets.find((w) => w.id === fromId) ?? from;
    const newCurrency = fromWallet?.currency ?? wallets[0]?.currency ?? settings.defaultCurrency;
    setCurrency(newCurrency);
  }, [fromId, from, wallets, settings.defaultCurrency]);

  useEffect(() => {
    let cancelled = false;
    const toWallet = wallets.find((w) => w.id === toId);
    if (!toWallet) {
      setConverted(null);
      return;
    }

    const fromCurrency = currency;
    const toCurrency = toWallet.currency;
    if (!amount || amount === 0 || fromCurrency === toCurrency) {
      setConverted(null);
      return;
    }

    void convertCurrency(Number(amount), fromCurrency, toCurrency)
      .then((res) => {
        if (!cancelled) setConverted(res.converted_amount);
      })
      .catch(() => {
        if (!cancelled) setConverted(null);
      });

    return () => {
      cancelled = true;
    };
  }, [amount, toId, currency, wallets]);

  return (
    <form onSubmit={submit} className="space-y-3">
      <label className="block">
        <span className="mb-1.5 block text-xs font-semibold text-text-secondary">From</span>
        <select className="input" value={fromId} onChange={(e) => setFromId(e.target.value)}>
          {wallets.map((w) => (
            <option key={w.id} value={w.id}>{w.name} · {w.currency}</option>
          ))}
        </select>
      </label>

      <label className="block">
        <span className="mb-1.5 block text-xs font-semibold text-text-secondary">To</span>
        <select className="input" value={toId} onChange={(e) => setToId(e.target.value)}>
          {wallets.filter((w) => w.id !== fromId).map((w) => (
            <option key={w.id} value={w.id}>{w.name} · {w.currency}</option>
          ))}
        </select>
      </label>

      <Input label={`Amount (${currency})`} type="number" value={String(amount)} onChange={(e) => setAmount(Number(e.target.value))} />

      <Input
        label={`Service charge (${currency})`}
        type="number"
        min="0"
        step="0.01"
        value={String(feeAmount)}
        onChange={(e) => setFeeAmount(Number(e.target.value))}
        placeholder="0.00"
      />

      <p className="text-xs text-text-muted">
        Service charge is deducted from the source wallet only and is not added to the recipient transfer amount.
      </p>

      <div className="space-y-1 text-sm text-text-secondary">
        {converted !== null && (
          <p>
            Recipient receives ≈ {formatCurrency(converted, wallets.find((w) => w.id === toId)?.currency ?? settings.defaultCurrency, settings.locale)} ({wallets.find((w) => w.id === toId)?.currency})
          </p>
        )}
        {(amount || feeAmount) > 0 && (
          <p>
            Source wallet is debited ≈ {formatCurrency(Number(amount) + Number(feeAmount), currency, settings.locale)} including fee.
          </p>
        )}
      </div>

      <Input label="Notes" value={notes} onChange={(e) => setNotes(e.target.value)} />

      <div className="flex items-center gap-3">
        <Button type="submit">Transfer</Button>
        <Button variant="secondary" type="button" onClick={onCancel}>Cancel</Button>
      </div>
    </form>
  );
}

export default WalletTransferForm;
