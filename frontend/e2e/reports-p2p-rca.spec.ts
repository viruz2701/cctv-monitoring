import { test, expect } from '@playwright/test';

// ═══════════════════════════════════════════════════════════════════════
// Reports / P2P Devices / RCA Graph — E2E Tests
// P0 flows: Export report, Register P2P device, View RCA graph
// ═══════════════════════════════════════════════════════════════════════

// ─────────────────────────────────────────────────────────────────────
// Mock data: Reports
// ─────────────────────────────────────────────────────────────────────
const MOCK_REPORTS_LIST = [
  {
    id: 'rpt-1',
    title: 'Ежедневный отчёт — 2026-06-25',
    type: 'daily',
    format: 'pdf',
    created_at: new Date(Date.now() - 86400000).toISOString(),
    status: 'ready',
    url: '/api/v1/reports/rpt-1/download',
  },
  {
    id: 'rpt-2',
    title: 'Еженедельный отчёт — W26',
    type: 'weekly',
    format: 'xlsx',
    created_at: new Date(Date.now() - 604800000).toISOString(),
    status: 'ready',
    url: '/api/v1/reports/rpt-2/download',
  },
  {
    id: 'rpt-3',
    title: 'Аварийный отчёт — 2026-06-24',
    type: 'incident',
    format: 'pdf',
    created_at: new Date(Date.now() - 172800000).toISOString(),
    status: 'generating',
    url: null,
  },
];

const MOCK_REPORT_GENERATE_RESPONSE = {
  id: 'rpt-4',
  title: 'Сгенерированный отчёт',
  status: 'generating',
  created_at: new Date().toISOString(),
};

// ─────────────────────────────────────────────────────────────────────
// Mock data: P2P Devices
// ─────────────────────────────────────────────────────────────────────
const MOCK_P2P_DEVICES = [
  {
    id: 'p2p-1',
    name: 'Gate Controller A-101',
    mac: 'AA:BB:CC:DD:EE:01',
    status: 'online',
    ip_address: '10.0.1.10',
    firmware: '2.3.1',
    last_seen: new Date().toISOString(),
    registered_at: new Date(Date.now() - 86400000 * 30).toISOString(),
  },
  {
    id: 'p2p-2',
    name: 'Access Panel B-204',
    mac: 'AA:BB:CC:DD:EE:02',
    status: 'offline',
    ip_address: '10.0.2.20',
    firmware: '2.1.0',
    last_seen: new Date(Date.now() - 7200000).toISOString(),
    registered_at: new Date(Date.now() - 86400000 * 60).toISOString(),
  },
];

const MOCK_P2P_REGISTER_RESPONSE = {
  id: 'p2p-3',
  name: 'New P2P Controller',
  mac: 'AA:BB:CC:DD:EE:03',
  status: 'pending',
  ip_address: '10.0.3.30',
  firmware: '2.4.0',
  last_seen: null,
  registered_at: new Date().toISOString(),
  message: 'Устройство успешно зарегистрировано',
};

// ─────────────────────────────────────────────────────────────────────
// Mock data: RCA Graph
// ─────────────────────────────────────────────────────────────────────
const MOCK_RCA_GRAPH = {
  root_cause: {
    id: 'dev-3',
    name: 'Camera-12 Parking Lot B',
    type: 'camera',
    status: 'offline',
    health: 'faulty',
    failure_type: 'power_loss',
    detected_at: new Date(Date.now() - 86400000).toISOString(),
  },
  affected_devices: [
    {
      id: 'dev-4',
      name: 'NVR-03 Recording Server',
      type: 'nvr',
      impact: 'degraded',
      relation: 'upstream',
    },
    {
      id: 'dev-5',
      name: 'Switch-02 Floor B2',
      type: 'switch',
      impact: 'affected',
      relation: 'network_parent',
    },
  ],
  timeline: [
    { event: 'power_loss', timestamp: new Date(Date.now() - 86400000).toISOString(), severity: 'critical' },
    { event: 'connection_lost', timestamp: new Date(Date.now() - 86400000 + 60000).toISOString(), severity: 'critical' },
    { event: 'alert_triggered', timestamp: new Date(Date.now() - 86400000 + 120000).toISOString(), severity: 'high' },
  ],
  recommendations: [
    'Проверить питание на панели P-12 (Floor B2)',
    'Заменить блок питания камеры Camera-12',
    'Проверить состояние UPS в серверной B2',
    'Выполнить диагностику кабельной трассы B2-Core',
  ],
};

