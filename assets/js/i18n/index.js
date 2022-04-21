import Vue from "vue";
import VueI18n from "vue-i18n";
import de from "./de";
import en from "./en";
import it from "./it";
import lt from "./lt";

Vue.use(VueI18n);

const PREFERRED_LOCALE_KEY = "preferred_locale";

function getBrowserLocale() {
  const navigatorLocale =
    navigator.languages !== undefined ? navigator.languages[0] : navigator.language;
  if (!navigatorLocale) {
    return undefined;
  }
  const trimmedLocale = navigatorLocale.trim().split(/-|_/)[0];
  return trimmedLocale;
}

export default new VueI18n({
  locale: window.localStorage[PREFERRED_LOCALE_KEY] || getBrowserLocale(),
  fallbackLocale: "en",
  messages: { de, en, it, lt },
});
