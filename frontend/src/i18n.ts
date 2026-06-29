import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import en from './locales/en/translation.json';
import ru from './locales/ru/translation.json';
import be from './locales/be/translation.json';

const DEFAULT_LANGUAGES = ['en', 'ru', 'be'];

const resources: Record<string, { translation: Record<string, string> }> = {
  en: { translation: en },
  ru: { translation: ru },
  be: { translation: be },
};

i18n
  .use(initReactI18next)
  .init({
    resources,
    lng: 'ru',
    fallbackLng: 'ru',
    interpolation: { escapeValue: false },
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
