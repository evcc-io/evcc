import { reactive, watch } from "vue";

const SETTINGS_LOCALE = "settings_locale";
const SETTINGS_THEME = "settings_theme";
const SETTINGS_UNIT = "settings_unit";
const SETTINGS_HIDDEN_FEATURES = "settings_hidden_features";
const SETTINGS_ENERGYFLOW_DETAILS = "settings_energyflow_details";
const SESSION_INFO = "session_info";

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

function readArray(key) {
  return (read(key) || "").split(",");
}

function saveArray(key) {
  return (value) => {
    save(key)(value.join(","));
  };
}

const settings = reactive({
  telemetry: null,
  locale: read(SETTINGS_LOCALE),
  theme: read(SETTINGS_THEME),
  unit: read(SETTINGS_UNIT),
  hiddenFeatures: readBool(SETTINGS_HIDDEN_FEATURES),
  energyflowDetails: readBool(SETTINGS_ENERGYFLOW_DETAILS),
  sessionInfo: readArray(SESSION_INFO),
});

watch(() => settings.locale, save(SETTINGS_LOCALE));
watch(() => settings.theme, save(SETTINGS_THEME));
watch(() => settings.unit, save(SETTINGS_UNIT));
watch(() => settings.hiddenFeatures, saveBool(SETTINGS_HIDDEN_FEATURES));
watch(() => settings.energyflowDetails, saveBool(SETTINGS_ENERGYFLOW_DETAILS));
watch(() => settings.sessionInfo, saveArray(SESSION_INFO));

export default settings;
