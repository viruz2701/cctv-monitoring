import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useLocalStorage } from '../useLocalStorage';

// ═══════════════════════════════════════════════════════════════════════
// useLocalStorage — unit tests
// ═══════════════════════════════════════════════════════════════════════

describe('useLocalStorage', () => {
  const testKey = 'test_key';

  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('returns initial value when localStorage is empty', () => {
    const { result } = renderHook(() => useLocalStorage(testKey, 'default'));
    expect(result.current[0]).toBe('default');
  });

  it('returns stored value from localStorage', () => {
    localStorage.setItem(testKey, JSON.stringify('stored'));
    const { result } = renderHook(() => useLocalStorage(testKey, 'default'));
    expect(result.current[0]).toBe('stored');
  });

  it('updates value and writes to localStorage', () => {
    const { result } = renderHook(() => useLocalStorage(testKey, 'default'));

    act(() => {
      result.current[1]('new_value');
    });

    expect(result.current[0]).toBe('new_value');
    expect(JSON.parse(localStorage.getItem(testKey)!)).toBe('new_value');
  });

  it('updates value with function updater', () => {
    const { result } = renderHook(() => useLocalStorage<number>(testKey, 0));

    act(() => {
      result.current[1]((prev) => prev + 1);
    });

    expect(result.current[0]).toBe(1);
    expect(JSON.parse(localStorage.getItem(testKey)!)).toBe(1);
  });

  it('handles complex objects', () => {
    const initial = { mode: 'deadline', count: 0 };
    const { result } = renderHook(() => useLocalStorage(testKey, initial));

    act(() => {
      result.current[1]({ mode: 'creation', count: 1 });
    });

    expect(result.current[0]).toEqual({ mode: 'creation', count: 1 });
    expect(JSON.parse(localStorage.getItem(testKey)!)).toEqual({ mode: 'creation', count: 1 });
  });

  it('falls back to initial value on corrupt data', () => {
    localStorage.setItem(testKey, '{invalid json');
    const { result } = renderHook(() => useLocalStorage(testKey, 'fallback'));
    expect(result.current[0]).toBe('fallback');
  });

  it('persists across rerenders with same key', () => {
    const { result, rerender } = renderHook(() => useLocalStorage(testKey, 'default'));

    act(() => {
      result.current[1]('persisted');
    });

    rerender();
    expect(result.current[0]).toBe('persisted');
  });

  it('works with DateMode type (deadline/creation)', () => {
    type DateMode = 'deadline' | 'creation';
    const { result } = renderHook(() => useLocalStorage<DateMode>(testKey, 'deadline'));

    expect(result.current[0]).toBe('deadline');

    act(() => {
      result.current[1]('creation');
    });

    expect(result.current[0]).toBe('creation');
    expect(localStorage.getItem(testKey)).toBe('"creation"');
  });

  it('handles multiple independent keys', () => {
    const { result: r1 } = renderHook(() => useLocalStorage('key1', 'a'));
    const { result: r2 } = renderHook(() => useLocalStorage('key2', 0));

    act(() => { r1.current[1]('A'); });
    act(() => { r2.current[1](42); });

    expect(r1.current[0]).toBe('A');
    expect(r2.current[0]).toBe(42);
  });
});
