import Vue from "vue";
import VueI18n from "vue-i18n";
import de from "./de";
import en from "./en";
import it from "./it";

Vue.use(VueI18n);

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
  locale: getBrowserLocale(),
  fallbackLocale: "en",
  messages: { de, en, it },
});
