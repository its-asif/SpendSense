import { useEffect, useState } from 'react';
import { Button } from '../common/Button';
import { Input } from '../common/Input';
import { CurrencySelector } from '../expense/CurrencySelector';
import { listCurrencies } from '../../api/currencies';
import { useUserSettings } from '../../hooks/useUserSettings';
import type { Wallet, UpdateWalletRequest, CurrencyOption } from '../../types';

type WalletFormProps = {
  initial?: Wallet | null;
  onSubmit: (data: UpdateWalletRequest) => Promise<void> | void;
  onCancel?: () => void;
};

export function WalletForm({ initial, onSubmit, onCancel }: WalletFormProps) {
  const settings = useUserSettings();
  const [name, setName] = useState(initial?.name ?? '');
  const [walletType, setWalletType] = useState(initial?.wallet_type ?? 'BANK');
  const [provider, setProvider] = useState(initial?.provider ?? '');
  const [accountNumber, setAccountNumber] = useState(initial?.account_number ?? '');
  const [currency, setCurrency] = useState(initial?.currency ?? settings.defaultCurrency);
  const [currencies, setCurrencies] = useState<CurrencyOption[]>([]);
  const [loadingCurrencies, setLoadingCurrencies] = useState(false);
  const [openingBalance, setOpeningBalance] = useState<number>(initial?.opening_balance ?? initial?.current_balance ?? 0);

  const submit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    const payload: UpdateWalletRequest = {
      name,
      wallet_type: walletType,
      provider: provider || undefined,
      account_number: accountNumber || undefined,
      currency,
      opening_balance: Number(openingBalance ?? 0),
    };

    await onSubmit(payload);
  };

  useEffect(() => {
    let cancelled = false;
    setLoadingCurrencies(true);
    void listCurrencies()
      .then((list) => {
        if (!cancelled) setCurrencies(list);
      })
      .catch(() => {
        if (!cancelled) setCurrencies([]);
      })
      .finally(() => {
        if (!cancelled) setLoadingCurrencies(false);
      });

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <form onSubmit={submit} className="space-y-3">
      <Input label="Name" value={name} onChange={(e) => setName(e.target.value)} required />
      <label className="block text-sm text-text-muted">Wallet type</label>
      <select className="input" value={walletType} onChange={(e) => setWalletType(e.target.value)}>
        <option value="CASH">Cash</option>
        <option value="MOBILE_WALLET">Mobile wallet</option>
        <option value="BANK">Bank</option>
        <option value="CARD">Card</option>
      </select>

      <Input label="Provider" value={provider ?? ''} onChange={(e) => setProvider(e.target.value)} />
      <Input label="Account number" value={accountNumber ?? ''} onChange={(e) => setAccountNumber(e.target.value)} />
      <CurrencySelector currencies={currencies} value={currency} onChange={setCurrency} loading={loadingCurrencies} />
      <Input label="Opening balance" type="number" value={String(openingBalance)} onChange={(e) => setOpeningBalance(Number(e.target.value))} />

      <div className="flex items-center gap-3">
        <Button type="submit">Save</Button>
        <Button variant="secondary" type="button" onClick={onCancel}>Cancel</Button>
      </div>
    </form>
  );
}

export default WalletForm;
