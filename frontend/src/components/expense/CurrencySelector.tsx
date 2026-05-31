import type { CurrencyOption } from '../../types';

type CurrencySelectorProps = {
  currencies: CurrencyOption[];
  value: string;
  onChange: (currency: string) => void;
  disabled?: boolean;
  loading?: boolean;
};

export function CurrencySelector({ currencies, value, onChange, disabled, loading }: CurrencySelectorProps) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-xs font-semibold text-text-secondary">Currency</span>
      <select
        className="input"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={disabled || loading || currencies.length === 0}
      >
        {loading ? (
          <option value="">Loading currencies...</option>
        ) : currencies.length === 0 ? (
          <option value="">No currencies available</option>
        ) : null}
        {currencies.map((currency) => (
          <option key={currency.code} value={currency.code}>
            {currency.is_default ? `${currency.code} - ${currency.name} (default)` : `${currency.code} - ${currency.name}`}
          </option>
        ))}
      </select>
    </label>
  );
}
