const ru = {
  common: {
    loading: 'Загрузка...',
    error: 'Ошибка',
    retry: 'Повторить',
    cancel: 'Отмена',
    save: 'Сохранить',
    delete: 'Удалить',
    confirm: 'Подтвердить',
    back: 'Назад',
    close: 'Закрыть',
    done: 'Готово',
    continue: 'Продолжить',
    skip: 'Пропустить',
    submit: 'Отправить',
    yes: 'Да',
    no: 'Нет',
    offline: 'Нет соединения',
    online: 'В сети',
    syncPending: 'Ожидает синхронизации',
    syncing: 'Синхронизация...',
    synced: 'Синхронизировано',
    required: 'Обязательное поле',
  },
  auth: {
    title: 'CCTV Health Monitor',
    subtitle: 'Войдите в аккаунт техника',
    username: 'Имя пользователя',
    usernamePlaceholder: 'Введите имя пользователя',
    password: 'Пароль',
    passwordPlaceholder: 'Введите пароль',
    login: 'Войти',
    loggingIn: 'Вход...',
    invalidCredentials: 'Неверное имя пользователя или пароль',
    networkError: 'Ошибка сети. Проверьте подключение.',
  },
  dashboard: {
    title: 'Наряды',
    greeting: {
      morning: 'Доброе утро',
      afternoon: 'Добрый день',
      evening: 'Добрый вечер',
      night: 'Доброй ночи',
    },
    stats: {
      open: 'Открытые',
      inProgress: 'В работе',
      completed: 'Завершены',
    },
    empty: 'Нет активных нарядов',
    emptySubtext: 'Новые наряды появятся здесь после назначения',
    pullToRefresh: 'Потяните для обновления',
  },
  workOrder: {
    title: 'Наряд',
    device: 'Устройство',
    site: 'Объект',
    type: 'Тип',
    priority: 'Приоритет',
    status: 'Статус',
    slaDeadline: 'Срок SLA',
    assignedTo: 'Назначен',
    notes: 'Заметки',
    photos: 'Фото',
    partsUsed: 'Запчасти',
    createdAt: 'Создан',
    updatedAt: 'Обновлён',
    startedAt: 'Начат',
    completedAt: 'Завершён',
    checklist: 'Чек-лист',
    signature: 'Подпись',
    location: 'Местоположение',
    actions: {
      start: 'Начать работу',
      complete: 'Завершить',
      checklist: 'Чек-лист',
      addPhoto: 'Добавить фото',
      scanQR: 'Сканировать QR',
      signature: 'Подпись',
    },
    statuses: {
      open: 'Открыт',
      in_progress: 'В работе',
      completed: 'Завершён',
      cancelled: 'Отменён',
    },
    priorities: {
      critical: 'Критический',
      high: 'Высокий',
      medium: 'Средний',
      low: 'Низкий',
    },
    types: {
      preventive: 'Плановое ТО',
      corrective: 'Корректирующее',
      emergency: 'Аварийное',
    },
    sla: {
      onTrack: 'В срок',
      atRisk: 'Под угрозой',
      breached: 'Просрочен',
    },
    completed: 'Наряд завершён',
    completedMessage: 'Все работы выполнены успешно',
  },
  checklist: {
    title: 'Чек-лист',
    progress: '{{completed}} из {{total}} выполнено',
    completeAll: 'Выполните все пункты для продолжения',
    allDone: 'Все пункты выполнены!',
  },
  photo: {
    title: 'Фото',
    takePhoto: 'Сделать снимок',
    chooseFromGallery: 'Выбрать из галереи',
    permissionDenied: 'Доступ к камере запрещён',
    uploadError: 'Ошибка загрузки фото',
    uploading: 'Загрузка...',
    location: 'Место съёмки',
    noPhotos: 'Нет фотографий',
    addFirst: 'Добавьте первое фото',
  },
  signature: {
    title: 'Подпись',
    drawHere: 'Распишитесь здесь',
    clear: 'Очистить',
    notes: 'Заметки к наряду',
    notesPlaceholder: 'Опишите выполненные работы, проблемы, замечания...',
    summary: 'Проверка',
    checklistCompleted: 'Чек-лист: {{count}} пунктов',
    photosAttached: 'Фото: {{count}} шт.',
    partsUsed: 'Запчасти: {{count}} шт.',
    locationRecorded: 'Местоположение записано',
    confirmSubmit: 'Отправить наряд?',
    confirmMessage: 'После отправки наряд будет помечен как завершённый.',
    submitError: 'Ошибка отправки',
    savedOffline: 'Сохранено офлайн. Отправится при подключении к сети.',
  },
  qr: {
    title: 'Сканер QR',
    hint: 'Наведите камеру на QR-код устройства',
    permissionDenied: 'Доступ к камере запрещён',
    scanned: 'QR-код распознан',
    deviceFound: 'Устройство: {{name}}',
    invalidQR: 'Недействительный QR-код',
    rescan: 'Сканировать снова',
  },
  profile: {
    title: 'Профиль',
    stats: 'Статистика за месяц',
    completedThisMonth: 'Выполнено',
    totalWorkOrders: 'Всего нарядов',
    onTimePercent: 'В срок',
    avgRating: 'Средний балл',
    skills: 'Навыки',
    noSkills: 'Навыки не указаны',
    baseLocation: 'Базовая локация',
    logout: 'Выйти',
    logoutConfirm: 'Выйти из аккаунта?',
    logoutMessage: 'Офлайн-данные будут сохранены.',
  },
  tabs: {
    dashboard: 'Наряды',
    profile: 'Профиль',
  },
  offline: {
    indicator: 'Офлайн',
    pendingCount: '{{count}} в очереди',
    syncComplete: 'Синхронизация завершена',
    syncFailed: 'Ошибка синхронизации',
  },
};

