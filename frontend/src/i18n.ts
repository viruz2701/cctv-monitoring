import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

const DEFAULT_LANGUAGES = ['en', 'ru', 'be'];

// Все языки загружаются динамически для code-splitting (P2-OPT.3)
// Статические импорты удалены — они предотвращали корректный code-split
async function loadDefaultLanguages() {
  const [en, ru, be] = await Promise.all([
    import('./locales/en/translation.json'),
    import('./locales/ru/translation.json'),
    import('./locales/be/translation.json'),
  ]);

  return {
    en: { translation: en.default || en },
    ru: { translation: ru.default || ru },
    be: { translation: be.default || be },
  };
}

// Инициализируем i18n с динамической загрузкой
loadDefaultLanguages().then((resources) => {
  i18n.use(initReactI18next).init({
    resources,
    lng: 'ru',
    fallbackLng: 'ru',
    interpolation: { escapeValue: false },
  });
});

// Lazy-load non-default languages on switch
i18n.on('languageChanged', async (lng) => {
  if (DEFAULT_LANGUAGES.includes(lng)) return;
  if (i18n.hasResourceBundle(lng, 'translation')) return;

  try {
    const mod = await import(`./locales/${lng}/translation.json`);
    i18n.addResourceBundle(lng, 'translation', mod.default || mod);
  } catch (err) {
    console.warn(`[i18n] Failed to load locale: ${lng}`, err);
  }
});

export default i18n;
