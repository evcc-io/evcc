import toml from "toml";
import { nextTick } from "vue";
import { createI18n } from "vue-i18n";
import en from "../../i18n/en.toml";
import { i18n as i18nApi } from "./api";

const PREFERRED_LOCALE_KEY = "preferred_locale";

// https://github.com/joker-x/languages.js/blob/master/languages.json
export const LOCALES = {
  nl: ["Dutch", "Nederlands"],
  en: ["English", "English"],
  fr: ["French", "Français"],
  de: ["German", "Deutsch"],
  it: ["Italian", "Italiano"],
  lt: ["Lithuanian", "Lietuvių"],
  no: ["Norwegian", "Norsk"],
  pl: ["Polish", "Polski"],
  pt: ["Portuguese", "Português"],
  es: ["Spanish", "Español"],
};

function getBrowserLocale() {
  const navigatorLocale =
    navigator.languages !== undefined ? navigator.languages[0] : navigator.language;
  if (!navigatorLocale) {
    return undefined;
  }
  return navigatorLocale.trim().split(/-|_/)[0];
}

export function getLocalePreference() {
  return window.localStorage[PREFERRED_LOCALE_KEY];
}

export function removeLocalePreference(i18n) {
  try {
    delete window.localStorage[PREFERRED_LOCALE_KEY];
    setI18nLanguage(i18n, i18n.fallbackLocale);
  } catch (e) {
    console.error("unable to delete locale in localStorage", e);
  }
}

export function setLocalePreference(i18n, locale) {
  if (!LOCALES[locale]) {
    console.error("unknown locale", locale);
    return;
  }
  try {
    window.localStorage[PREFERRED_LOCALE_KEY] = locale;
    setI18nLanguage(i18n, locale);
    ensureCurrentLocaleMessages(i18n);
  } catch (e) {
    console.error("unable to write locale to localStorage", e);
  }
}

function getLocale() {
  return getLocalePreference() || getBrowserLocale();
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
  setI18nLanguage(i18n.global, getLocale());
  return i18n;
}

export function setI18nLanguage(i18n, locale) {
  i18n.locale = locale;
  document.querySelector("html").setAttribute("lang", locale);
}

async function loadLocaleMessages(i18n, locale) {
  try {
    const response = await i18nApi.get(`${locale}.toml`, { params: { v: window.evcc?.version } });
    const messages = toml.parse(response.data);
    i18n.setLocaleMessage(locale, messages);
  } catch (e) {
    console.error(`unable to load translation for [${locale}]`, e);
  }

  return nextTick();
}

export async function ensureCurrentLocaleMessages(i18n) {
  const { locale } = i18n;
  if (!i18n.availableLocales.includes(locale)) {
    await loadLocaleMessages(i18n, locale);
  }
}
