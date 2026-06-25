// ═══════════════════════════════════════════════════════════════════════
// Helpers — Array, data, and type-safe utility functions
// ═══════════════════════════════════════════════════════════════════════

/**
 * getArrayData — безопасное извлечение массива из ответа API.
 *
 * Бэкенд может вернуть данные в двух форматах:
 *   1. Чистый массив:  `[item1, item2, ...]`
 *   2. Обёрнутый объект: `{ data: [item1, item2, ...] }`
 *
 * При ошибке (502/500) или нестандартном формате возвращает пустой массив,
 * предотвращая `xxx.map is not a function`.
 *
 * @example
 * const items = getArrayData<Device>(rawDevices);
 * // rawDevices = [d1, d2]           → [d1, d2]
 * // rawDevices = { data: [d1] }     → [d1]
 * // rawDevices = null | undefined   → []
 */
export function getArrayData<T>(raw: unknown): T[] {
  if (Array.isArray(raw)) return raw as T[];
  if (raw && typeof raw === 'object' && 'data' in raw) {
    const nested = (raw as Record<string, unknown>).data;
    if (Array.isArray(nested)) return nested as T[];
  }
  return [];
}
