import { createI18n } from "vue-i18n";
import de from "../i18n/de.toml";
import en from "../i18n/en.toml";
import fr from "../i18n/fr.toml";
import it from "../i18n/it.toml";
import lt from "../i18n/lt.toml";
import nl from "../i18n/nl.toml";
import nb_no from "../i18n/nb_NO.toml";
import pl from "../i18n/pl.toml";
import pt from "../i18n/pt.toml";

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
  messages: { de, en, fr, it, lt, nb_no, nl, pl, pt },
});
