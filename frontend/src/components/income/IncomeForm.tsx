import { useEffect, useState } from 'react';
import { Button } from '../common/Button';
import { Input } from '../common/Input';
import { useUserSettings } from '../../hooks/useUserSettings';
import type { CreateIncomeRequest, CurrencyOption, ExpenseCategory, Income, Wallet } from '../../types';
import { CategorySelector } from '../expense/CategorySelector';
import { CurrencySelector } from '../expense/CurrencySelector';
import { WalletSelector } from '../expense/WalletSelector';
import { listCurrencies } from '../../api/currencies';

type IncomeFormProps = {
  categories: ExpenseCategory[];
  wallets: Wallet[];
  onSubmit: (data: CreateIncomeRequest) => Promise<void>;
  initialIncome?: Income | null;
  onCancel?: () => void;
};

function getInitialDate(income?: Income | null) {
  if (income?.income_date) {
    return income.income_date;
  }

  return new Date().toISOString().slice(0, 10);
}

export function IncomeForm({ categories, wallets, onSubmit, initialIncome, onCancel }: IncomeFormProps) {
  const settings = useUserSettings();
  const [walletId, setWalletId] = useState(initialIncome?.wallet_id ?? '');
  const [categoryId, setCategoryId] = useState(initialIncome?.category_id ?? '');
  const [amount, setAmount] = useState(initialIncome ? initialIncome.amount.toFixed(2) : '');
  const [currency, setCurrency] = useState(initialIncome?.currency ?? settings.defaultCurrency);
  const [currencyOptions, setCurrencyOptions] = useState<CurrencyOption[]>([]);
  const [isLoadingCurrencies, setIsLoadingCurrencies] = useState(false);
  const [date, setDate] = useState(() => getInitialDate(initialIncome));
  const [sourceName, setSourceName] = useState(initialIncome?.source_name ?? '');
  const [notes, setNotes] = useState(initialIncome?.notes ?? '');
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function loadCurrencies() {
      setIsLoadingCurrencies(true);

      try {
        const currencies = await listCurrencies(initialIncome?.currency ?? settings.defaultCurrency);
        if (cancelled) {
          return;
        }

        setCurrencyOptions(currencies);
        setCurrency((currentCurrency) => {
          if (currentCurrency && currencies.some((item) => item.code === currentCurrency)) {
            return currentCurrency;
          }

          return currencies[0]?.code ?? initialIncome?.currency ?? settings.defaultCurrency;
        });
      } catch {
        if (!cancelled) {
          setCurrencyOptions([]);
          setCurrency((currentCurrency) => currentCurrency || initialIncome?.currency || settings.defaultCurrency);
        }
      } finally {
        if (!cancelled) {
          setIsLoadingCurrencies(false);
        }
      }
    }

    loadCurrencies();

    return () => {
      cancelled = true;
    };
  }, [initialIncome?.currency, settings.defaultCurrency]);

  return (
    <form
      className="space-y-4"
      onSubmit={async (event) => {
        event.preventDefault();
        setIsSaving(true);
        setError(null);

        try {
          await onSubmit({
            wallet_id: walletId,
            amount: Number(amount),
            currency,
            category_id: categoryId || undefined,
            source_name: sourceName,
            income_date: date,
            notes: notes || undefined,
          });

          if (!initialIncome) {
            setAmount('');
            setSourceName('');
            setNotes('');
            setWalletId('');
            setCategoryId('');
            setCurrency(currencyOptions[0]?.code ?? settings.defaultCurrency);
            setDate(new Date().toISOString().slice(0, 10));
          }
        } catch {
          setError('Failed to save income. Check the backend and required fields.');
        } finally {
          setIsSaving(false);
        }
      }}
    >
      <div className="grid gap-4 md:grid-cols-2">
        <WalletSelector wallets={wallets} value={walletId} onChange={setWalletId} disabled={isSaving} />
        <CategorySelector categories={categories} value={categoryId} onChange={setCategoryId} disabled={isSaving} />
        <Input label="Amount" type="number" min="0" step="0.01" value={amount} onChange={(event) => setAmount(event.target.value)} placeholder="5000.00" required />
        <CurrencySelector currencies={currencyOptions} value={currency} onChange={setCurrency} disabled={isSaving} loading={isLoadingCurrencies} />
        <Input label="Date" type="date" value={date} onChange={(event) => setDate(event.target.value)} required />
        <Input label="Source" value={sourceName} onChange={(event) => setSourceName(event.target.value)} placeholder="Salary, freelance, refund" required />
      </div>

      <label className="block">
        <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Notes</span>
        <textarea
          className="input min-h-24 resize-none py-3"
          value={notes}
          onChange={(event) => setNotes(event.target.value)}
          placeholder="Optional details"
        />
      </label>

      {error && <p className="text-sm text-accent-red">{error}</p>}

      <div className="flex flex-col gap-3 sm:flex-row">
        <Button type="submit" className="w-full" disabled={isSaving || !walletId || !sourceName}>
          {isSaving ? 'Saving...' : initialIncome ? 'Update income' : 'Add income'}
        </Button>
        {initialIncome && onCancel && (
          <Button type="button" variant="secondary" className="w-full" onClick={onCancel} disabled={isSaving}>
            Cancel edit
          </Button>
        )}
      </div>
    </form>
  );
}