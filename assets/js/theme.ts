import settings from "./settings";
import { THEME } from "./types/evcc";

const darkModeMatcher = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)");

export function getThemePreference(): THEME | null {
  const theme = settings.theme;
  if (theme && Object.values(THEME).includes(theme)) {
    return theme;
  }
  return THEME.AUTO;
}

export function setThemePreference(theme: THEME) {
  if (!Object.values(THEME).includes(theme)) {
    return;
  }
  settings.theme = theme;
  updateTheme();
}

function setMetaThemeColor(theme: Exclude<THEME, THEME.AUTO> | null) {
  const themeColors = { light: "#f3f3f7", dark: "#020318" };
  const $metaThemeColor = document.querySelector("meta[name=theme-color]");
  if ($metaThemeColor && theme) {
    $metaThemeColor.setAttribute("content", themeColors[theme]);
  }
}

function getCurrentTheme() {
  let theme = getThemePreference();
  if (theme === THEME.AUTO) {
    theme = darkModeMatcher?.matches ? THEME.DARK : THEME.LIGHT;
  }
  return theme;
}

function updateTheme() {
  const theme = getCurrentTheme();

  // update iOS title bar color
  setMetaThemeColor(theme);

  // toggle the class on html root
  const $html = document.querySelector("html");
  if ($html) {
    $html.classList.add("no-transitions");
    $html.classList.toggle("dark", theme === THEME.DARK);
    window.setTimeout(function () {
      $html.classList.remove("no-transitions");
    }, 100);
  }
}

function updateMetaThemeForBackdrop() {
  const $backdrop = document.querySelector("[data-bs-backdrop=true][aria-modal=true]");
  // dark if there is a backdrop, otherwise use the current theme
  const theme = $backdrop ? THEME.DARK : getCurrentTheme();
  setMetaThemeColor(theme);
}

export function watchThemeChanges() {
  if (darkModeMatcher && darkModeMatcher.addEventListener) {
    darkModeMatcher.addEventListener("change", updateTheme);
  }
  updateTheme();

  // listen for modal backdrops
  document.addEventListener("shown.bs.modal", updateMetaThemeForBackdrop);
  document.addEventListener("hidden.bs.modal", updateMetaThemeForBackdrop);
}
