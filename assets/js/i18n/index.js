import { createI18n } from "vue-i18n/index";
import de from "./de";
import en from "./en";
import it from "./it";

function getBrowserLocale() {
  const navigatorLocale =
    navigator.languages !== undefined ? navigator.languages[0] : navigator.language;
  if (!navigatorLocale) {
    return undefined;
  }
  return navigatorLocale.trim().split(/-|_/)[0];
}

export default createI18n({
  locale: getBrowserLocale(),
  fallbackLocale: "en",
  messages: { de, en, it },
});
