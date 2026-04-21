import { defineComponent } from "vue";
import { is12hFormat } from "@/units";
import { CURRENCY } from "../types/evcc";
import settings from "@/settings";
import type { DateFormat } from "@/settings";

// Format the day+month portion of a date according to the user's date-order
// preference.  Month names are always rendered in the given UI locale so they
// stay translated regardless of the ordering choice.
//   dmy  → "17 May"   (or "17 Mai" in German)
//   mdy  → "May 17"
//   ymd  → "2025-05-17"
//   ""   → locale-native (existing behaviour, auto)
function formatDayMonth(date: Date, locale: string | undefined, fmt: DateFormat): string {
  const monthName = new Intl.DateTimeFormat(locale, { month: "short" }).format(date);
  const day = date.getDate();
  const year = date.getFullYear();
  const mm = String(date.getMonth() + 1).padStart(2, "0");
  const dd = String(day).padStart(2, "0");
  if (fmt === "mdy") return `${monthName} ${day}`;
  if (fmt === "ymd") return `${year}-${mm}-${dd}`;
  if (fmt === "dmy") return `${day} ${monthName}`;
  // auto: use locale-native ordering
  return new Intl.DateTimeFormat(locale, { month: "short", day: "numeric" }).format(date);
}

// Format the numeric date portion (no month names) with the right day/month
// ordering.  Used in the compact "short" date format.
//   dmy  → "17/05"  (via en-GB locale)
//   mdy  → "5/17"   (via en-US locale)
//   ymd  → "2025-05-17" (explicit construction — Intl does not include year with only month+day)
//   ""   → locale-native
function formatNumericDate(date: Date, locale: string | undefined, fmt: DateFormat): string {
  if (fmt === "ymd") {
    const mm = String(date.getMonth() + 1).padStart(2, "0");
    const dd = String(date.getDate()).padStart(2, "0");
    return `${date.getFullYear()}-${mm}-${dd}`;
  }
  const orderLocale = fmt === "mdy" ? "en-US" : fmt === "dmy" ? "en-GB" : locale;
  return new Intl.DateTimeFormat(orderLocale, { month: "numeric", day: "numeric" }).format(date);
}

const CURRENCY_SYMBOLS: Record<CURRENCY, string> = {
  AUD: "$",
  BGN: "лв",
  BRL: "R$",
  CAD: "$",
  CHF: "Fr.",
  CNY: "¥",
  CZK: "Kč",
  EUR: "€",
  GBP: "£",
  HUF: "Ft",
  ILS: "₪",
  JPY: "¥",
  NZD: "$",
  NOK: "kr",
  PLN: "zł",
  RON: "lei",
  USD: "$",
  DKK: "kr",
  SEK: "kr",
};

// list of currencies where energy price should be displayed in subunits (factor 100)
const ENERGY_PRICE_IN_SUBUNIT: Partial<Record<CURRENCY, string>> = {
  AUD: "c", // Australian cent
  BGN: "st", // Bulgarian stotinka
  BRL: "¢", // Brazilian centavo
  CAD: "¢", // Canadian cent
  EUR: "ct", // Euro cent
  GBP: "p", // GB pence
  ILS: "ag", // Israeli agora
  NZD: "c", // New Zealand cent
  NOK: "øre", // Norwegian øre
  PLN: "gr", // Polish grosz
  USD: "¢", // US cent
  DKK: "øre", // Danish øre
  SEK: "öre", // Swedish öre
};

export enum POWER_UNIT {
  W = "W",
  KW = "kW",
  MW = "MW",
  AUTO = "",
}

