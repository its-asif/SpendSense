import React, { useEffect, useState } from 'react';
import { Button } from '../common/Button';
import { Input } from '../common/Input';
import { WalletSelector } from '../expense/WalletSelector';
import { CategorySelector } from '../expense/CategorySelector';
import { CurrencySelector } from '../expense/CurrencySelector';
import { listCurrencies } from '../../api/currencies';
import { useUserSettings } from '../../hooks/useUserSettings';
import type { CurrencyOption, ExpenseCategory, Wallet } from '../../types';
import type { RecurringPayment, CreateRecurringRequest } from '../../api/recurring';

type RecurringPaymentFormProps = {
  categories: ExpenseCategory[];
  wallets: Wallet[];
  onSubmit: (data: CreateRecurringRequest) => Promise<void>;
  initialPayment?: RecurringPayment | null;
  onCancel?: () => void;
};

const WEEKDAYS = ['Saturday', 'Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday'];
const MONTHS = [
  { value: '1', label: 'January' },
  { value: '2', label: 'February' },
  { value: '3', label: 'March' },
  { value: '4', label: 'April' },
  { value: '5', label: 'May' },
  { value: '6', label: 'June' },
  { value: '7', label: 'July' },
  { value: '8', label: 'August' },
  { value: '9', label: 'September' },
  { value: '10', label: 'October' },
  { value: '11', label: 'November' },
  { value: '12', label: 'December' },
];

