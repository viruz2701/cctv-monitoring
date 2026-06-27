import { useState, useCallback } from 'react';

// ═══════════════════════════════════════════════════════════════════════
// useLocalStorage — синхронизирует состояние с localStorage
// ═══════════════════════════════════════════════════════════════════════
// Generic хук для персистентного хранения предпочтений пользователя.
// Автоматически сериализует/десериализует JSON.
// При ошибке парсинга возвращает default value.
// ═══════════════════════════════════════════════════════════════════════

export function useLocalStorage<T>(
  key: string,
  initialValue: T,
): [T, (value: T | ((prev: T) => T)) => void] {
  const [storedValue, setStoredValue] = useState<T>(() => {
    try {
      const item = localStorage.getItem(key);
      if (item === null) return initialValue;
      return JSON.parse(item) as T;
    } catch {
      // Если localStorage недоступен или данные повреждены
      return initialValue;
    }
  });

  const setValue = useCallback(
    (value: T | ((prev: T) => T)) => {
      setStoredValue((prev) => {
        const nextValue = value instanceof Function ? value(prev) : value;
        try {
          localStorage.setItem(key, JSON.stringify(nextValue));
        } catch {
          // localStorage может быть заполнен или недоступен
        }
        return nextValue;
      });
    },
    [key],
  );

  return [storedValue, setValue];
}
