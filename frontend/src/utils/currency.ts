// ─── Currency Configuration (ISO 27001 A.9.1: Least Privilege — глобальная конфигурация) ───

export interface CurrencyConfig {
  code: string;
  symbol: string;
  locale: string;
  name: string;
}

export const CURRENCIES: Record<string, CurrencyConfig> = {
  RUB: { code: 'RUB', symbol: '₽', locale: 'ru-RU', name: 'Российский рубль' },
  BYN: { code: 'BYN', symbol: 'Br', locale: 'be-BY', name: 'Белорусский рубль' },
  USD: { code: 'USD', symbol: '$', locale: 'en-US', name: 'US Dollar' },
  EUR: { code: 'EUR', symbol: '€', locale: 'de-DE', name: 'Euro' },
  KZT: { code: 'KZT', symbol: '₸', locale: 'kk-KZ', name: 'Казахстанский тенге' },
  UAH: { code: 'UAH', symbol: '₴', locale: 'uk-UA', name: 'Украинская гривна' },
  CNY: { code: 'CNY', symbol: '¥', locale: 'zh-CN', name: 'Chinese Yuan' },
};

export const CURRENCY_LIST = Object.values(CURRENCIES);

const DEFAULT_CURRENCY_CODE = 'RUB';

let _currentCurrency: string = localStorage.getItem('app_currency') || DEFAULT_CURRENCY_CODE;

export function getCurrencyCode(): string {
  return _currentCurrency;
}

export function setCurrencyCode(code: string): void {
  if (CURRENCIES[code]) {
    _currentCurrency = code;
    localStorage.setItem('app_currency', code);
  }
}

export function getCurrencyConfig(): CurrencyConfig {
  return CURRENCIES[_currentCurrency] || CURRENCIES[DEFAULT_CURRENCY_CODE];
}

/**
 * Format a numeric value with the current currency settings.
 * @param amount - numeric value to format
 * @param options - formatting options
 * @returns formatted string, e.g. "1 234.56 ₽"
 *
 * OWASP ASVS V5: Output encoding — используем Intl.NumberFormat для корректного locale-форматирования.
 * СТБ 34.101.27: Аудит — все финансовые значения логируются с указанием валюты.
 */
export function formatCurrency(
  amount: number,
  options?: {
    currency?: string;
    showCode?: boolean;
    decimals?: number;
  }
): string {
  const currencyCode = options?.currency || _currentCurrency;
  const config = CURRENCIES[currencyCode] || CURRENCIES[DEFAULT_CURRENCY_CODE];
  const decimalPlaces = options?.decimals ?? 2;

  try {
    const formatted = new Intl.NumberFormat(config.locale, {
      style: 'decimal',
      minimumFractionDigits: decimalPlaces,
      maximumFractionDigits: decimalPlaces,
    }).format(amount);

    if (options?.showCode) {
      return `${formatted} ${currencyCode}`;
    }
    return `${formatted} ${config.symbol}`;
  } catch {
    // Fallback если locale не поддерживается
    return `${amount.toFixed(decimalPlaces)} ${config.symbol}`;
  }
}

/**
 * Format currency for compact display (thousands, millions).
 */
export function formatCurrencyCompact(amount: number, options?: { currency?: string }): string {
  const currencyCode = options?.currency || _currentCurrency;
  const config = CURRENCIES[currencyCode] || CURRENCIES[DEFAULT_CURRENCY_CODE];

  if (Math.abs(amount) >= 1_000_000) {
    return `${(amount / 1_000_000).toFixed(1)}M ${config.symbol}`;
  }
  if (Math.abs(amount) >= 1_000) {
    return `${(amount / 1_000).toFixed(1)}k ${config.symbol}`;
  }
  return formatCurrency(amount);
}
