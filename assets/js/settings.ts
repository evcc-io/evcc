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
const SESSION_INFO = "session_info";
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
  sessionInfo: SessionInfoKey[];
  sessionColumns: string[];
  savingsPeriod: string;
  savingsRegion: string;
  sessionsGroup: string;
  sessionsType: string;
  solarAdjusted: boolean;
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
  sessionInfo: readArray(SESSION_INFO) as SessionInfoKey[],
  sessionColumns: readArray(SESSION_COLUMNS),
  savingsPeriod: read(SAVINGS_PERIOD),
  savingsRegion: read(SAVINGS_REGION),
  sessionsGroup: read(SESSIONS_GROUP),
  sessionsType: read(SESSIONS_TYPE),
  solarAdjusted: readBool(SETTINGS_SOLAR_ADJUSTED),
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
watch(() => settings.sessionInfo, saveArray(SESSION_INFO));
watch(() => settings.sessionColumns as string[], saveArray(SESSION_COLUMNS));
watch(() => settings.savingsPeriod, save(SAVINGS_PERIOD));
watch(() => settings.savingsRegion, save(SAVINGS_REGION));
watch(() => settings.sessionsGroup, save(SESSIONS_GROUP));
watch(() => settings.sessionsType, save(SESSIONS_TYPE));
watch(() => settings.solarAdjusted, saveBool(SETTINGS_SOLAR_ADJUSTED));
export default settings;
