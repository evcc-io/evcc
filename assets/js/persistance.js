const DEFAULT_TARGET_TIME = "7:00";
const LAST_TARGET_TIME_KEY = "last_target_time";
const ENERGYFLOW_DETAILS_KEY = "energyflow_details";
const SELECTED_LOADPOINT_KEY = "selected_loadpoint";

function read(key) {
  return window.localStorage[key];
}

function save(key, value) {
  try {
    window.localStorage[key] = value;
  } catch (e) {
    console.warn(e);
  }
}

export function readLastTargetTime() {
  return read(LAST_TARGET_TIME_KEY) || DEFAULT_TARGET_TIME;
}

export function saveLastTargetTime(time) {
  save(LAST_TARGET_TIME_KEY, time);
}

export function readEnergyflowDetails() {
  return read(ENERGYFLOW_DETAILS_KEY) === "true";
}

export function saveEnergyflowDetails(isOpen) {
  save(ENERGYFLOW_DETAILS_KEY, `${isOpen}`);
}

export function readSelectedLoadpoint() {
  return parseInt(read(SELECTED_LOADPOINT_KEY), 10) || 0;
}

export function saveSelectedLoadpoint(index) {
  save(SELECTED_LOADPOINT_KEY, `${index}`);
}
