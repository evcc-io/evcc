import { reactive, watch } from "vue";
import type { THEME, SessionInfoKey } from "./types/evcc";
import type { LOCALES } from "./i18n";

const SETTINGS_LOCALE = "settings_locale";
const SETTINGS_THEME = "settings_theme";
const SETTINGS_UNIT = "settings_unit";
const SETTINGS_12H_FORMAT = "settings_12h_format";
const SETTINGS_HIDDEN_FEATURES = "settings_hidden_features";
const SETTINGS_ENERGYFLOW_DETAILS = "settings_energyflow_details";
const SETTINGS_ENERGYFLOW_PV = "settings_energyflow_pv";
const SETTINGS_ENERGYFLOW_BATTERY = "settings_energyflow_battery";
const SETTINGS_ENERGYFLOW_LOADPOINTS = "settings_energyflow_loadpoints";
const SETTINGS_ENERGYFLOW_CONSUMERS = "settings_energyflow_consumers";
const LOADPOINTS = "loadpoints";
const SESSION_COLUMNS = "session_columns";
const SAVINGS_PERIOD = "savings_period";
const SAVINGS_REGION = "savings_region";
const SESSIONS_GROUP = "sessions_group";
const SESSIONS_TYPE = "sessions_type";
const SETTINGS_SOLAR_ADJUSTED = "settings_solar_adjusted";

function read(key: string) {
  return window.localStorage[key];
}

function save(key: string) {
  return (value: string | null) => {
    try {
      if (value) {
        window.localStorage[key] = value;
      } else {
        delete window.localStorage[key];
      }
    } catch (e) {
      console.error("unable to write to localstorage", { key, value, e });
    }
  };
}

function readBool(key: string) {
  return read(key) === "true";
}

function saveBool(key: string) {
  return (value: boolean) => {
    save(key)(value ? "true" : "false");
  };
}

function readArray(key: string) {
  const value = read(key);
  return value ? value.split(",") : [];
}

function saveArray(key: string) {
  return (value: string[]) => {
    save(key)(value.join(","));
  };
}

function readJSON(key: string) {
  const value = read(key);
  try {
    return value ? JSON.parse(value) : {};
  } catch (e) {
    console.error("unable to parse JSON from localStorage", { key, value, e });
    return {};
  }
}

function saveJSON(key: string) {
  return (value: object) => {
    try {
      save(key)(JSON.stringify(value));
    } catch (e) {
      console.error("unable to stringify JSON for localStorage", { key, value, e });
    }
  };
}

export interface LoadpointSettings {
  order?: number;
  visible?: boolean;
  info?: SessionInfoKey;
}

export interface Settings {
  locale: keyof typeof LOCALES | null;
  theme: THEME | null;
  unit: string;
  is12hFormat: boolean;
  hiddenFeatures: boolean;
  energyflowDetails: boolean;
  energyflowPv: boolean;
  energyflowBattery: boolean;
  energyflowLoadpoints: boolean;
  energyflowConsumers: boolean;
  sessionColumns: string[];
  savingsPeriod: string;
  savingsRegion: string;
  sessionsGroup: string;
  sessionsType: string;
  solarAdjusted: boolean;
  loadpoints: Record<string, LoadpointSettings>;
}

const settings: Settings = reactive({
  locale: read(SETTINGS_LOCALE),
  theme: read(SETTINGS_THEME),
  unit: read(SETTINGS_UNIT),
  is12hFormat: readBool(SETTINGS_12H_FORMAT),
  hiddenFeatures: readBool(SETTINGS_HIDDEN_FEATURES),
  energyflowDetails: readBool(SETTINGS_ENERGYFLOW_DETAILS),
  energyflowPv: readBool(SETTINGS_ENERGYFLOW_PV),
  energyflowBattery: readBool(SETTINGS_ENERGYFLOW_BATTERY),
  energyflowLoadpoints: readBool(SETTINGS_ENERGYFLOW_LOADPOINTS),
  energyflowConsumers: readBool(SETTINGS_ENERGYFLOW_CONSUMERS),
  sessionColumns: readArray(SESSION_COLUMNS),
  savingsPeriod: read(SAVINGS_PERIOD),
  savingsRegion: read(SAVINGS_REGION),
  sessionsGroup: read(SESSIONS_GROUP),
  sessionsType: read(SESSIONS_TYPE),
  solarAdjusted: readBool(SETTINGS_SOLAR_ADJUSTED),
  loadpoints: readJSON(LOADPOINTS),
});

watch(() => settings.locale, save(SETTINGS_LOCALE));
watch(() => settings.theme, save(SETTINGS_THEME));
watch(() => settings.unit, save(SETTINGS_UNIT));
watch(() => settings.is12hFormat, saveBool(SETTINGS_12H_FORMAT));
watch(() => settings.hiddenFeatures, saveBool(SETTINGS_HIDDEN_FEATURES));
watch(() => settings.energyflowDetails, saveBool(SETTINGS_ENERGYFLOW_DETAILS));
watch(() => settings.energyflowPv, saveBool(SETTINGS_ENERGYFLOW_PV));
watch(() => settings.energyflowBattery, saveBool(SETTINGS_ENERGYFLOW_BATTERY));
watch(() => settings.energyflowLoadpoints, saveBool(SETTINGS_ENERGYFLOW_LOADPOINTS));
watch(() => settings.energyflowConsumers, saveBool(SETTINGS_ENERGYFLOW_CONSUMERS));
watch(() => settings.sessionColumns as string[], saveArray(SESSION_COLUMNS));
watch(() => settings.savingsPeriod, save(SAVINGS_PERIOD));
watch(() => settings.savingsRegion, save(SAVINGS_REGION));
watch(() => settings.sessionsGroup, save(SESSIONS_GROUP));
watch(() => settings.sessionsType, save(SESSIONS_TYPE));
watch(() => settings.solarAdjusted, saveBool(SETTINGS_SOLAR_ADJUSTED));
watch(() => settings.loadpoints, saveJSON(LOADPOINTS), { deep: true });

export default settings;

// MIGRATIONS

// Convert old comma-separated session_info to new loadpoints structure
// TODO: remove in later release
const SESSION_INFO = "session_info";
const oldSessionInfo = read(SESSION_INFO);
if (oldSessionInfo && Object.keys(settings.loadpoints).length === 0) {
  const sessionInfoArray = oldSessionInfo.split(",");
  sessionInfoArray.forEach((info: string, index: number) => {
    if (info.trim()) {
      const loadpointId = `${index + 1}`;
      if (!settings.loadpoints[loadpointId]) {
        settings.loadpoints[loadpointId] = {};
      }
      settings.loadpoints[loadpointId].info = info.trim() as SessionInfoKey;
    }
  });
  // Remove the old session_info key
  delete window.localStorage[SESSION_INFO];
}
