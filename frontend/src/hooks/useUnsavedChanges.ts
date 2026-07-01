// ═══════════════════════════════════════════════════════════════════════
// useUnsavedChanges — защита от потери данных при навигации (P0-CR-12)
//
// Предотвращает случайную потерю заполненных форм при:
//   - навигации по history.back/forward
//   - клике по ссылке внутри приложения
//   - закрытии вкладки/окна (через beforeunload)
//
// Использование:
//   const [isDirty, setDirty] = useState(false);
//   useUnsavedChanges(isDirty);
//   // ... setDirty(true) при изменении формы
//   // ... setDirty(false) после сохранения
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — prevention of data loss)
//   - ОАЦ РБ (Защита от потери данных оператора)
// ═══════════════════════════════════════════════════════════════════════

import { useEffect, useCallback } from 'react';
import { useBlocker } from 'react-router-dom';

/**
 * useUnsavedChanges блокирует навигацию при наличии несохранённых изменений.
 *
 * @param isDirty - true если есть несохранённые изменения
 * @param message - сообщение для beforeunload (браузерное, показывается при закрытии вкладки)
 */
export function useUnsavedChanges(
  isDirty: boolean,
  message: string = 'У вас есть несохранённые изменения. Вы уверены, что хотите покинуть страницу?',
) {
  // Блокировка навигации внутри SPA (React Router v7 useBlocker)
  const blocker = useBlocker(
    useCallback(() => isDirty, [isDirty]),
  );

  // Показываем confirm диалог при попытке навигации
  useEffect(() => {
    if (blocker.state === 'blocked') {
      const confirmLeave = window.confirm(message);
      if (confirmLeave) {
        blocker.proceed();
      } else {
        blocker.reset();
      }
    }
  }, [blocker, message]);

  // Блокировка закрытия вкладки/браузера (beforeunload)
  useEffect(() => {
    if (!isDirty) return;

    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = message;
      return message;
    };

    window.addEventListener('beforeunload', handleBeforeUnload);
    return () => window.removeEventListener('beforeunload', handleBeforeUnload);
  }, [isDirty, message]);
}