export default defineComponent({
  data() {
    return {
      POWER_UNIT,
      fmtLimit: 100,
      fmtDigits: 1,
    };
  },
  computed: {
    dateFormat(): DateFormat {
      return settings.dateFormat || "";
    },
  },
  methods: {
    energyPriceSubunit(currency: CURRENCY): string | undefined {
      if (currency === CURRENCY.CHF) {
        return this.$i18n?.locale === "de" ? "Rp." : "ct.";
      }
      return ENERGY_PRICE_IN_SUBUNIT[currency];
    },
    round(num: number, precision: number) {
      const base = 10 ** precision;
      return (Math.round(num * base) / base).toFixed(precision);
    },
    fmtW(watt = 0, format = POWER_UNIT.KW, withUnit = true, digits?: number) {
      let unit = format;
      let d = digits;
      if (POWER_UNIT.AUTO === unit) {
        if (watt >= 10_000_000) {
          unit = POWER_UNIT.MW;
        } else if (watt >= 1000 || 0 === watt) {
          unit = POWER_UNIT.KW;
        } else {
          unit = POWER_UNIT.W;
        }
      }
      let value = watt;
      if (POWER_UNIT.KW === unit) {
        value = watt / 1000;
      } else if (POWER_UNIT.MW === unit) {
        value = watt / 1_000_000;
      }
      if (d === undefined) {
        d =
          POWER_UNIT.KW === unit || POWER_UNIT.MW === unit || (POWER_UNIT.W !== unit && 0 === watt)
            ? 1
            : 0;
      }
      return `${new Intl.NumberFormat(this.$i18n?.locale, {
        style: "decimal",
        minimumFractionDigits: d,
        maximumFractionDigits: d,
      }).format(value)}${withUnit ? ` ${unit}` : ""}`;
    },
    fmtWh(watt: number, format = POWER_UNIT.KW, withUnit = true, digits?: number) {
      return this.fmtW(watt, format, withUnit, digits) + (withUnit ? "h" : "");
    },
    fmtNumber(number: number, decimals: number | undefined, unit?: string) {
      const style = unit ? "unit" : "decimal";
      return new Intl.NumberFormat(this.$i18n?.locale, {
        style,
        unit,
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      }).format(number);
    },
    fmtGrams(gramms: number, withUnit = true) {
      let unit = "gram";
      let value = gramms;
      if (gramms >= 1000) {
        unit = "kilogram";
        value = gramms / 1000;
      }
      return new Intl.NumberFormat(this.$i18n?.locale, {
        style: withUnit ? "unit" : "decimal",
        unit,
        minimumFractionDigits: 0,
        maximumFractionDigits: 0,
      }).format(value);
    },
    fmtCo2Short(gramms = 0) {
      return `${this.fmtNumber(gramms, 0)} g`;
    },
    fmtCo2Medium(gramms = 0) {
      return `${this.fmtNumber(gramms, 0)} g/kWh`;
    },
    fmtCo2Long(gramms = 0) {
      return `${this.fmtNumber(gramms, 0)} gCO₂e/kWh`;
    },
    fmtNumberToLocale(val: number, pad = 0) {
      return val.toLocaleString(this.$i18n?.locale).padStart(pad, "0");
    },
    fmtDurationToTime(date: Date) {
      const diff = date.getTime() - new Date().getTime();
      return this.fmtDuration(diff / 1000, true, "h");
    },
    fmtDurationNs(duration = 0, withUnit = true, minUnit = "s") {
      return this.fmtDuration(duration / 1e9, withUnit, minUnit);
    },
    fmtDuration(duration = 0, withUnit = true, minUnit = "s") {
      if (duration <= 0) {
        return "—";
      }
      const roundedDuration = Math.round(duration);
      const seconds = roundedDuration % 60;
      const minutes = Math.floor(roundedDuration / 60) % 60;
      const hours = Math.floor(roundedDuration / 3600);
      let result = "";
      let unit = "";
      if (hours >= 1 || minUnit === "h") {
        result = `${this.fmtNumberToLocale(hours)}:${this.fmtNumberToLocale(minutes, 2)}`;
        unit = "h";
      } else if (minutes >= 1 || minUnit === "m") {
        result = `${this.fmtNumberToLocale(minutes)}:${this.fmtNumberToLocale(seconds, 2)}`;
        unit = "m";
      } else {
        result = `${this.fmtNumberToLocale(seconds)}`;
        unit = "s";
      }
      if (withUnit) {
        result += `\u202F${unit}`;
      }
      return result;
    },
    fmtDurationLong(seconds: number, style: "short" | "long" = "long") {
      // @ts-expect-error - Intl.DurationFormat is a new API not yet in TS types, see https://github.com/microsoft/TypeScript/issues/60608
      if (!Intl.DurationFormat) {
        // old browser fallback
        return this.fmtDuration(seconds);
      }
      const hours = Math.floor(seconds / 3600);
      const minutes = Math.floor((seconds % 3600) / 60);

      // @ts-expect-error - Intl.DurationFormat is a new API not yet in TS types, see https://github.com/microsoft/TypeScript/issues/60608
      const formatter = new Intl.DurationFormat(this.$i18n?.locale, { style });
      return formatter.format({ minutes, hours });
    },
    fmtDurationParts(parts: Record<string, number>) {
      // @ts-expect-error - Intl.DurationFormat is a new API not yet in TS types
      return new Intl.DurationFormat(this.$i18n?.locale, { style: "long" }).format(parts);
    },
    fmtDayString(date: Date) {
      const YY = `${date.getFullYear()}`;
      const MM = `${date.getMonth() + 1}`.padStart(2, "0");
      const DD = `${date.getDate()}`.padStart(2, "0");
      return `${YY}-${MM}-${DD}`;
    },
    fmtTimeString(date: Date) {
      const HH = `${date.getHours()}`.padStart(2, "0");
      const mm = `${date.getMinutes()}`.padStart(2, "0");
      return `${HH}:${mm}`;
    },
    isToday(date: Date) {
      const today = new Date();
      return today.toDateString() === date.toDateString();
    },
    weekdayPrefix(date: Date) {
      if (this.isToday(date)) {
        return "";
      }
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        weekday: "short",
      }).format(date);
    },
    hourShort(date: Date) {
      const locale = this.$i18n?.locale;
      // special: use shorter german format
      if (locale === "de") return date.getHours();
      return new Intl.DateTimeFormat(locale, {
        hour: "numeric",
        hour12: is12hFormat(),
      }).format(date);
    },
    weekdayShort(date: Date) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        weekday: "short",
      }).format(date);
    },
    weekdayLong(date: Date) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        weekday: "long",
      }).format(date);
    },
    fmtAbsoluteDate(date: Date) {
      const weekday = this.weekdayPrefix(date);
      const hour = new Intl.DateTimeFormat(this.$i18n?.locale, {
        hour: "numeric",
        minute: "numeric",
        hour12: is12hFormat(),
      }).format(date);

      return `${weekday} ${hour}`.trim();
    },
    fmtHourMinute(date: Date) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        hour: "numeric",
        minute: "numeric",
        hour12: is12hFormat(),
      }).format(date);
    },
    fmtFullDateTime(date: Date, short: boolean) {
      const locale = this.$i18n?.locale;
      const fmt = this.dateFormat;
      if (!fmt) {
        // auto: single Intl call preserves locale-native separators (e.g. German "So., 15. Jan.,")
        return new Intl.DateTimeFormat(locale, {
          weekday: short ? undefined : "short",
          month: short ? "numeric" : "short",
          day: "numeric",
          hour: "numeric",
          minute: "numeric",
          hour12: is12hFormat(),
        }).format(date);
      }
      const time = new Intl.DateTimeFormat(locale, {
        hour: "numeric",
        minute: "numeric",
        hour12: is12hFormat(),
      }).format(date);
      if (short) {
        return `${formatNumericDate(date, locale, fmt)} ${time}`.trim();
      }
      const weekday = new Intl.DateTimeFormat(locale, { weekday: "short" }).format(date);
      return `${weekday} ${formatDayMonth(date, locale, fmt)} ${time}`.trim();
    },
    fmtWeekdayTime(date: Date) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        weekday: "short",
        hour: "numeric",
        minute: "numeric",
        hour12: is12hFormat(),
      }).format(date);
    },
    fmtMonthYear(date: Date) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        month: "long",
        year: "numeric",
      }).format(date);
    },
    fmtMonth(date: Date, short: boolean) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        month: short ? "short" : "long",
      }).format(date);
    },
    fmtDayMonth(date: Date) {
      const locale = this.$i18n?.locale;
      const fmt = this.dateFormat;
      if (!fmt) {
        return new Intl.DateTimeFormat(locale, {
          weekday: "short",
          day: "numeric",
          month: "short",
        }).format(date);
      }
      const weekday = new Intl.DateTimeFormat(locale, { weekday: "short" }).format(date);
      return `${weekday} ${formatDayMonth(date, locale, fmt)}`.trim();
    },
    fmtDurationUnit(value: number, unit = "second") {
      return new Intl.NumberFormat(this.$i18n?.locale, {
        style: "unit",
        unit,
        unitDisplay: "long",
      })
        .formatToParts(value)
        .find((part) => part.type === "unit")?.value;
    },
    fmtMoney(amout = 0, currency = CURRENCY.EUR, decimals = true, withSymbol = false) {
      const currencyDisplay = withSymbol ? "narrowSymbol" : "code";
      const digits = decimals ? undefined : 0;
      const result = new Intl.NumberFormat(this.$i18n?.locale, {
        style: "currency",
        currency,
        currencyDisplay,
        minimumFractionDigits: digits,
        maximumFractionDigits: digits,
      }).format(amout);

      return withSymbol ? result : result.replace(currency, "").trim();
    },
    fmtCurrencySymbol(currency = CURRENCY.EUR) {
      return CURRENCY_SYMBOLS[currency] || currency;
    },
    fmtCurrencyName(currency: CURRENCY) {
      return (
        new Intl.DisplayNames(this.$i18n?.locale, { type: "currency" }).of(currency) || currency
      );
    },
    fmtPricePerKWh(amout = 0, currency = CURRENCY.EUR, short = false, withUnit = true) {
      const factor = this.pricePerKWhDisplayFactor(currency);
      const value = amout * factor;
      const minimumFractionDigits = 1;
      const maximumFractionDigits = this.energyPriceSubunit(currency) ? 1 : 3;
      const price = new Intl.NumberFormat(this.$i18n?.locale, {
        style: "decimal",
        minimumFractionDigits,
        maximumFractionDigits,
      }).format(value);
      if (withUnit) {
        return `${price} ${this.pricePerKWhUnit(currency, short)}`;
      }
      return price;
    },
    timezone() {
      return Intl?.DateTimeFormat?.().resolvedOptions?.().timeZone || "UTC";
    },
    pricePerKWhUnit(currency = CURRENCY.EUR, short = false) {
      const unit = this.energyPriceSubunit(currency) || CURRENCY_SYMBOLS[currency] || currency;
      return `${unit}${short ? "" : "/kWh"}`;
    },
    pricePerKWhDisplayFactor(currency = CURRENCY.EUR) {
      return this.energyPriceSubunit(currency) ? 100 : 1;
    },
    fmtTimeAgo(elapsed: number) {
      const units = {
        day: 24 * 60 * 60 * 1000,
        hour: 60 * 60 * 1000,
        minute: 60 * 1000,
        second: 1000,
      };

      // "Math.abs" accounts for both "past" & "future" scenarios
      for (const u in units) {
        const unitKey = u as keyof typeof units;
        if (Math.abs(elapsed) > units[unitKey] || u == "second") {
          try {
            const rtf = new Intl.RelativeTimeFormat(this.$i18n?.locale, {
              numeric: "auto",
            });
            return rtf.format(Math.round(elapsed / units[unitKey]), unitKey);
          } catch (e) {
            console.warn("fmtTimeAgo: Intl.RelativeTimeFormat not supported", e);
            return `${elapsed} ${u}s ago`;
          }
        }
      }

      return "";
    },
    fmtSocOption(soc: number, rangePerSoc?: number, distanceUnit?: string, heating?: boolean) {
      let result = heating ? this.fmtTemperature(soc) : `${this.fmtPercentage(soc)}`;
      if (rangePerSoc && distanceUnit) {
        const range = soc * rangePerSoc;
        result += ` (${this.fmtNumber(range, 0)} ${distanceUnit})`;
      }
      return result;
    },
    fmtPercentage(value: number, digits = 0, forceSign = false) {
      const sign = forceSign && value > 0 ? "+" : "";
      return `${sign}${new Intl.NumberFormat(this.$i18n?.locale, {
        style: "percent",
        minimumFractionDigits: digits,
        maximumFractionDigits: digits,
      }).format(value / 100)}`;
    },
    hasLeadingPercentageSign() {
      return ["tr", "ar"].includes(this.$i18n?.locale);
    },
    fmtTemperature(value: number) {
      // TODO: handle fahrenheit
      return this.fmtNumber(value, 1, "celsius");
    },
    fmtWeekdayByIndex(index: number, format: Intl.DateTimeFormatOptions["weekday"]) {
      // June 7, 2021 is Monday (index 1), June 6 is Sunday (index 0)
      const day = index === 0 ? 6 : 6 + index;
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        weekday: format,
      }).format(new Date(2021, 5, day)); // local date avoids UTC timezone day shift
    },
    fmtMonthByIndex(index: number, format: Intl.DateTimeFormatOptions["month"]) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        month: format,
      }).format(new Date(2021, index, 1)); // local date avoids UTC timezone day shift
    },
    getWeekdaysList(
      format: Intl.DateTimeFormatOptions["weekday"]
    ): { name: string; value: number }[] {
      return Array.from({ length: 7 }, (_, i) => {
        const value = (i + 1) % 7; // Mon=1, Tue=2, ..., Sat=6, Sun=0
        return { name: this.fmtWeekdayByIndex(value, format), value };
      });
    },
    getMonthsList(format: Intl.DateTimeFormatOptions["month"]): { name: string; value: number }[] {
      return Array.from({ length: 12 }, (_, i) => ({
        name: this.fmtMonthByIndex(i, format),
        value: i,
      }));
    },
    fmtConsecutiveRange(
      selectedIndices: number[],
      getNameFn: (transformedIndex: number) => string | undefined,
      transformFn?: (index: number) => number
    ): string {
      if (!selectedIndices || selectedIndices.length === 0) {
        return "–";
      }

      // Transform indices if needed (e.g., Sunday 0 -> 7 for weekdays)
      const workingIndices = transformFn ? selectedIndices.map(transformFn) : selectedIndices;

      // Sort the indices
      const sorted = [...workingIndices].sort((a, b) => a - b);
      let label = "";
      const max = Math.max(...sorted);

      for (let i = 0; i < sorted.length; i++) {
        const rangeStart = sorted[i];
        if (rangeStart === undefined) continue;

        label += getNameFn(rangeStart);

        let rangeEnd = rangeStart;
        let j = i;

        // Find consecutive indices
        while (j + 1 < sorted.length && sorted[j + 1] === rangeEnd + 1) {
          rangeEnd++;
          j++;
        }

        if (rangeEnd - rangeStart > 1) {
          // more than 2 consecutive items selected
          label += " – " + getNameFn(rangeEnd);
          i = j;
        } else if (rangeEnd > rangeStart) {
          // 2 consecutive items selected
          label += ", " + getNameFn(rangeEnd);
          i = j;
        }

        const current = sorted[i];
        if (current !== undefined && current < max) {
          label += ", ";
        }
      }

      return label;
    },
    fmtWeekdaysRange(selectedWeekdays: number[]): string {
      const getName = (i: number) => this.fmtWeekdayByIndex(i % 7, "short");
      const transform = (i: number) => i || 7;
      return this.fmtConsecutiveRange(selectedWeekdays, getName, transform);
    },
    fmtMonthsRange(selectedMonths: number[]): string {
      const getName = (i: number) => this.fmtMonthByIndex(i, "short");
      return this.fmtConsecutiveRange(selectedMonths, getName);
    },
    // format a HH:MM to proper formatted time
    fmtTimeStr(timeStr: string): string {
      const [hour, minute] = timeStr.split(":").map((s) => parseInt(s, 10));
      const date = new Date(2021, 0, 1);
      date.setHours(hour!, minute!);
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        hour: "numeric",
        minute: "2-digit",
        hour12: is12hFormat(),
      }).format(date);
    },
    // format a HH:MM-HH:MM to proper formatted range
    fmtTimeRange(timeRange: string): string {
      if (!timeRange) return "";
      const parts = timeRange.split("-");
      if (parts.length !== 2) return timeRange;
      return `${this.fmtTimeStr(parts[0]!)} – ${this.fmtTimeStr(parts[1]!)}`;
    },
  },
});
