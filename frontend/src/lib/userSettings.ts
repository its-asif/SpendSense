import { readSession } from './storage';

export type UserSettings = {
  defaultCurrency: string;
  locale: string;
  timezone: string;
};

const SETTINGS_KEY = 'spendsense-user-settings';
const SETTINGS_EVENT = 'spendsense-settings-changed';

function getAccountBaseCurrency() {
  return readSession()?.user.baseCurrency?.trim().toUpperCase() || '';
}

export function getDefaultUserSettings(): UserSettings {
  const accountCurrency = getAccountBaseCurrency();
  return {
    defaultCurrency: accountCurrency || 'USD',
    locale: 'en-US',
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
  };
}

export function readUserSettings(): UserSettings {
  if (typeof window === 'undefined') {
    return getDefaultUserSettings();
  }

  try {
    const raw = window.localStorage.getItem(SETTINGS_KEY);
    if (!raw) {
      return getDefaultUserSettings();
    }

    const parsed = JSON.parse(raw) as Partial<UserSettings>;
    const accountCurrency = getAccountBaseCurrency();
    return {
      defaultCurrency: parsed.defaultCurrency?.trim().toUpperCase() || accountCurrency || 'USD',
      locale: parsed.locale || 'en-US',
      timezone: parsed.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
    };
  } catch {
    return getDefaultUserSettings();
  }
}

export function saveUserSettings(settings: UserSettings) {
  if (typeof window === 'undefined') {
    return;
  }

  window.localStorage.setItem(
    SETTINGS_KEY,
    JSON.stringify({
      ...settings,
      defaultCurrency: settings.defaultCurrency.trim().toUpperCase(),
    }),
  );
  window.dispatchEvent(new Event(SETTINGS_EVENT));
}

export function subscribeToUserSettings(onChange: () => void) {
  if (typeof window === 'undefined') {
    return () => undefined;
  }

  const handler = () => onChange();
  window.addEventListener('storage', handler);
  window.addEventListener(SETTINGS_EVENT, handler);

  return () => {
    window.removeEventListener('storage', handler);
    window.removeEventListener(SETTINGS_EVENT, handler);
  };
}

export function formatCurrency(amount: number, currencyCode?: string, locale?: string) {
  const resolvedCurrency = currencyCode || readUserSettings().defaultCurrency;
  const resolvedLocale = locale || readUserSettings().locale;

  try {
    return new Intl.NumberFormat(resolvedLocale, { style: 'currency', currency: resolvedCurrency }).format(amount);
  } catch {
    return `${amount.toFixed(2)} ${resolvedCurrency}`;
  }
}

export function formatNumber(value: number, locale?: string, options?: Intl.NumberFormatOptions) {
  const resolvedLocale = locale || readUserSettings().locale;
  try {
    return new Intl.NumberFormat(resolvedLocale, options).format(value);
  } catch {
    return String(value);
  }
}
