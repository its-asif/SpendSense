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
  onSubmit: (data: CreateExpenseRequest, file: File | null) => Promise<void>;
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

  // Drag and drop receipt uploader state
  const [receiptFile, setReceiptFile] = useState<File | null>(null);
  const [isDragActive, setIsDragActive] = useState(false);


  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "dragenter" || e.type === "dragover") {
      setIsDragActive(true);
    } else if (e.type === "dragleave") {
      setIsDragActive(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragActive(false);
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      setReceiptFile(e.dataTransfer.files[0]);
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setReceiptFile(e.target.files[0]);
    }
  };

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
            is_recurring: false,
            recurring_rule: undefined,
          }, receiptFile);

          if (!initialExpense) {
            setAmount('');
            setMerchant('');
            setNotes('');
            setWalletId('');
            setCategoryId('');
            setReceiptFile(null);
            setCurrency(currencyOptions[0]?.code ?? settings.defaultCurrency);
            setDate(new Date().toISOString().slice(0, 10));
          }
        } catch (err) {
          setError(err instanceof Error ? err.message : 'Failed to save expense. Check the backend and required fields.');
        } finally {
          setIsSaving(false);
        }
      }}
    >
      <div className="grid gap-4 md:grid-cols-2">
        <WalletSelector wallets={wallets} value={walletId} onChange={setWalletId} disabled={isSaving} />
        <CategorySelector categories={categories} value={categoryId} onChange={setCategoryId} disabled={isSaving} kind="EXPENSE" />
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

      {/* Drag & Drop Receipt Attachment */}
      <div className="block">
        <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Receipt Attachment (Optional)</span>
        <div
          onDragEnter={handleDrag}
          onDragOver={handleDrag}
          onDragLeave={handleDrag}
          onDrop={handleDrop}
          className={`relative border border-dashed rounded-xl p-4 flex flex-col items-center justify-center transition-all cursor-pointer ${
            isDragActive ? 'border-accent-blue bg-accent-blue/10' : 'border-dark-elevated hover:border-text-muted bg-dark-bg/30'
          }`}
          onClick={() => document.getElementById('receipt-input')?.click()}
        >
          <input
            id="receipt-input"
            type="file"
            accept="image/*,application/pdf"
            onChange={handleFileChange}
            className="hidden"
          />
          <div className="text-center space-y-1 py-1">
            <span className="text-xl">📄</span>
            {receiptFile ? (
              <div>
                <p className="text-sm font-semibold text-text-primary">{receiptFile.name}</p>
                <p className="text-xs text-text-muted">{(receiptFile.size / 1024).toFixed(1)} KB</p>
              </div>
            ) : initialExpense?.receipt_url ? (
              <div>
                <p className="text-sm font-semibold text-accent-green">✓ Receipt already attached</p>
                <p className="text-xs text-text-muted">Click or drag a new file to overwrite</p>
              </div>
            ) : (
              <div>
                <p className="text-sm font-medium text-text-primary">Drag & drop receipt here, or <span className="text-accent-blue font-semibold">browse</span></p>
                <p className="text-xs text-text-muted">Supports images and PDF (Max 5MB)</p>
              </div>
            )}
          </div>
          {receiptFile && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                setReceiptFile(null);
              }}
              className="absolute top-2 right-2 text-xs bg-dark-elevated text-text-primary h-5 w-5 rounded-full flex items-center justify-center hover:bg-destructive hover:text-white"
            >
              ✕
            </button>
          )}
        </div>
      </div>

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