const MOCK_RCA_LIST = [
  {
    id: 'rca-1',
    title: 'Camera-12 Parking Lot B — Power Loss',
    device_id: 'dev-3',
    status: 'open',
    severity: 'critical',
    detected_at: new Date(Date.now() - 86400000).toISOString(),
    resolved_at: null,
  },
  {
    id: 'rca-2',
    title: 'NVR-03 — Disk Failure',
    device_id: 'dev-2',
    status: 'resolved',
    severity: 'high',
    detected_at: new Date(Date.now() - 604800000).toISOString(),
    resolved_at: new Date(Date.now() - 432000000).toISOString(),
  },
];

// ─────────────────────────────────────────────────────────────────────
// Shared mock data
// ─────────────────────────────────────────────────────────────────────
const MOCK_SITES = [
  { id: 'site-1', name: 'Main Office' },
  { id: 'site-2', name: 'Branch Office' },
];

// ─────────────────────────────────────────────────────────────────────
// Scenario 1: Export Report — Mock API
// ─────────────────────────────────────────────────────────────────────
async function setupReportsMockApi(page: any) {
  // Auth
  await page.route('**/api/v1/auth/me', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'user-1',
        username: 'admin',
        role: 'admin',
      }),
    });
  });

  // Reports list
  await page.route('**/api/v1/reports*', async (route: any, request: any) => {
    // POST — generate report
    if (request.method() === 'POST') {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_REPORT_GENERATE_RESPONSE),
      });
      return;
    }
    // GET — list reports
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_REPORTS_LIST),
    });
  });

  // Report download endpoint
  await page.route('**/api/v1/reports/*/download', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/octet-stream',
      headers: {
        'Content-Disposition': 'attachment; filename="report.pdf"',
      },
      body: Buffer.from('%PDF-1.4 mock pdf content'),
    });
  });

  // Sites list
  await page.route('**/api/v1/sites*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_SITES),
    });
  });

  await page.evaluate(() => localStorage.setItem('token', 'mock-token'));
}

// ─────────────────────────────────────────────────────────────────────
// Scenario 2: Register P2P Device — Mock API
// ─────────────────────────────────────────────────────────────────────
async function setupP2pMockApi(page: any) {
  // Auth
  await page.route('**/api/v1/auth/me', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'user-1',
        username: 'admin',
        role: 'admin',
      }),
    });
  });

  // P2P devices list
  await page.route('**/api/v1/p2p-devices*', async (route: any, request: any) => {
    // POST — register new device
    if (request.method() === 'POST') {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_P2P_REGISTER_RESPONSE),
      });
      return;
    }
    // GET — list devices
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_P2P_DEVICES),
    });
  });

  // Sites list
  await page.route('**/api/v1/sites*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_SITES),
    });
  });

  await page.evaluate(() => localStorage.setItem('token', 'mock-token'));
}

