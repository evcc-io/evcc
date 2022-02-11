import { createI18n } from "vue-i18n";
import de from "./de";
import en from "./en";
import it from "./it";

function getBrowserLocale() {
  const navigatorLocale =
    navigator.languages !== undefined ? navigator.languages[0] : navigator.language;
  if (!navigatorLocale) {
    return undefined;
  }
  const trimmedLocale = navigatorLocale.trim().split(/-|_/)[0];
  console.log(trimmedLocale);
  return trimmedLocale;
}

export default createI18n({
  locale: getBrowserLocale(),
  fallbackLocale: "en",
  messages: { de, en, it },
});
