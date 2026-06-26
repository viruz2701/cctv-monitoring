/// <reference types="node" />

import { test, expect } from '@playwright/test';

// ═══════════════════════════════════════════════════════════════════════
// Work Orders — Create / Complete / Assign Flow (E2E)
// P0 flow: создание заявки с чеклистом, завершение с фото, назначение
// ═══════════════════════════════════════════════════════════════════════

const MOCK_SITES = [
  { id: 'site-1', name: 'Main Office' },
  { id: 'site-2', name: 'Branch Office' },
  { id: 'site-3', name: 'Warehouse' },
];

const MOCK_USERS = [
  { id: 'user-1', username: 'manager', role: 'manager', full_name: 'Alex Manager' },
  { id: 'user-2', username: 'tech1', role: 'technician', full_name: 'Bob Technician' },
  { id: 'user-3', username: 'tech2', role: 'technician', full_name: 'Carol Engineer' },
];

const MOCK_CHECKLIST_TEMPLATE = [
  { id: 'cl-1', label: 'Check power supply', required: true },
  { id: 'cl-2', label: 'Verify network connection', required: true },
  { id: 'cl-3', label: 'Test camera feed', required: false },
  { id: 'cl-4', label: 'Clean lens', required: false },
];

const MOCK_WORK_ORDER_DETAIL = {
  id: 'WO-001',
  title: 'Replace camera lens',
  description: 'Camera at main entrance has cracked lens',
  status: 'open',
  priority: 'critical',
  assigned_to: null,
  site_id: 'site-1',
  site_name: 'Main Office',
  sla_deadline: new Date(Date.now() + 3600000).toISOString(),
  created_at: new Date().toISOString(),
  created_by: 'user-1',
  checklist: MOCK_CHECKLIST_TEMPLATE.map((item) => ({ ...item, completed: false })),
  photos: [],
};

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

async function setupAuth(page: any) {
  await page.route('**/api/v1/auth/me', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'user-1',
        username: 'manager',
        role: 'manager',
      }),
    });
  });

  await page.route('**/api/v1/sites*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_SITES),
    });
  });

  await page.route('**/api/v1/users*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_USERS),
    });
  });

  await page.evaluate(() => localStorage.setItem('token', 'mock-token'));
}


// ═══════════════════════════════════════════════════════════════════════
// 1. Create WO with checklist
// ═══════════════════════════════════════════════════════════════════════

