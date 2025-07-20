import settings from "@/settings";
import type { SessionInfoKey } from "@/types/evcc";

export function getSessionInfo(index: number): SessionInfoKey | undefined {
  return settings.sessionInfo[index - 1];
}

export function setSessionInfo(index: number, value: SessionInfoKey) {
  const clone = [...settings.sessionInfo];
  clone[index - 1] = value;
  clone.map((v) => v || "");
  settings.sessionInfo = clone;
}