const en: typeof ru = {
  common: {
    loading: 'Loading...',
    error: 'Error',
    retry: 'Retry',
    cancel: 'Cancel',
    save: 'Save',
    delete: 'Delete',
    confirm: 'Confirm',
    back: 'Back',
    close: 'Close',
    done: 'Done',
    continue: 'Continue',
    skip: 'Skip',
    submit: 'Submit',
    yes: 'Yes',
    no: 'No',
    offline: 'No connection',
    online: 'Online',
    syncPending: 'Pending sync',
    syncing: 'Syncing...',
    synced: 'Synced',
    required: 'Required field',
  },
  auth: {
    title: 'CCTV Health Monitor',
    subtitle: 'Sign in to technician account',
    username: 'Username',
    usernamePlaceholder: 'Enter username',
    password: 'Password',
    passwordPlaceholder: 'Enter password',
    login: 'Sign In',
    loggingIn: 'Signing in...',
    invalidCredentials: 'Invalid username or password',
    networkError: 'Network error. Check your connection.',
  },
  dashboard: {
    title: 'Work Orders',
    greeting: {
      morning: 'Good morning',
      afternoon: 'Good afternoon',
      evening: 'Good evening',
      night: 'Good night',
    },
    stats: {
      open: 'Open',
      inProgress: 'In Progress',
      completed: 'Completed',
    },
    empty: 'No active work orders',
    emptySubtext: 'New work orders will appear here once assigned',
    pullToRefresh: 'Pull to refresh',
  },
  workOrder: {
    title: 'Work Order',
    device: 'Device',
    site: 'Site',
    type: 'Type',
    priority: 'Priority',
    status: 'Status',
    slaDeadline: 'SLA Deadline',
    assignedTo: 'Assigned to',
    notes: 'Notes',
    photos: 'Photos',
    partsUsed: 'Parts Used',
    createdAt: 'Created',
    updatedAt: 'Updated',
    startedAt: 'Started',
    completedAt: 'Completed',
    checklist: 'Checklist',
    signature: 'Signature',
    location: 'Location',
    actions: {
      start: 'Start Work',
      complete: 'Complete',
      checklist: 'Checklist',
      addPhoto: 'Add Photo',
      scanQR: 'Scan QR',
      signature: 'Signature',
    },
    statuses: {
      open: 'Open',
      in_progress: 'In Progress',
      completed: 'Completed',
      cancelled: 'Cancelled',
    },
    priorities: {
      critical: 'Critical',
      high: 'High',
      medium: 'Medium',
      low: 'Low',
    },
    types: {
      preventive: 'Preventive',
      corrective: 'Corrective',
      emergency: 'Emergency',
    },
    sla: {
      onTrack: 'On Track',
      atRisk: 'At Risk',
      breached: 'Breached',
    },
    completed: 'Work Order Completed',
    completedMessage: 'All work has been completed successfully',
  },
  checklist: {
    title: 'Checklist',
    progress: '{{completed}} of {{total}} completed',
    completeAll: 'Complete all items to continue',
    allDone: 'All items completed!',
  },
  photo: {
    title: 'Photos',
    takePhoto: 'Take Photo',
    chooseFromGallery: 'Choose from Gallery',
    permissionDenied: 'Camera permission denied',
    uploadError: 'Photo upload failed',
    uploading: 'Uploading...',
    location: 'Photo location',
    noPhotos: 'No photos',
    addFirst: 'Add your first photo',
  },
  signature: {
    title: 'Signature',
    drawHere: 'Sign here',
    clear: 'Clear',
    notes: 'Work Order Notes',
    notesPlaceholder: 'Describe completed work, issues, remarks...',
    summary: 'Review',
    checklistCompleted: 'Checklist: {{count}} items',
    photosAttached: 'Photos: {{count}}',
    partsUsed: 'Parts: {{count}}',
    locationRecorded: 'Location recorded',
    confirmSubmit: 'Submit work order?',
    confirmMessage: 'After submission, the work order will be marked as completed.',
    submitError: 'Submit error',
    savedOffline: 'Saved offline. Will be sent when connection is restored.',
  },
  qr: {
    title: 'QR Scanner',
    hint: 'Point camera at device QR code',
    permissionDenied: 'Camera permission denied',
    scanned: 'QR code scanned',
    deviceFound: 'Device: {{name}}',
    invalidQR: 'Invalid QR code',
    rescan: 'Scan Again',
  },
  profile: {
    title: 'Profile',
    stats: 'Monthly Statistics',
    completedThisMonth: 'Completed',
    totalWorkOrders: 'Total Orders',
    onTimePercent: 'On Time',
    avgRating: 'Avg Rating',
    skills: 'Skills',
    noSkills: 'No skills listed',
    baseLocation: 'Base Location',
    logout: 'Sign Out',
    logoutConfirm: 'Sign out of your account?',
    logoutMessage: 'Offline data will be preserved.',
  },
  tabs: {
    dashboard: 'Orders',
    profile: 'Profile',
  },
  offline: {
    indicator: 'Offline',
    pendingCount: '{{count}} in queue',
    syncComplete: 'Sync complete',
    syncFailed: 'Sync failed',
  },
};

export type Locale = typeof ru;
export type LocaleKey = keyof Locale;

const translations: Record<string, Locale> = { ru, en };

let currentLocale = 'ru';

export function setLocale(locale: string): void {
  if (translations[locale]) {
    currentLocale = locale;
  }
}

export function getLocale(): string {
  return currentLocale;
}

export function t<K extends keyof Locale>(section: K): Locale[K] {
  return translations[currentLocale]?.[section] ?? ru[section];
}

export function tx(path: string, params?: Record<string, string | number>): string {
  const keys = path.split('.');
  let value: unknown = translations[currentLocale] ?? ru;

  for (const key of keys) {
    if (value && typeof value === 'object' && key in value) {
      value = (value as Record<string, unknown>)[key];
    } else {
      return path;
    }
  }

  if (typeof value !== 'string') {
    return path;
  }

  let result = value as string;
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      result = result.replace(`{{${k}}}`, String(v));
    }
  }
  return result;
}

export default translations;