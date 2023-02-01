import { reactive, watch } from "vue";

const SETTINGS_LOCALE = "settings_locale";
const SETTINGS_THEME = "settings_theme";
const SETTINGS_UNIT = "settings_unit";
const SETTINGS_ENERGYFLOW_DETAILS = "settings_energyflow_details";
const SETTINGS_GRID_DETAILS = "settings_grid_details";

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

function readBool(key) {
  return read(key) === "true";
}

function saveBool(key) {
  return (value) => {
    save(key)(value ? "true" : "false");
  };
}

const settings = reactive({
  telemetry: null,
  locale: read(SETTINGS_LOCALE),
  theme: read(SETTINGS_THEME),
  unit: read(SETTINGS_UNIT),
  gridDetails: read(SETTINGS_GRID_DETAILS),
  energyflowDetails: readBool(SETTINGS_ENERGYFLOW_DETAILS),
});

watch(() => settings.locale, save(SETTINGS_LOCALE));
watch(() => settings.theme, save(SETTINGS_THEME));
watch(() => settings.unit, save(SETTINGS_UNIT));
watch(() => settings.gridDetails, save(SETTINGS_GRID_DETAILS));
watch(() => settings.energyflowDetails, saveBool(SETTINGS_ENERGYFLOW_DETAILS));

export default settings;
