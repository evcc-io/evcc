import { COUNTRIES } from "@/types/evcc";

// country of an IANA time zone from the browser's CLDR data, undefined when unknown
// or unsupported (Firefox); Safari exposes the accessor, Chrome the method
export function timeZoneCountry(
  timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone
): string | undefined {
  return COUNTRIES.find((code) => {
    const locale = new Intl.Locale(`und-${code}`) as {
      getTimeZones?: () => string[];
      timeZones?: string[];
    };
    return (locale.getTimeZones?.() ?? locale.timeZones)?.includes(timeZone);
  });
}