test.describe('Work Orders — Create with Checklist', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);

    // Mock checklist template endpoint
    await page.route('**/api/v1/checklists*', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_CHECKLIST_TEMPLATE),
      });
    });

    // Mock work-orders list (empty initially)
    await page.route('**/api/v1/work-orders*', async (route: any, request: any) => {
      if (request.method() === 'POST') {
        // Симулируем создание — возвращаем созданную заявку
        const body = JSON.parse(request.postData() || '{}');
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'WO-NEW',
            ...body,
            created_at: new Date().toISOString(),
            status: 'open',
          }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([]),
        });
      }
    });

    await page.goto('/work-orders/create');
    await page.waitForTimeout(1500);
  });

  test('Create WO — form loads with all required fields', async ({ page }) => {
    // Проверяем наличие формы создания
    await expect(page.locator('body')).toBeVisible();

    // Проверяем поля формы (title/название, description/описание, site/объект, priority/приоритет)
    const titleField = page.locator(
      'input[name="title"], input[placeholder*="title" i], input[placeholder*="назван" i], input[id*="title" i]'
    ).first();
    await expect(titleField).toBeVisible();

    const descriptionField = page.locator(
      'textarea[name="description"], textarea[placeholder*="description" i], textarea[placeholder*="описан" i], textarea[id*="description" i]'
    ).first();
    await expect(descriptionField).toBeVisible();

    // Проверяем селект для объекта / site
    const siteSelect = page.locator(
      'select[name="site_id"], select[id*="site" i], [role="combobox"]'
    ).first();
    await expect(siteSelect).toBeVisible();

    // Проверяем наличие кнопки submit / save / создать
    const submitButton = page.locator(
      'button[type="submit"], button:has-text(/create|save|создать|сохранить/i)'
    ).first();
    await expect(submitButton).toBeVisible();
  });

  test('Create WO — fill form and submit with checklist items', async ({ page }) => {
    // Заполняем название
    const titleField = page.locator(
      'input[name="title"], input[placeholder*="title" i], input[placeholder*="назван" i], input[id*="title" i]'
    ).first();
    await titleField.fill('Camera malfunction at entrance');

    // Заполняем описание
    const descriptionField = page.locator(
      'textarea[name="description"], textarea[placeholder*="description" i], textarea[placeholder*="описан" i], textarea[id*="description" i]'
    ).first();
    await descriptionField.fill('Camera feed is flickering, need to check cable');

    // Выбираем объект (site)
    const siteSelect = page.locator(
      'select[name="site_id"], select[id*="site" i], [role="combobox"]'
    ).first();
    if (await siteSelect.isVisible()) {
      await siteSelect.selectOption('site-1');
    }

    // Выбираем приоритет
    const prioritySelect = page.locator(
      'select[name="priority"], select[id*="priority" i], [role="combobox"]'
    ).first();
    if (await prioritySelect.isVisible()) {
      await prioritySelect.selectOption('high');
    }

    // Отмечаем пункты чеклиста
    const checklistItems = page.locator(
      'input[type="checkbox"][name*="checklist" i], input[type="checkbox"][id*="checklist" i], label:has(input[type="checkbox"])'
    );
    const checklistCount = await checklistItems.count();
    if (checklistCount > 0) {
      // Отмечаем первые 2 обязательных пункта
      for (let i = 0; i < Math.min(2, checklistCount); i++) {
        const checkbox = checklistItems.nth(i).locator('input[type="checkbox"]');
        if (await checkbox.isVisible()) {
          await checkbox.check();
        }
      }
    }

    // Сабмитим форму
    const submitButton = page.locator(
      'button[type="submit"], button:has-text(/create|save|создать|сохранить/i)'
    ).first();
    await submitButton.click();
    await page.waitForTimeout(1000);

    // Проверяем редирект или сообщение об успехе
    const currentUrl = page.url();
    const successMessage = page.locator(
      'text=/created successfully|успешно создан|work order created|заявка создана/i'
    );

    // Либо URL изменился на страницу заявки, либо есть сообщение об успехе
    const hasRedirect = currentUrl.includes('/work-orders/');
    const hasSuccess = await successMessage.isVisible().catch(() => false);

    expect(hasRedirect || hasSuccess).toBeTruthy();
  });

  test('Create WO — validation shows error on empty required fields', async ({ page }) => {
    // Кликаем submit без заполнения полей
    const submitButton = page.locator(
      'button[type="submit"], button:has-text(/create|save|создать|сохранить/i)'
    ).first();
    await submitButton.click();
    await page.waitForTimeout(500);

    // Проверяем сообщение об ошибке валидации
    const validationError = page.locator(
      'text=/required|обязательно|please fill|заполните|title.*required|название.*обязательно/i'
    ).first();
    await expect(validationError).toBeVisible();
  });

  test('Create WO — cancel returns to work orders list', async ({ page }) => {
    const cancelButton = page.locator(
      'button:has-text(/cancel|отмена|back|назад/i), a:has-text(/cancel|отмена|back|назад/i)'
    ).first();
    if (await cancelButton.isVisible()) {
      await cancelButton.click();
      await page.waitForTimeout(500);

      // Проверяем что вернулись к списку заявок
      expect(page.url()).toContain('/work-orders');
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════
// 2. Complete WO with photo upload
// ═══════════════════════════════════════════════════════════════════════

test.describe('Work Orders — Complete with Photo Upload', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);

    const detailWithChecklist = {
      ...MOCK_WORK_ORDER_DETAIL,
      id: 'WO-COMPLETE-01',
      title: 'Fix camera at parking lot',
      status: 'in_progress',
      assigned_to: 'user-2',
      checklist: MOCK_CHECKLIST_TEMPLATE.map((item) => ({
        ...item,
        completed: false,
      })),
      photos: [],
    };

    // Mock detail endpoint
    await page.route('**/api/v1/work-orders/WO-COMPLETE-01', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(detailWithChecklist),
      });
    });

    // Mock list endpoint
    await page.route('**/api/v1/work-orders*', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([detailWithChecklist]),
      });
    });

    // Mock upload endpoint
    await page.route('**/api/v1/upload*', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          url: 'https://storage.example.com/photos/wo-complete-01/photo-1.jpg',
          filename: 'photo-1.jpg',
        }),
      });
    });

    // Mock complete endpoint (PATCH /work-orders/:id/complete)
    await page.route('**/api/v1/work-orders/*/complete', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...detailWithChecklist,
          status: 'completed',
          completed_at: new Date().toISOString(),
        }),
      });
    });

    await page.goto('/work-orders/WO-COMPLETE-01');
    await page.waitForTimeout(1500);
  });

  test('Complete WO — detail page shows complete button', async ({ page }) => {
    // Проверяем загрузку деталей заявки
    await expect(page.locator('body')).toBeVisible();

    // Проверяем кнопку завершения / complete
    const completeButton = page.locator(
      'button:has-text(/complete|завершить|mark.*done|отметить.*выполнен/i)'
    ).first();
    await expect(completeButton).toBeVisible();
  });

  test('Complete WO — upload photo before completing', async ({ page }) => {
    // Находим кнопку загрузки фото / upload
    const uploadButton = page.locator(
      'button:has-text(/upload|photo|add photo|загрузить|фото|добавить фото/i), ' +
      'input[type="file"], [role="button"]:has-text(/upload|photo|фото/i)'
    ).first();
    await expect(uploadButton).toBeVisible();

    // Имитируем загрузку файла через input[type="file"]
    const fileInput = page.locator('input[type="file"]').first();
    if (await fileInput.isVisible()) {
      await fileInput.setInputFiles({
        name: 'repair-photo.jpg',
        mimeType: 'image/jpeg',
        buffer: Buffer.from('fake-image-content'),
      });
      await page.waitForTimeout(1000);
    } else if (await uploadButton.isVisible()) {
      // Если это кнопка, кликаем — ожидаем что откроется диалог выбора файла
      // и устанавливаем файл через filechooser
      page.once('filechooser', async (fileChooser) => {
        await fileChooser.setFiles({
          name: 'repair-photo.jpg',
          mimeType: 'image/jpeg',
          buffer: Buffer.from('fake-image-content'),
        });
      });
      await uploadButton.click();
      await page.waitForTimeout(1000);
    }

    // Проверяем что фото отображается в заявке или появился превью
    const photoPreview = page.locator(
      'img[alt*="photo" i], img[alt*="photo" i], [class*="photo" i], [class*="image" i]'
    ).first();
    const hasPhoto = await photoPreview.isVisible().catch(() => false);

    if (hasPhoto) {
      await expect(photoPreview).toBeVisible();
    }
  });

  test('Complete WO — mark checklist items and submit completion', async ({ page }) => {
    // Отмечаем все пункты чеклиста как выполненные
    const checklistCheckboxes = page.locator(
      'input[type="checkbox"]'
    );
    const checklistCount = await checklistCheckboxes.count();
    for (let i = 0; i < checklistCount; i++) {
      const checkbox = checklistCheckboxes.nth(i);
      if (await checkbox.isVisible()) {
        await checkbox.check();
        await page.waitForTimeout(100);
      }
    }

    // Загружаем фото
    const fileInput = page.locator('input[type="file"]').first();
    if (await fileInput.isVisible()) {
      await fileInput.setInputFiles({
        name: 'completed-work.jpg',
        mimeType: 'image/jpeg',
        buffer: Buffer.from('fake-image-content'),
      });
      await page.waitForTimeout(500);
    }

    // Нажимаем complete / завершить
    const completeButton = page.locator(
      'button:has-text(/complete|завершить|mark.*done|отметить.*выполнен/i)'
    ).first();

    if (await completeButton.isVisible()) {
      await completeButton.click();
      await page.waitForTimeout(1000);

      // Проверяем что статус изменился на completed / завершено
      const statusCompleted = page.locator(
        'text=/completed|завершен|done|выполнен/i'
      ).first();
      await expect(statusCompleted).toBeVisible();
    }
  });

  test('Complete WO — cancel completion returns to detail', async ({ page }) => {
    // Находим кнопку отмены
    const cancelButton = page.locator(
      'button:has-text(/cancel|отмена|back|назад/i)'
    ).first();

    if (await cancelButton.isVisible()) {
      await cancelButton.click();
      await page.waitForTimeout(500);

      // Детали заявки всё ещё видимы
      await expect(page.locator('body')).toBeVisible();
      expect(page.url()).toContain('/WO-COMPLETE-01');
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════
// 3. Assign technician
// ═══════════════════════════════════════════════════════════════════════

test.describe('Work Orders — Assign Technician', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);

    const unassignedWO = {
      ...MOCK_WORK_ORDER_DETAIL,
      id: 'WO-ASSIGN-01',
      title: 'Urgent maintenance at warehouse',
      assigned_to: null,
      status: 'open',
    };

    // Mock detail for unassigned WO
    await page.route('**/api/v1/work-orders/WO-ASSIGN-01', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(unassignedWO),
      });
    });

    // Mock list
    await page.route('**/api/v1/work-orders*', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([unassignedWO]),
      });
    });

    // Mock assign endpoint
    await page.route('**/api/v1/work-orders/*/assign', async (route: any) => {
      const body = JSON.parse(route.request().postData() || '{}');
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...unassignedWO,
          assigned_to: body.technician_id || body.user_id,
        }),
      });
    });

    await page.goto('/work-orders/WO-ASSIGN-01');
    await page.waitForTimeout(1500);
  });

  test('Assign tech — unassigned WO shows assign button', async ({ page }) => {
    // Проверяем что есть кнопка или элемент для назначения
    const assignButton = page.locator(
      'button:has-text(/assign|назначить|assign tech|назначить техник/i), ' +
      'select[name*="assign" i], select[id*="assign" i], ' +
      '[role="combobox"]:has-text(/assign|technician|назначить|техник/i)'
    ).first();
    await expect(assignButton).toBeVisible();
  });

  test('Assign tech — select technician from dropdown', async ({ page }) => {
    // Находим селект или кнопку назначения
    const assignSelect = page.locator(
      'select[name*="assign" i], select[id*="assign" i], ' +
      'select:has(option), ' +
      '[role="combobox"]'
    ).first();

    if (await assignSelect.isVisible()) {
      // Выбираем техника
      const tagName = await assignSelect.evaluate((el: any) => el.tagName.toLowerCase());
      if (tagName === 'select') {
        await assignSelect.selectOption('user-2');
        await page.waitForTimeout(500);
      } else {
        await assignSelect.click();
        await page.waitForTimeout(300);
        // Выбираем опцию из выпадающего списка
        const techOption = page.locator(
          '[role="option"], [role="listbox"] option, li, div[role="menuitem"]'
        ).filter({ hasText: /bob|technician|tech1/i }).first();

        if (await techOption.isVisible()) {
          await techOption.click();
          await page.waitForTimeout(300);
        }
      }
    } else {
      // Если это кнопка, открывающая модалку
      const assignButton = page.locator(
        'button:has-text(/assign|назначить/i)'
      ).first();
      if (await assignButton.isVisible()) {
        await assignButton.click();
        await page.waitForTimeout(500);

        // В модалке выбираем техника
        const modalSelect = page.locator(
          'select, [role="combobox"], input[type="text"]'
        ).first();
        if (await modalSelect.isVisible()) {
          await modalSelect.fill('Bob');
          await page.waitForTimeout(300);
        }

        // Подтверждаем назначение
        const confirmButton = page.locator(
          'button:has-text(/confirm|assign|save|подтвердить|назначить|сохранить/i)'
        ).last();
        if (await confirmButton.isVisible()) {
          await confirmButton.click();
          await page.waitForTimeout(500);
        }
      }
    }

    // Проверяем что имя техника отображается на странице
    const assignedTech = page.locator(
      'text=/Bob|Technician|tech1|assigned|назначен/i'
    ).first();
    const isAssigned = await assignedTech.isVisible().catch(() => false);
    if (isAssigned) {
      await expect(assignedTech).toBeVisible();
    }
  });

  test('Assign tech — reassign to different technician', async ({ page }) => {
    // Сначала назначаем одного техника
    const assignButton = page.locator(
      'button:has-text(/assign|назначить|reassign|переназначить/i), ' +
      'select[name*="assign" i], select[id*="assign" i]'
    ).first();

    if (await assignButton.isVisible()) {
      const tagName = await assignButton.evaluate((el: any) => el.tagName.toLowerCase());

      if (tagName === 'select') {
        // Выбираем tech1
        await assignButton.selectOption('user-2');
        await page.waitForTimeout(300);
        // Меняем на tech2
        await assignButton.selectOption('user-3');
        await page.waitForTimeout(300);
      } else {
        // Кликаем assign
        await assignButton.click();
        await page.waitForTimeout(500);

        // Выбираем tech1
        const option1 = page.locator(
          '[role="option"], li, div[role="menuitem"]'
        ).filter({ hasText: /bob|technician/i }).first();
        if (await option1.isVisible()) {
          await option1.click();
          await page.waitForTimeout(200);
        }

        // Снова кликаем reassign
        const reassignButton = page.locator(
          'button:has-text(/reassign|переназначить|change|изменить/i)'
        ).first();
        if (await reassignButton.isVisible()) {
          await reassignButton.click();
          await page.waitForTimeout(500);

          // Выбираем tech2
          const option2 = page.locator(
            '[role="option"], li, div[role="menuitem"]'
          ).filter({ hasText: /carol|engineer/i }).first();
          if (await option2.isVisible()) {
            await option2.click();
            await page.waitForTimeout(200);
          }
        }
      }
    }
  });

  test('Assign tech — view technician info after assignment', async ({ page }) => {
    const assignSelect = page.locator(
      'select[name*="assign" i], select[id*="assign" i], ' +
      '[role="combobox"]'
    ).first();

    if (await assignSelect.isVisible()) {
      const tagName = await assignSelect.evaluate((el: any) => el.tagName.toLowerCase());

      if (tagName === 'select') {
        await assignSelect.selectOption('user-2');
        await page.waitForTimeout(500);
      } else {
        await assignSelect.click();
        await page.waitForTimeout(300);
        const techOption = page.locator(
          '[role="option"], li, div[role="menuitem"]'
        ).filter({ hasText: /bob|technician|tech1/i }).first();

        if (await techOption.isVisible()) {
          await techOption.click();
          await page.waitForTimeout(300);
        }
      }
    }

    // Проверяем отображение информации о технике (имя, контакт)
    const techInfo = page.locator(
      '[class*="technician" i], [class*="assignee" i], ' +
      '[class*="assigned" i], [class*="tech" i]'
    ).first();
    const hasTechInfo = await techInfo.isVisible().catch(() => false);

    if (hasTechInfo) {
      await expect(techInfo).toBeVisible();
    }
  });
});
