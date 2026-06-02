import { listCurrencies } from '../api/currencies';
import type { CurrencyOption } from '../types';

type CurrencyCacheEntry = {
  symbolNative?: string;
  decimalDigits?: number;
};

const MAP_KEY = 'spendsense-currency-map';
let currencyMap: Record<string, CurrencyCacheEntry> = {};

function normalize(code?: string) {
  return (code || '').trim().toUpperCase();
}

async function fetchAndCache() {
  try {
    const list = await listCurrencies();
    const map: Record<string, CurrencyCacheEntry> = {};
    for (const c of list) {
      map[normalize(c.code)] = {
        symbolNative: c.symbol_native || c.symbol,
        decimalDigits: c.decimal_digits ?? 2,
      };
    }
    currencyMap = map;
    try {
      if (typeof window !== 'undefined' && window.localStorage) {
        window.localStorage.setItem(MAP_KEY, JSON.stringify(map));
      }
    } catch {}
  } catch {}
}

// Initialize from localStorage (fast) and refresh in background
export function initCurrencies() {
  if (typeof window === 'undefined') return;
  try {
    const raw = window.localStorage.getItem(MAP_KEY);
    if (raw) {
      currencyMap = JSON.parse(raw);
    }
  } catch {}
  // refresh in background
  void fetchAndCache();
}

export function getSymbolNativeSync(code?: string): string | undefined {
  return currencyMap[normalize(code)]?.symbolNative;
}

export function getDecimalDigitsSync(code?: string): number {
  return currencyMap[normalize(code)]?.decimalDigits ?? 2;
}

export function seedCurrencyCache(currencies: CurrencyOption[]) {
  const map: Record<string, CurrencyCacheEntry> = {};
  for (const currency of currencies) {
    map[normalize(currency.code)] = {
      symbolNative: currency.symbol_native || currency.symbol,
      decimalDigits: currency.decimal_digits ?? 2,
    };
  }
  currencyMap = map;
  try {
    if (typeof window !== 'undefined' && window.localStorage) {
      window.localStorage.setItem(MAP_KEY, JSON.stringify(map));
    }
  } catch {}
}

// allow manual refresh if needed
export function refreshCurrencies() {
  return fetchAndCache();
}

export default {
  initCurrencies,
  getSymbolNativeSync,
  getDecimalDigitsSync,
  seedCurrencyCache,
  refreshCurrencies,
};
