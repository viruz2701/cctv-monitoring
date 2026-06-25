import { test, expect } from '@playwright/test';

// ═══════════════════════════════════════════════════════════════════════
// Login Page — E2E Tests
// P0 flow: authentication gateway
// ═══════════════════════════════════════════════════════════════════════

test.describe('Login Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('Login page loads with form elements', async ({ page }) => {
    // Проверяем основные элементы страницы
    await expect(page.locator('h2')).toContainText(/sign in|login|войти/i);
    await expect(page.locator('input[type="email"]')).toBeVisible();
    await expect(page.locator('input[type="password"]')).toBeVisible();
    await expect(page.getByRole('button', { name: /sign in|login|войти/i })).toBeVisible();
  });

  test('Form validation — empty fields show error', async ({ page }) => {
    // Кликаем по кнопке без заполнения полей
    await page.getByRole('button', { name: /sign in|login|войти/i }).click();

    // Проверяем появление ошибки валидации
    await expect(page.locator('text=Please enter your email and password')).toBeVisible();
  });

  test('Form validation — invalid password shows error', async ({ page }) => {
    // Заполняем email коротким паролем
    await page.locator('input[type="email"]').fill('admin@test.com');
    await page.locator('input[type="password"]').fill('123');
    await page.getByRole('button', { name: /sign in|login|войти/i }).click();

    // Проверяем ошибку валидации пароля
    await expect(page.locator('text=Password must be at least 6 characters')).toBeVisible();
  });

  test('Successful login redirects to dashboard', async ({ page }) => {
    // Мокаем успешный ответ API
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          token: 'mock-jwt-token',
          user: {
            id: '1',
            username: 'admin',
            role: 'admin',
          },
        }),
      });
    });

    // Мокаем запрос текущего пользователя
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: '1',
          username: 'admin',
          role: 'admin',
        }),
      });
    });

    // Заполняем форму
    await page.locator('input[type="email"]').fill('admin@test.com');
    await page.locator('input[type="password"]').fill('password123');
    await page.getByRole('button', { name: /sign in|login|войти/i }).click();

    // Ждем редиректа на dashboard
    await page.waitForURL('**/dashboard');
    await expect(page).toHaveURL(/dashboard/);
  });

  test('Shows 2FA step when required', async ({ page }) => {
    // Мокаем ответ с требованием 2FA
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 202,
        contentType: 'application/json',
        body: JSON.stringify({
          requires2FA: true,
          session_token: 'mock-session-token',
        }),
      });
    });

    // Заполняем форму
    await page.locator('input[type="email"]').fill('admin@test.com');
    await page.locator('input[type="password"]').fill('password123');
    await page.getByRole('button', { name: /sign in|login|войти/i }).click();

    // Проверяем появление поля для OTP
    await expect(page.locator('input[type="text"]')).toBeVisible();
    await expect(page.getByText(/2FA|two-factor|auth code/i)).toBeVisible();
  });

  test('Toggle password visibility', async ({ page }) => {
    // Проверяем toggle показа пароля
    const passwordInput = page.locator('input[type="password"]');
    await expect(passwordInput).toBeVisible();

    // Кликаем на иконку глаза
    const toggleButton = page.locator('button[aria-label*="password" i], button:has(svg)').last();
    await toggleButton.click();

    // Поле должно сменить тип на text
    await expect(page.locator('input[type="text"]')).toBeVisible();
  });
});