export function RecurringPaymentForm({
  categories,
  wallets,
  onSubmit,
  initialPayment,
  onCancel,
}: RecurringPaymentFormProps) {
  const settings = useUserSettings();
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Form states
  const [title, setTitle] = useState(initialPayment?.title ?? '');
  const [amount, setAmount] = useState(initialPayment?.amount ? String(initialPayment.amount) : '');
  const [walletId, setWalletId] = useState(initialPayment?.wallet_id ?? '');
  const [categoryId, setCategoryId] = useState(initialPayment?.category_id ?? '');
  const [currency, setCurrency] = useState(initialPayment?.currency ?? '');
  const [interval, setInterval] = useState(initialPayment?.interval ?? 'monthly');

  // Helpers to parse initial alert rule
  const getInitialAlertSelection = (rule?: string) => {
    if (!rule) return '1d';
    if (['start', '1d', '7d'].includes(rule)) {
      return rule;
    }
    if (rule.endsWith('d')) return 'custom_d';
    return 'start';
  };

  const getInitialCustomAlertValue = (rule?: string) => {
    if (!rule) return '';
    if (['start', '1d', '7d'].includes(rule)) {
      return '';
    }
    return rule.replace(/[d]/g, '');
  };

  const [alertSelection, setAlertSelection] = useState(() => getInitialAlertSelection(initialPayment?.alert_rule));
  const [customAlertValue, setCustomAlertValue] = useState(() => getInitialCustomAlertValue(initialPayment?.alert_rule));

  // Helpers to parse initial payment dates
  const getWeekdayName = (dateStr?: string) => {
    if (!dateStr) return 'Monday';
    const date = new Date(dateStr);
    const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
    return days[date.getDay()];
  };

  const getDayOfMonth = (dateStr?: string) => {
    if (!dateStr) return '1';
    const date = new Date(dateStr);
    return String(date.getDate());
  };

  const getMonthNum = (dateStr?: string) => {
    if (!dateStr) return '1';
    const date = new Date(dateStr);
    return String(date.getMonth() + 1);
  };

  // State options derived from interval
  const [startWeeklyDay, setStartWeeklyDay] = useState(() => getWeekdayName(initialPayment?.start_date));
  const [deadlineWeeklyDay, setDeadlineWeeklyDay] = useState(() => getWeekdayName(initialPayment?.deadline));

  const [startMonthlyDay, setStartMonthlyDay] = useState(() => getDayOfMonth(initialPayment?.start_date));
  const [deadlineMonthlyDay, setDeadlineMonthlyDay] = useState(() => getDayOfMonth(initialPayment?.deadline));

  const [startYearlyMonth, setStartYearlyMonth] = useState(() => getMonthNum(initialPayment?.start_date));
  const [startYearlyDay, setStartYearlyDay] = useState(() => getDayOfMonth(initialPayment?.start_date));
  const [deadlineYearlyMonth, setDeadlineYearlyMonth] = useState(() => getMonthNum(initialPayment?.deadline));
  const [deadlineYearlyDay, setDeadlineYearlyDay] = useState(() => getDayOfMonth(initialPayment?.deadline));

  // End of recurrence states
  const [endType, setEndType] = useState<'never' | 'date' | '1m' | '6m' | '1y' | '2y' | '5y'>(
    initialPayment?.end_date ? 'date' : 'never'
  );
  const [endDate, setEndDate] = useState(initialPayment?.end_date ?? '');

  // Currencies state
  const [currencyOptions, setCurrencyOptions] = useState<CurrencyOption[]>([]);
  const [isLoadingCurrencies, setIsLoadingCurrencies] = useState(false);

  useEffect(() => {
    const loadCurrencies = async () => {
      setIsLoadingCurrencies(true);
      try {
        const response = await listCurrencies();
        setCurrencyOptions(response);
        if (!currency) {
          const defaultOpt = response.find((c) => c.is_default);
          setCurrency(defaultOpt ? defaultOpt.code : settings.defaultCurrency);
        }
      } catch {
        setCurrencyOptions([{ code: 'USD', name: 'US Dollar', symbol: '$', symbol_native: '$', decimal_digits: 2, rounding: 0, name_plural: 'US dollars', is_default: true }]);
        if (!currency) setCurrency(settings.defaultCurrency);
      } finally {
        setIsLoadingCurrencies(false);
      }
    };
    void loadCurrencies();
  }, []);

  // Pre-fill categories/wallets if single option
  useEffect(() => {
    if (!walletId && wallets.length > 0) {
      setWalletId(wallets[0].id);
    }
    if (!categoryId && categories.length > 0) {
      const expCat = categories.find((c) => c.kind === 'EXPENSE');
      if (expCat) {
        setCategoryId(expCat.id);
      }
    }
  }, [wallets, categories]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!walletId || !categoryId || !amount || !title) {
      setError('Please fill in all required fields.');
      return;
    }

    setIsSaving(true);
    setError(null);

    // Compute standard date YYYY-MM-DD from user selections
    let finalStartDateStr = '';
    let finalDeadlineStr = '';
    const today = new Date();

    if (interval === 'daily') {
      finalStartDateStr = today.toISOString().slice(0, 10);
      finalDeadlineStr = today.toISOString().slice(0, 10);
    } else if (interval === 'weekly') {
      const getWeeklyDate = (dayName: string) => {
        const days = ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday'];
        const targetDay = days.indexOf(dayName.toLowerCase());
        const result = new Date();
        const currentDay = result.getDay();
        let diff = targetDay - currentDay;
        if (diff < 0) diff += 7;
        result.setDate(result.getDate() + diff);
        return result.toISOString().slice(0, 10);
      };
      finalStartDateStr = getWeeklyDate(startWeeklyDay);
      finalDeadlineStr = getWeeklyDate(deadlineWeeklyDay);
    } else if (interval === 'monthly') {
      const getMonthlyDate = (dayNum: number) => {
        const result = new Date();
        const maxDays = new Date(result.getFullYear(), result.getMonth() + 1, 0).getDate();
        result.setDate(Math.min(dayNum, maxDays));
        return result.toISOString().slice(0, 10);
      };
      finalStartDateStr = getMonthlyDate(Number(startMonthlyDay));
      finalDeadlineStr = getMonthlyDate(Number(deadlineMonthlyDay));
    } else if (interval === 'yearly') {
      const getYearlyDate = (monthNum: number, dayNum: number) => {
        const result = new Date();
        result.setMonth(monthNum - 1);
        const maxDays = new Date(result.getFullYear(), monthNum, 0).getDate();
        result.setDate(Math.min(dayNum, maxDays));
        return result.toISOString().slice(0, 10);
      };
      finalStartDateStr = getYearlyDate(Number(startYearlyMonth), Number(startYearlyDay));
      finalDeadlineStr = getYearlyDate(Number(deadlineYearlyMonth), Number(deadlineYearlyDay));
    }

    // Compile Alert Rule string
    let finalAlertRule = alertSelection;
    if (alertSelection === 'custom_d') {
      finalAlertRule = `${customAlertValue || '1'}d`;
    }

    // Calculate end date based on endType
    let finalEndDate: string | null = null;
    if (endType === 'date') {
      finalEndDate = endDate || null;
    } else if (endType !== 'never') {
      const start = new Date(finalStartDateStr);
      if (endType === '1m') start.setMonth(start.getMonth() + 1);
      else if (endType === '6m') start.setMonth(start.getMonth() + 6);
      else if (endType === '1y') start.setFullYear(start.getFullYear() + 1);
      else if (endType === '2y') start.setFullYear(start.getFullYear() + 2);
      else if (endType === '5y') start.setFullYear(start.getFullYear() + 5);
      finalEndDate = start.toISOString().slice(0, 10);
    }

    try {
      await onSubmit({
        wallet_id: walletId,
        category_id: categoryId,
        title: title.trim(),
        amount: Number(amount),
        currency,
        interval,
        start_date: finalStartDateStr,
        deadline: finalDeadlineStr,
        alert_rule: finalAlertRule,
        end_date: finalEndDate,
      });

      // Clear form inputs on success if creating new
      if (!initialPayment) {
        setTitle('');
        setAmount('');
        setInterval('monthly');
        setStartMonthlyDay('1');
        setDeadlineMonthlyDay('1');
        setAlertSelection('1d');
        setCustomAlertValue('');
        setEndType('never');
        setEndDate('');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save recurring payment.');
    } finally {
      setIsSaving(false);
    }
  };

  const daysOption = Array.from({ length: 31 }, (_, i) => String(i + 1));

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {error && (
        <div className="rounded-xl border border-accent-red/20 bg-accent-red/10 p-3 text-sm text-accent-red">
          {error}
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-2">
        <Input
          label="Title / Merchant"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="e.g. Rent, Internet Bill, Spotify"
          required
          disabled={isSaving}
        />
        <Input
          label="Amount"
          type="number"
          min="0.01"
          step="0.01"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          placeholder="1200.00"
          required
          disabled={isSaving}
        />
        <WalletSelector wallets={wallets} value={walletId} onChange={setWalletId} disabled={isSaving} />
        <CategorySelector categories={categories} value={categoryId} onChange={setCategoryId} disabled={isSaving} kind="EXPENSE" />
        <CurrencySelector currencies={currencyOptions} value={currency} onChange={setCurrency} disabled={isSaving} loading={isLoadingCurrencies} />
        
        <label className="block">
          <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Repeat Interval</span>
          <select
            value={interval}
            onChange={(e) => setInterval(e.target.value)}
            className="select"
            disabled={isSaving}
          >
            <option value="daily">Daily</option>
            <option value="weekly">Weekly</option>
            <option value="monthly">Monthly</option>
            <option value="yearly">Yearly</option>
          </select>
        </label>

        {/* Weekly start & deadline selections */}
        {interval === 'weekly' && (
          <>
            <label className="block">
              <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Payment Start Day</span>
              <select
                value={startWeeklyDay}
                onChange={(e) => setStartWeeklyDay(e.target.value)}
                className="select"
                disabled={isSaving}
              >
                {WEEKDAYS.map((d) => (
                  <option key={d} value={d}>{d}</option>
                ))}
              </select>
            </label>
            <label className="block">
              <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Payment Deadline Day</span>
              <select
                value={deadlineWeeklyDay}
                onChange={(e) => setDeadlineWeeklyDay(e.target.value)}
                className="select"
                disabled={isSaving}
              >
                {WEEKDAYS.map((d) => (
                  <option key={d} value={d}>{d}</option>
                ))}
              </select>
            </label>
          </>
        )}

        {/* Monthly start & deadline selections */}
        {interval === 'monthly' && (
          <>
            <label className="block">
              <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Payment Start Day (1-31)</span>
              <select
                value={startMonthlyDay}
                onChange={(e) => setStartMonthlyDay(e.target.value)}
                className="select"
                disabled={isSaving}
              >
                {daysOption.map((d) => (
                  <option key={d} value={d}>Day {d}</option>
                ))}
              </select>
            </label>
            <label className="block">
              <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Payment Deadline Day (1-31)</span>
              <select
                value={deadlineMonthlyDay}
                onChange={(e) => setDeadlineMonthlyDay(e.target.value)}
                className="select"
                disabled={isSaving}
              >
                {daysOption.map((d) => (
                  <option key={d} value={d}>Day {d}</option>
                ))}
              </select>
            </label>
          </>
        )}

        {/* Yearly start & deadline selections */}
        {interval === 'yearly' && (
          <>
            <div className="grid grid-cols-2 gap-2">
              <label className="block">
                <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Start Month</span>
                <select
                  value={startYearlyMonth}
                  onChange={(e) => setStartYearlyMonth(e.target.value)}
                  className="select"
                  disabled={isSaving}
                >
                  {MONTHS.map((m) => (
                    <option key={m.value} value={m.value}>{m.label}</option>
                  ))}
                </select>
              </label>
              <label className="block">
                <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Start Day</span>
                <select
                  value={startYearlyDay}
                  onChange={(e) => setStartYearlyDay(e.target.value)}
                  className="select"
                  disabled={isSaving}
                >
                  {daysOption.map((d) => (
                    <option key={d} value={d}>{d}</option>
                  ))}
                </select>
              </label>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <label className="block">
                <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Deadline Month</span>
                <select
                  value={deadlineYearlyMonth}
                  onChange={(e) => setDeadlineYearlyMonth(e.target.value)}
                  className="select"
                  disabled={isSaving}
                >
                  {MONTHS.map((m) => (
                    <option key={m.value} value={m.value}>{m.label}</option>
                  ))}
                </select>
              </label>
              <label className="block">
                <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Deadline Day</span>
                <select
                  value={deadlineYearlyDay}
                  onChange={(e) => setDeadlineYearlyDay(e.target.value)}
                  className="select"
                  disabled={isSaving}
                >
                  {daysOption.map((d) => (
                    <option key={d} value={d}>{d}</option>
                  ))}
                </select>
              </label>
            </div>
          </>
        )}

        <label className="block">
          <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Notification Alert</span>
          <select
            value={alertSelection}
            onChange={(e) => setAlertSelection(e.target.value)}
            className="select"
            disabled={isSaving}
          >
            <option value="start">When start date arrives</option>
            <option value="1d">1 Day before deadline</option>
            <option value="7d">7 Days before deadline</option>
            <option value="custom_d">Custom Days before deadline...</option>
          </select>
        </label>

        {alertSelection === 'custom_d' && (
          <Input
            label="Custom Days Offset"
            type="number"
            min="1"
            value={customAlertValue}
            onChange={(e) => setCustomAlertValue(e.target.value)}
            placeholder="e.g. 3"
            required
            disabled={isSaving}
          />
        )}

        <label className="block">
          <span className="mb-1.5 block text-xs font-semibold text-text-secondary">End of Recurrence</span>
          <select
            value={endType}
            onChange={(e) => setEndType(e.target.value as any)}
            className="select"
            disabled={isSaving}
          >
            <option value="never">Never Ends (Continuous)</option>
            <option value="date">Ends on Specific Date</option>
            <option value="1m">Ends after 1 Month</option>
            <option value="6m">Ends after 6 Months</option>
            <option value="1y">Ends after 1 Year</option>
            <option value="2y">Ends after 2 Years</option>
            <option value="5y">Ends after 5 Years</option>
          </select>
        </label>

        {endType === 'date' && (
          <Input
            label="Recurrence End Date"
            type="date"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            required
            disabled={isSaving}
          />
        )}
      </div>

      <div className="flex items-center justify-end gap-3 pt-4 border-t border-dark-elevated">
        {onCancel && (
          <Button variant="secondary" type="button" onClick={onCancel} disabled={isSaving}>
            Cancel
          </Button>
        )}
        <Button type="submit" disabled={isSaving}>
          {isSaving ? 'Saving...' : (initialPayment ? 'Save Changes' : 'Create Recurring Payment')}
        </Button>
      </div>
    </form>
  );
}