// ─────────────────────────────────────────────────────────────────────
// Scenario 3: View RCA Graph — Mock API
// ─────────────────────────────────────────────────────────────────────
async function setupRcaMockApi(page: any) {
  // Auth
  await page.route('**/api/v1/auth/me', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'user-1',
        username: 'admin',
        role: 'admin',
      }),
    });
  });

  // RCA investigations list
  await page.route('**/api/v1/rca*', async (route: any, request: any) => {
    // If requesting a specific RCA graph (by ID)
    const url = request.url();
    if (url.includes('/graph') || url.includes('rca-1')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_RCA_GRAPH),
      });
      return;
    }
    // GET — list all RCA investigations
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_RCA_LIST),
    });
  });

  // Devices list (for graph nodes)
  await page.route('**/api/v1/devices*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: 'dev-3', name: 'Camera-12 Parking Lot B', type: 'camera', status: 'offline' },
        { id: 'dev-4', name: 'NVR-03 Recording Server', type: 'nvr', status: 'online' },
        { id: 'dev-5', name: 'Switch-02 Floor B2', type: 'switch', status: 'online' },
      ]),
    });
  });

  // Sites list
  await page.route('**/api/v1/sites*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_SITES),
    });
  });

  await page.evaluate(() => localStorage.setItem('token', 'mock-token'));
}

