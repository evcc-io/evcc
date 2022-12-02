import toml from "toml";
import { nextTick } from "vue";
import { createI18n } from "vue-i18n";
import en from "../../i18n/en.toml";
import { i18n as i18nApi } from "./api";

const PREFERRED_LOCALE_KEY = "preferred_locale";

function getBrowserLocale() {
  const navigatorLocale =
    navigator.languages !== undefined ? navigator.languages[0] : navigator.language;
  if (!navigatorLocale) {
    return undefined;
  }
  return navigatorLocale.trim().split(/-|_/)[0];
}

function getLocale() {
  return window.localStorage[PREFERRED_LOCALE_KEY] || getBrowserLocale();
}

export default function setupI18n() {
  const i18n = createI18n({
    legacy: true,
    silentFallbackWarn: true,
    silentTranslationWarn: true,
    locale: "en",
    fallbackLocale: "en",
    messages: { en },
  });
  setI18nLanguage(i18n, getLocale());
  return i18n;
}

export function setI18nLanguage(i18n, locale) {
  i18n.global.locale = locale;
  document.querySelector("html").setAttribute("lang", locale);
}

async function loadLocaleMessages(i18n, locale) {
  try {
    const response = await i18nApi.get(`${locale}.toml`, { params: { v: window.evcc?.version } });
    const messages = toml.parse(response.data);
    i18n.global.setLocaleMessage(locale, messages);
  } catch (e) {
    console.error(`unable to load translation for [${locale}]`, e);
  }

  return nextTick();
}

export async function ensureCurrentLocaleMessages(i18n) {
  const { locale } = i18n.global;
  if (!i18n.global.availableLocales.includes(locale)) {
    await loadLocaleMessages(i18n, locale);
  }
}
