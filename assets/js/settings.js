import { reactive, watch } from "vue";

const SETTING_LOCALE = "setting_locale";
const SETTING_THEME = "setting_theme";
const SETTING_UNIT = "setting_unit";

function read(key) {
  return window.localStorage[key];
}

function save(key) {
  return (value) => {
    try {
      if (value) {
        window.localStorage[key] = value;
      } else {
        delete window.localStorage[key];
      }
    } catch (e) {
      console.error("unable to write to localstorage", { key, value });
    }
  };
}

const settings = reactive({
  telemetry: null,
  locale: read(SETTING_LOCALE),
  theme: read(SETTING_THEME),
  unit: read(SETTING_UNIT),
});

watch(() => settings.locale, save(SETTING_LOCALE));
watch(() => settings.theme, save(SETTING_THEME));
watch(() => settings.unit, save(SETTING_UNIT));

export default settings;
