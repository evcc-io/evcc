import settings from "./settings";

export function getSessionInfo(index, fallback) {
  return settings.sessionInfo[index - 1] || fallback;
}

export function setSessionInfo(index, value) {
  const clone = [...settings.sessionInfo];
  clone[index - 1] = value;
  clone.map((v) => v || "");
  settings.sessionInfo = clone;
}
