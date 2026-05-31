import { useEffect, useState } from 'react';
import { Button } from '../common/Button';
import { Input } from '../common/Input';
import { useUserSettings } from '../../hooks/useUserSettings';
import type { CreateExpenseRequest, CurrencyOption, Expense, ExpenseCategory, Wallet } from '../../types';
import { CategorySelector } from './CategorySelector';
import { CurrencySelector } from './CurrencySelector';
import { WalletSelector } from './WalletSelector';
import { listCurrencies } from '../../api/currencies';

type ExpenseFormProps = {
  categories: ExpenseCategory[];
  wallets: Wallet[];
  onSubmit: (data: CreateExpenseRequest) => Promise<void>;
  initialExpense?: Expense | null;
  onCancel?: () => void;
};

function getInitialDate(expense?: Expense | null) {
  if (expense?.date) {
    return expense.date;
  }

  return new Date().toISOString().slice(0, 10);
}

export function ExpenseForm({ categories, wallets, onSubmit, initialExpense, onCancel }: ExpenseFormProps) {
  const settings = useUserSettings();
  const [walletId, setWalletId] = useState(initialExpense?.wallet_id ?? '');
  const [categoryId, setCategoryId] = useState(initialExpense?.category_id ?? '');
  const [amount, setAmount] = useState(initialExpense ? initialExpense.amount.toFixed(2) : '');
  const [currency, setCurrency] = useState(initialExpense?.currency ?? settings.defaultCurrency);
  const [currencyOptions, setCurrencyOptions] = useState<CurrencyOption[]>([]);
  const [isLoadingCurrencies, setIsLoadingCurrencies] = useState(false);
  const [date, setDate] = useState(() => getInitialDate(initialExpense));
  const [merchant, setMerchant] = useState(initialExpense?.merchant ?? '');
  const [notes, setNotes] = useState(initialExpense?.notes ?? '');
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function loadCurrencies() {
      setIsLoadingCurrencies(true);

      try {
        const currencies = await listCurrencies(initialExpense?.currency ?? settings.defaultCurrency);
        if (cancelled) {
          return;
        }

        setCurrencyOptions(currencies);
        setCurrency((currentCurrency) => {
          if (currentCurrency && currencies.some((item) => item.code === currentCurrency)) {
            return currentCurrency;
          }

          return currencies[0]?.code ?? initialExpense?.currency ?? settings.defaultCurrency;
        });
      } catch {
        if (!cancelled) {
          setCurrencyOptions([]);
          setCurrency((currentCurrency) => currentCurrency || initialExpense?.currency || settings.defaultCurrency);
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
  }, [initialExpense?.currency, settings.defaultCurrency]);

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
            category_id: categoryId,
            merchant: merchant || undefined,
            date,
            notes: notes || undefined,
            fx_rate_to_base: 1,
            is_recurring: initialExpense?.is_recurring ?? false,
            recurring_rule: initialExpense?.recurring_rule ?? undefined,
          });

          if (!initialExpense) {
            setAmount('');
            setMerchant('');
            setNotes('');
            setWalletId('');
            setCategoryId('');
            setCurrency(currencyOptions[0]?.code ?? settings.defaultCurrency);
            setDate(new Date().toISOString().slice(0, 10));
          }
        } catch {
          setError('Failed to save expense. Check the backend and required fields.');
        } finally {
          setIsSaving(false);
        }
      }}
    >
      <div className="grid gap-4 md:grid-cols-2">
        <WalletSelector wallets={wallets} value={walletId} onChange={setWalletId} disabled={isSaving} />
        <CategorySelector categories={categories} value={categoryId} onChange={setCategoryId} disabled={isSaving} />
        <Input label="Amount" type="number" min="0" step="0.01" value={amount} onChange={(event) => setAmount(event.target.value)} placeholder="12.50" required />
        <CurrencySelector currencies={currencyOptions} value={currency} onChange={setCurrency} disabled={isSaving} loading={isLoadingCurrencies} />
        <Input label="Date" type="date" value={date} onChange={(event) => setDate(event.target.value)} required />
        <Input label="Merchant" value={merchant} onChange={(event) => setMerchant(event.target.value)} placeholder="Coffee shop" />
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
        <Button type="submit" className="w-full" disabled={isSaving || !walletId || !categoryId}>
          {isSaving ? 'Saving...' : initialExpense ? 'Update expense' : 'Add expense'}
        </Button>
        {initialExpense && onCancel && (
          <Button type="button" variant="secondary" className="w-full" onClick={onCancel} disabled={isSaving}>
            Cancel edit
          </Button>
        )}
      </div>
    </form>
  );
}