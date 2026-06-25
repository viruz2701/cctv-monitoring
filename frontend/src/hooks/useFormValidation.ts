import { useState, useCallback, useRef, useEffect } from 'react';
import type { ZodSchema, ZodError } from 'zod';

interface UseFormValidationResult<T> {
  errors: Record<string, string>;
  validate: (data: T) => boolean;
  validateField: (field: string, data: T) => void;
  isValid: boolean;
  touched: Set<string>;
  setTouched: (field: string) => void;
  setErrors: (errors: Record<string, string>) => void;
  reset: () => void;
}

/**
 * useFormValidation — хук для real-time Zod валидации формы.
 *
 * Особенности:
 * - `validateField` имеет debounce 300ms для real-time feedback
 * - `touched` отслеживает какие поля были изменены (UI feedback)
 * - `errors` — Record<string, string> с сообщениями об ошибках
 * - `isValid` — флаг валидности всей формы
 *
 * @example
 * ```tsx
 * const { errors, validate, validateField, isValid, touched } = useFormValidation(siteSchema);
 *
 * const handleSubmit = (e: FormEvent) => {
 *   e.preventDefault();
 *   if (validate(formData)) {
 *     // submit
 *   }
 * };
 * ```
 */
export function useFormValidation<T>(
  schema: ZodSchema<T>,
  options?: { debounceMs?: number }
): UseFormValidationResult<T> {
  const debounceMs = options?.debounceMs ?? 300;
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [touched, setTouchedState] = useState<Set<string>>(new Set());
  const debounceTimers = useRef<Record<string, ReturnType<typeof setTimeout>>>({});
  const isValid = Object.keys(errors).length === 0;

  // Cleanup debounce timers on unmount
  useEffect(() => {
    return () => {
      Object.values(debounceTimers.current).forEach(clearTimeout);
    };
  }, []);

  const setTouched = useCallback((field: string) => {
    setTouchedState(prev => {
      if (prev.has(field)) return prev;
      const next = new Set(prev);
      next.add(field);
      return next;
    });
  }, []);

  const validate = useCallback(
    (data: T): boolean => {
      const result = schema.safeParse(data);
      if (result.success) {
        setErrors({});
        return true;
      }

      const fieldErrors: Record<string, string> = {};
      const zodError = result.error as ZodError;
      for (const issue of zodError.issues) {
        const path = issue.path.join('.');
        if (!fieldErrors[path]) {
          fieldErrors[path] = issue.message;
        }
      }
      setErrors(fieldErrors);
      return false;
    },
    [schema]
  );

  const validateField = useCallback(
    (field: string, data: T) => {
      // Clear previous debounce timer for this field
      if (debounceTimers.current[field]) {
        clearTimeout(debounceTimers.current[field]);
      }

      debounceTimers.current[field] = setTimeout(() => {
        // Mark field as touched
        setTouched(field);

        // Validate the full object and extract error for this specific field
        const result = schema.safeParse(data);

        if (result.success) {
          setErrors(prev => {
            const next = { ...prev };
            delete next[field];
            return next;
          });
        } else {
          const zodError = result.error as ZodError;
          const fieldIssue = zodError.issues.find(issue => issue.path[0] === field);
          if (fieldIssue) {
            setErrors(prev => ({ ...prev, [field]: fieldIssue.message }));
          } else {
            // Field is valid, remove its error
            setErrors(prev => {
              const next = { ...prev };
              delete next[field];
              return next;
            });
          }
        }
      }, debounceMs);
    },
    [schema, debounceMs, setTouched]
  );

  const reset = useCallback(() => {
    // Clear all debounce timers
    Object.values(debounceTimers.current).forEach(clearTimeout);
    debounceTimers.current = {};
    setErrors({});
    setTouchedState(new Set());
  }, []);

  return {
    errors,
    validate,
    validateField,
    isValid,
    touched,
    setTouched,
    setErrors,
    reset,
  };
}