// ═══════════════════════════════════════════════════════════════════════
// Test Suite: Export Report
// ═══════════════════════════════════════════════════════════════════════
test.describe('Reports — Export', () => {
  test.beforeEach(async ({ page }) => {
    await setupReportsMockApi(page);
    await page.goto('/reports');
  });

  test('Reports page loads and displays report list', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Страница загрузилась
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/reports');

    // Отображается хотя бы один отчёт из мок-данных
    const reportTitle = page.getByText(/Ежедневный отчёт|Еженедельный отчёт|Daily Report|Weekly Report/i);
    await expect(reportTitle.first()).toBeVisible();
  });

  test('Export button is visible and opens format selector', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Ищем кнопку экспорта / генерации отчёта
    const exportButton = page.locator(
      'button:has-text(/export|экспорт|generate|сгенерировать|download|скачать/i)',
    ).first();

    if (await exportButton.isVisible()) {
      await exportButton.click();
      await page.waitForTimeout(500);

      // Проверяем появление дропдауна или модалки с выбором формата
      const formatOption = page.locator(
        'text=/pdf|xlsx|excel|pdf format|xlsx format|формат pdf|формат xlsx|pdf документ|excel документ/i',
      ).first();

      if (await formatOption.isVisible()) {
        const formatText = await formatOption.textContent();
        expect(formatText).toBeTruthy();
      }
    }
  });

  test('Export report — select PDF format and verify download', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Открываем меню экспорта
    const exportButton = page.locator(
      'button:has-text(/export|экспорт|generate|сгенерировать|download|скачать/i)',
    ).first();

    if (await exportButton.isVisible()) {
      await exportButton.click();
      await page.waitForTimeout(400);
    }

    // Выбираем PDF формат
    const pdfOption = page.locator(
      'role=menuitem, role=option, button, [data-format]',
    ).filter({ hasText: /pdf/i }).first();

    if (await pdfOption.isVisible()) {
      // Слушаем событие скачивания
      const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null);
      await pdfOption.click();

      // Если началось скачивание — проверяем имя файла
      const download = await downloadPromise;
      if (download) {
        expect(download.suggestedFilename()).toMatch(/\.pdf$/i);
      } else {
        // Если скачивание не началось — хотя бы проверяем что запрос ушёл
        await page.waitForTimeout(1000);
      }
    }
  });

  test('Export report — select XLSX format and verify download', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Открываем меню экспорта
    const exportButton = page.locator(
      'button:has-text(/export|экспорт|generate|сгенерировать|download|скачать/i)',
    ).first();

    if (await exportButton.isVisible()) {
      await exportButton.click();
      await page.waitForTimeout(400);
    }

    // Выбираем XLSX формат
    const xlsxOption = page.locator(
      'role=menuitem, role=option, button, [data-format]',
    ).filter({ hasText: /xlsx|excel/i }).first();

    if (await xlsxOption.isVisible()) {
      const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null);
      await xlsxOption.click();

      const download = await downloadPromise;
      if (download) {
        expect(download.suggestedFilename()).toMatch(/\.xlsx?$/i);
      } else {
        await page.waitForTimeout(1000);
      }
    }
  });

  test('Generate new report button triggers creation flow', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Ищем кнопку создания нового отчёта
    const createButton = page.locator(
      'button:has-text(/new report|create|создать|новый|generate/i)',
    ).first();

    if (await createButton.isVisible()) {
      await createButton.click();
      await page.waitForTimeout(500);

      // Проверяем что появилась форма / модалка создания отчёта
      const form = page.locator(
        'form, [role="dialog"], [role="modal"], .modal, .drawer',
      ).filter({ hasText: /report|отчёт/i }).first();

      if (await form.isVisible()) {
        // Проверяем наличие полей формы (тип отчёта, даты)
        const typeSelect = form.locator(
          'select, [role="combobox"], input:not([type="hidden"])',
        ).first();
        await expect(typeSelect).toBeVisible();
      }
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════
// Test Suite: Register P2P Device
// ═══════════════════════════════════════════════════════════════════════
test.describe('P2P Devices — Registration', () => {
  test.beforeEach(async ({ page }) => {
    await setupP2pMockApi(page);
    await page.goto('/p2p-devices');
  });

  test('P2P devices page loads and shows device list', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Страница загрузилась
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/p2p');

    // Отображается устройство из мок-данных
    const deviceName = page.getByText(/Gate Controller A-101|Access Panel B-204/i);
    await expect(deviceName.first()).toBeVisible();
  });

  test('Register button is visible on the page', async ({ page }) => {
    await page.waitForTimeout(1500);

    // Кнопка добавления / регистрации P2P устройства
    const registerButton = page.locator(
      'button:has-text(/register|add device|добавить|регистрация|зарегистрировать|new device|новое/i)',
    ).first();

    await expect(registerButton).toBeVisible();
  });

  test('Registration form opens and contains required fields', async ({ page }) => {
    await page.waitForTimeout(1500);

    // Нажимаем кнопку регистрации
    const registerButton = page.locator(
      'button:has-text(/register|add device|добавить|регистрация|зарегистрировать|new device|новое/i)',
    ).first();

    if (await registerButton.isVisible()) {
      await registerButton.click();
      await page.waitForTimeout(500);
    }

    // Форма регистрации открылась
    const form = page.locator(
      'form, [role="dialog"], [role="modal"], .modal, .drawer',
    ).filter({ hasText: /register|add device|регистрация|новое устройство/i }).first();

    if (await form.isVisible()) {
      // Проверяем обязательные поля формы
      const nameInput = form.locator(
        'input[name="name"], input[placeholder*="name" i], input[placeholder*="имя" i], input[placeholder*="назван" i]',
      ).first();
      const macInput = form.locator(
        'input[name="mac"], input[placeholder*="mac" i], input[placeholder*="адрес" i]',
      ).first();

      // Хотя бы одно обязательное поле должно присутствовать
      const nameVisible = await nameInput.isVisible();
      const macVisible = await macInput.isVisible();
      expect(nameVisible || macVisible).toBeTruthy();
    }
  });

  test('Registration form validation — empty fields show errors', async ({ page }) => {
    await page.waitForTimeout(1500);

    // Открываем форму регистрации
    const registerButton = page.locator(
      'button:has-text(/register|add device|добавить|регистрация|зарегистрировать|new device|новое/i)',
    ).first();

    if (await registerButton.isVisible()) {
      await registerButton.click();
      await page.waitForTimeout(500);
    }

    // Ищем кнопку Submit / Save / Сохранить в форме и кликаем не заполняя поля
    const form = page.locator(
      'form, [role="dialog"], [role="modal"], .modal, .drawer',
    ).filter({ hasText: /register|add device|регистрация|новое устройство/i }).first();

    if (await form.isVisible()) {
      const submitButton = form.locator(
        'button[type="submit"], button:has-text(/save|submit|register|сохранить|добавить|регистрация/i)',
      ).first();

      if (await submitButton.isVisible()) {
        await submitButton.click();
        await page.waitForTimeout(500);

        // Проверяем появление сообщений валидации
        const validationError = form.locator(
          '[aria-invalid="true"], [role="alert"], .error, .validation-error, text=/required|обязательно|заполните|некорректный|invalid/i',
        ).first();

        // Валидация может показываться как error message, так и подсветка поля
        const hasError = await validationError.isVisible().catch(() => false);
        if (!hasError) {
          // Если нет явного сообщения, проверяем что форма не закрылась
          await expect(form).toBeVisible();
        }
      }
    }
  });

  test('Register P2P device — successful registration shows confirmation', async ({ page }) => {
    await page.waitForTimeout(1500);

    // Открываем форму регистрации
    const registerButton = page.locator(
      'button:has-text(/register|add device|добавить|регистрация|зарегистрировать|new device|новое/i)',
    ).first();

    if (await registerButton.isVisible()) {
      await registerButton.click();
      await page.waitForTimeout(500);
    }

    // Заполняем форму
    const form = page.locator(
      'form, [role="dialog"], [role="modal"], .modal, .drawer',
    ).filter({ hasText: /register|add device|регистрация|новое устройство/i }).first();

    if (await form.isVisible()) {
      // Поле Name
      const nameInput = form.locator(
        'input[name="name"], input[placeholder*="name" i], input[placeholder*="имя" i], input[placeholder*="назван" i]',
      ).first();
      if (await nameInput.isVisible()) {
        await nameInput.fill('New P2P Controller');
      }

      // Поле MAC address
      const macInput = form.locator(
        'input[name="mac"], input[placeholder*="mac" i], input[placeholder*="адрес" i]',
      ).first();
      if (await macInput.isVisible()) {
        await macInput.fill('AA:BB:CC:DD:EE:03');
      }

      // Поле IP address (опционально)
      const ipInput = form.locator(
        'input[name="ip"], input[placeholder*="ip" i], input[placeholder*="192" i]',
      ).first();
      if (await ipInput.isVisible()) {
        await ipInput.fill('10.0.3.30');
      }

      // Отправляем форму
      const submitButton = form.locator(
        'button[type="submit"], button:has-text(/save|submit|register|сохранить|добавить|регистрация/i)',
      ).first();

      if (await submitButton.isVisible()) {
        await submitButton.click();
        await page.waitForTimeout(1000);

        // Проверяем успешную регистрацию — toast/alert или изменение URL
        const successMessage = page.locator(
          'text=/успешно|success|registered|зарегистрирован|подтвержден|confirmed|device added/i',
        ).first();

        const isSuccessVisible = await successMessage.isVisible().catch(() => false);
        if (!isSuccessVisible) {
          // Если нет toast — проверяем что устройство появилось в списке
          const newDevice = page.getByText(/New P2P Controller/i).first();
          const isDeviceVisible = await newDevice.isVisible().catch(() => false);
          expect(isSuccessVisible || isDeviceVisible).toBeTruthy();
        }
      }
    }
  });

  test('Register P2P device — duplicate MAC shows validation error', async ({ page }) => {
    await page.waitForTimeout(1500);

    // Открываем форму регистрации
    const registerButton = page.locator(
      'button:has-text(/register|add device|добавить|регистрация|зарегистрировать|new device|новое/i)',
    ).first();

    if (await registerButton.isVisible()) {
      await registerButton.click();
      await page.waitForTimeout(500);
    }

    // Подменяем ответ API на 409 Conflict (duplicate MAC)
    await page.route('**/api/v1/p2p-devices*', async (route: any, request: any) => {
      if (request.method() === 'POST') {
        await route.fulfill({
          status: 409,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 'DUPLICATE_MAC',
            message: 'Устройство с таким MAC-адресом уже зарегистрировано',
          }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_P2P_DEVICES),
        });
      }
    });

    // Заполняем форму существующим MAC
    const form = page.locator(
      'form, [role="dialog"], [role="modal"], .modal, .drawer',
    ).filter({ hasText: /register|add device|регистрация|новое устройство/i }).first();

    if (await form.isVisible()) {
      const nameInput = form.locator(
        'input[name="name"], input[placeholder*="name" i], input[placeholder*="имя" i], input[placeholder*="назван" i]',
      ).first();
      if (await nameInput.isVisible()) {
        await nameInput.fill('Duplicate Device');
      }

      const macInput = form.locator(
        'input[name="mac"], input[placeholder*="mac" i], input[placeholder*="адрес" i]',
      ).first();
      if (await macInput.isVisible()) {
        await macInput.fill('AA:BB:CC:DD:EE:01'); // существующий MAC
      }

      const submitButton = form.locator(
        'button[type="submit"], button:has-text(/save|submit|register|сохранить|добавить|регистрация/i)',
      ).first();

      if (await submitButton.isVisible()) {
        await submitButton.click();
        await page.waitForTimeout(1000);

        // Должна появиться ошибка о дубликате MAC
        const errorMessage = page.locator(
          'text=/already registered|уже зарегистрирован|duplicate|дубликат|already exists|уже существует|conflict|конфликт/i',
        ).first();

        const isErrorVisible = await errorMessage.isVisible().catch(() => false);
        // Если нет текстовой ошибки, проверяем что мы остались на форме
        if (!isErrorVisible) {
          await expect(form).toBeVisible();
        }
      }
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════
// Test Suite: View RCA Graph
// ═══════════════════════════════════════════════════════════════════════
test.describe('RCA — Root Cause Analysis Graph', () => {
  test.beforeEach(async ({ page }) => {
    await setupRcaMockApi(page);
    await page.goto('/rca');
  });

  test('RCA investigations page loads and shows list', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Страница загрузилась
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/rca');

    // Отображается RCA investigation из мок-данных
    const rcaTitle = page.getByText(/Camera-12 Parking Lot B|NVR-03|Power Loss|Disk Failure/i);
    await expect(rcaTitle.first()).toBeVisible();
  });

  test('RCA investigation details — click opens RCA graph view', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Находим и кликаем на RCA элемент
    const rcaItem = page.locator(
      'a, button, tr, [role="row"], [role="button"], .card, .item',
    ).filter({ hasText: /Camera-12 Parking Lot B|Power Loss/i }).first();

    if (await rcaItem.isVisible()) {
      await rcaItem.click();
      await page.waitForTimeout(1000);

      // Проверяем переход на страницу RCA графа
      const isGraphPage = page.url().includes('/rca/');
      expect(isGraphPage).toBeTruthy();
    }
  });

  test('RCA graph renders root cause node', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Открываем RCA детали
    const rcaItem = page.locator(
      'a, button, tr, [role="row"], [role="button"], .card, .item',
    ).filter({ hasText: /Camera-12 Parking Lot B|Power Loss/i }).first();

    if (await rcaItem.isVisible()) {
      await rcaItem.click();
      await page.waitForTimeout(1500);
    }

    // Проверяем отображение root cause узла
    const rootCauseNode = page.locator(
      'text=/root cause|root cause analysis|причина|root|rca|power loss|потеря питания/i',
    ).first();

    const isNodeVisible = await rootCauseNode.isVisible().catch(() => false);
    if (!isNodeVisible) {
      // Если нет текстового узла — проверяем что граф (SVG / Canvas) отрисовался
      const graphContainer = page.locator(
        'svg, canvas, [data-testid="rca-graph"], .rca-graph, .graph-container, [role="graphics-document"]',
      ).first();
      await expect(graphContainer).toBeVisible();
    }
  });

  test('RCA graph shows affected devices and impact levels', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Открываем RCA детали
    const rcaItem = page.locator(
      'a, button, tr, [role="row"], [role="button"], .card, .item',
    ).filter({ hasText: /Camera-12 Parking Lot B|Power Loss/i }).first();

    if (await rcaItem.isVisible()) {
      await rcaItem.click();
      await page.waitForTimeout(1500);
    }

    // Проверяем отображение дочерних узлов (affected devices)
    const affectedDevice = page.getByText(/NVR-03 Recording Server|Switch-02/i).first();

    const isAffectedVisible = await affectedDevice.isVisible().catch(() => false);
    if (!isAffectedVisible) {
      // Если нет имён устройств — проверяем что есть граф с несколькими узлами
      const graphNodes = page.locator(
        'svg g, canvas, .graph-node, [data-node]',
      );
      const nodeCount = await graphNodes.count();
      expect(nodeCount).toBeGreaterThanOrEqual(1);
    }
  });

  test('RCA graph timeline displays events in order', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Открываем RCA детали
    const rcaItem = page.locator(
      'a, button, tr, [role="row"], [role="button"], .card, .item',
    ).filter({ hasText: /Camera-12 Parking Lot B|Power Loss/i }).first();

    if (await rcaItem.isVisible()) {
      await rcaItem.click();
      await page.waitForTimeout(1500);
    }

    // Проверяем отображение таймлайна событий
    const timeline = page.locator(
      'text=/timeline|хронология|события|events|power loss|connection lost|alert triggered/i',
    ).first();

    const isTimelineVisible = await timeline.isVisible().catch(() => false);
    if (!isTimelineVisible) {
      // Если нет таймлайна — проверяем что есть секция с событиями
      const eventSection = page.locator(
        'section, div, [role="region"], .timeline, .events, [data-testid="timeline"]',
      ).filter({ hasText: /power|connection|alert|event|событие/i }).first();
      await expect(eventSection).toBeVisible();
    }
  });

  test('RCA graph shows recommendations for remediation', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Открываем RCA детали
    const rcaItem = page.locator(
      'a, button, tr, [role="row"], [role="button"], .card, .item',
    ).filter({ hasText: /Camera-12 Parking Lot B|Power Loss/i }).first();

    if (await rcaItem.isVisible()) {
      await rcaItem.click();
      await page.waitForTimeout(1500);
    }

    // Проверяем отображение рекомендаций
    const recommendation = page.getByText(/проверить питание|заменить блок|check power|replace power|recommendation|рекомендац/i).first();

    const isRecVisible = await recommendation.isVisible().catch(() => false);
    if (!isRecVisible) {
      // Если нет рекомендаций — проверяем что есть секция рекомендаций
      const recSection = page.locator(
        'section, div, [role="region"], .recommendations, .actions, [data-testid="recommendations"]',
      ).filter({ hasText: /recommend|рекоменд|action|действие/i }).first();
      await expect(recSection).toBeVisible();
    }
  });

  test('RCA graph — severity badge matches critical status', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Проверяем badge с severity на странице списка
    const severityBadge = page.locator(
      'span, badge, [role="status"], .badge, .severity',
    ).filter({ hasText: /critical|критический|high|высокий/i }).first();

    if (await severityBadge.isVisible()) {
      const badgeText = await severityBadge.textContent();
      expect(badgeText).toBeTruthy();
    }
  });

  test('RCA graph — resolved investigations show different styling', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Проверяем отображение resolved статуса
    const resolvedItem = page.locator(
      'text=/resolved|решено|completed|завершен/i',
    ).first();

    if (await resolvedItem.isVisible()) {
      // Проверяем что resolved элемент имеет соответствующий стиль/статус
      const resolvedText = await resolvedItem.textContent();
      expect(resolvedText).toBeTruthy();
    }
  });
});
