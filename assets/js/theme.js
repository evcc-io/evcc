import settings from "./settings";

const darkModeMatcher = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)");

const THEME_AUTO = "auto";
const THEME_LIGHT = "light";
const THEME_DARK = "dark";
export const THEMES = [THEME_AUTO, THEME_LIGHT, THEME_DARK];

export function getThemePreference() {
  const theme = settings.theme;
  if (THEMES.includes(theme)) {
    return theme;
  }
  return THEME_AUTO;
}

export function setThemePreference(theme) {
  if (!THEMES.includes(theme)) {
    return;
  }
  settings.theme = theme;
  updateTheme();
}

function setMetaThemeColor(theme) {
  const themeColors = { light: "#f3f3f7", dark: "#020318" };
  const $metaThemeColor = document.querySelector("meta[name=theme-color]");
  if ($metaThemeColor) {
    $metaThemeColor.setAttribute("content", themeColors[theme]);
  }
}

function getCurrentTheme() {
  let theme = getThemePreference();
  if (theme === THEME_AUTO) {
    theme = darkModeMatcher?.matches ? THEME_DARK : THEME_LIGHT;
  }
  return theme;
}

function updateTheme() {
  const theme = getCurrentTheme();

  // update iOS title bar color
  setMetaThemeColor(theme);

  // toggle the class on html root
  const $html = document.querySelector("html");
  $html.classList.add("no-transitions");
  $html.classList.toggle("dark", theme === THEME_DARK);
  window.setTimeout(function () {
    $html.classList.remove("no-transitions");
  }, 100);
}

function updateMetaThemeForBackdrop() {
  const $backdrop = document.querySelector("[data-bs-backdrop=true][aria-modal=true]");
  // dark if there is a backdrop, otherwise use the current theme
  const theme = $backdrop ? THEME_DARK : getCurrentTheme();
  console.log("backdrop", theme);
  setMetaThemeColor(theme);
}

export function watchThemeChanges() {
  darkModeMatcher?.addEventListener("change", updateTheme);
  updateTheme();

  // listen for modal backdrops
  document.addEventListener("shown.bs.modal", updateMetaThemeForBackdrop);
  document.addEventListener("hidden.bs.modal", updateMetaThemeForBackdrop);
}
