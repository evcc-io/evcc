const darkModeMatcher = window.matchMedia("(prefers-color-scheme: dark)");

const THEME_AUTO = "auto";
const THEME_LIGHT = "light";
const THEME_DARK = "dark";
export const THEMES = [THEME_AUTO, THEME_LIGHT, THEME_DARK];
const LOCAL_STORAGE_KEY = "theme";

export function getThemePreference() {
  try {
    const theme = window.localStorage[LOCAL_STORAGE_KEY];
    if (THEMES.includes(theme)) {
      return theme;
    }
  } catch (e) {
    console.error("unable to read theme from localStorage", e);
  }
  return THEME_AUTO;
}

export function setThemePreference(theme) {
  console.log({ theme });
  if (!THEMES.includes(theme)) {
    return;
  }
  try {
    window.localStorage[LOCAL_STORAGE_KEY] = theme;
    updateTheme();
  } catch (e) {
    console.error("unable to write theme to localStorage", e);
  }
}

function updateTheme() {
  let theme = getThemePreference();

  if (theme === THEME_AUTO) {
    theme = darkModeMatcher.matches ? THEME_DARK : THEME_LIGHT;
  }

  // update iOS title bar color
  const themeColors = { light: "#f3f3f7", dark: "#020318" };
  const $metaThemeColor = document.querySelector("meta[name=theme-color]");
  if ($metaThemeColor) {
    $metaThemeColor.setAttribute("content", themeColors[theme]);
  }

  // toggle the class on html root
  const $html = document.querySelector("html");
  $html.classList.add("no-transitions");
  $html.classList.toggle("dark", theme === THEME_DARK);
  window.setTimeout(function () {
    $html.classList.remove("no-transitions");
  }, 100);
}

export function watchThemeChanges() {
  darkModeMatcher.addEventListener("change", updateTheme);
  updateTheme();
}
