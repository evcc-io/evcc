import { createI18n } from "vue-i18n";
import de from "./de";
import en from "./en";
import it from "./it";
import lt from "./lt";

const PREFERRED_LOCALE_KEY = "preferred_locale";

function getBrowserLocale() {
  const navigatorLocale =
    navigator.languages !== undefined ? navigator.languages[0] : navigator.language;
  if (!navigatorLocale) {
    return undefined;
  }
  return navigatorLocale.trim().split(/-|_/)[0];
}

export default createI18n({
  locale: window.localStorage[PREFERRED_LOCALE_KEY] || getBrowserLocale(),
  fallbackLocale: "en",
  messages: { de, en, it, lt },
});
