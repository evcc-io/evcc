import settings from "./settings";
import { LENGTH_UNIT } from "./types/evcc";

const MILES_FACTOR = 0.6213711922;

function isMiles() {
  return settings.unit === LENGTH_UNIT.MILES;
}

export function distanceValue(value: number) {
  return isMiles() ? value * MILES_FACTOR : value;
}

export function distanceUnit() {
  return isMiles() ? "mi" : "km";
}

export function getUnits() {
  return isMiles() ? LENGTH_UNIT.MILES : LENGTH_UNIT.KM;
}

export function setUnits(value: LENGTH_UNIT) {
  settings.unit = value;
}

export function is12hFormat() {
  return settings.is12hFormat;
}

export function set12hFormat(value: boolean) {
  settings.is12hFormat = value;
}

export function getTimezone() {
  return settings.timezone || "";
}

export function setTimezone(value: string) {
  settings.timezone = value;
}

// resolveTimezone returns the effective IANA timezone string given the user
// preference and the server-reported timezone.  Centralising this policy here
// keeps formatter.ts focused on display logic and makes the rules easy to test.
export function resolveTimezone(
  userPref: string | undefined,
  serverTz: string | undefined
): string {
  const pref = userPref || "";
  const browserTz = Intl?.DateTimeFormat?.().resolvedOptions?.().timeZone || "UTC";
  if (!pref) return browserTz;
  if (pref === "server") return serverTz || browserTz;
  return pref;
}

// Parse "YYYY-MM-DDTHH:mm" as a local time in the given IANA timezone,
// returning the corresponding UTC Date.
//
// DST note: this implementation uses an Intl round-trip to find the UTC offset
// for the given wall-clock time.  Around DST transitions, a local time that is
// skipped (spring-forward gap) will be resolved to the post-transition offset,
// and an ambiguous time (fall-back overlap) will be resolved to the
// pre-transition (summer) offset — consistent with most calendar UIs.
export function parseLocalTimeInTz(dateTimeStr: string, tz: string): Date {
  // Parse the string as if it were UTC to get a reference point
  const naiveUTC = new Date(dateTimeStr + "Z");

  // Find what the target timezone says for this UTC moment
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: tz,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).formatToParts(naiveUTC);
  const get = (t: string) => parts.find((p) => p.type === t)?.value ?? "00";
  const h = get("hour").replace("24", "00");
  const tzDate = new Date(
    `${get("year")}-${get("month")}-${get("day")}T${h}:${get("minute")}:${get("second")}Z`
  );

  // The offset between the target timezone's local time and UTC
  const offsetMs = naiveUTC.getTime() - tzDate.getTime();
  return new Date(naiveUTC.getTime() + offsetMs);
}
