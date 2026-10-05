import settings from "./settings";
import { LOCALES } from "./i18n";
import { LENGTH_UNIT, THEME, TIME_FORMAT } from "./types/evcc";

export function applyUrlSettings() {
  const url = new URL(window.location.href);
  const params = url.searchParams;
  let touched = false;

  const theme = params.get("theme");
  if (theme && Object.values(THEME).includes(theme as THEME)) {
    settings.theme = theme as THEME;
    touched = true;
  }

  const lang = params.get("lang");
  if (lang === "auto") {
    settings.locale = null;
    touched = true;
  } else if (lang && lang in LOCALES) {
    settings.locale = lang as keyof typeof LOCALES;
    touched = true;
  }

  const unit = params.get("unit");
  if (unit && Object.values(LENGTH_UNIT).includes(unit as LENGTH_UNIT)) {
    settings.unit = unit as LENGTH_UNIT;
    touched = true;
  }

  const format = params.get("format");
  if (format && Object.values(TIME_FORMAT).includes(format as TIME_FORMAT)) {
    settings.is12hFormat = format === TIME_FORMAT.H12;
    touched = true;
  }

  if (touched) {
    ["theme", "lang", "unit", "format"].forEach((k) => params.delete(k));
    window.history.replaceState(window.history.state, "", url.pathname + url.search + url.hash);
  }
}
