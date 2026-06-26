// ──────────────────────────────────────────────────
// Вспомогательные функции для Detox E2E тестов
// ──────────────────────────────────────────────────

/**
 * Ожидание появления элемента с ретраями.
 * Используется для стабильности тестов на медленных эмуляторах.
 */
export async function waitForElement(
  element: Detox.NativeElement,
  timeout: number = 8000,
): Promise<void> {
  await waitFor(element).toBeVisible().withTimeout(timeout);
}

/**
 * Ожидание исчезновения элемента.
 */
export async function waitForElementToDisappear(
  element: Detox.NativeElement,
  timeout: number = 8000,
): Promise<void> {
  await waitFor(element).not.toBeVisible().withTimeout(timeout);
}

/**
 * Скролл к элементу по идентификатору.
 */
export async function scrollToElement(
  scrollViewId: string,
  elementId: string,
  direction: 'down' | 'up' = 'down',
): Promise<void> {
  await waitFor(element(by.id(elementId)))
    .toBeVisible()
    .whileElement(by.id(scrollViewId))
    .scroll(50, direction);
}

/**
 * Тап по элементу с ожиданием.
 */
export async function tapAndWait(
  element: Detox.NativeElement,
  targetElement: Detox.NativeElement,
  timeout: number = 8000,
): Promise<void> {
  await element.tap();
  await waitFor(targetElement).toBeVisible().withTimeout(timeout);
}

/**
 * Ввод текста с очисткой поля.
 */
export async function typeText(
  inputElement: Detox.NativeElement,
  text: string,
): Promise<void> {
  await inputElement.tap();
  await inputElement.clearText();
  await inputElement.typeText(text);
}

/**
 * Ожидание выполнения анимации/перехода.
 */
export async function waitForAnimation(milliseconds: number = 1500): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}

/**
 * Получить mock-сервер URL для Device Mock API.
 */
export function getMockApiUrl(path: string): string {
  const baseUrl = 'http://localhost:3000';
  return `${baseUrl}${path}`;
}
