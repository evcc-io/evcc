// list of currencies where energy price should be displayed in subunits (factor 100)
const ENERGY_PRICE_IN_SUBUNIT = {
  AUD: "c", // Australian cent
  BGN: "st", // Bulgarian stotinka
  BRL: "¢", // Brazilian centavo
  CAD: "¢", // Canadian cent
  CHF: "rp", // Swiss Rappen
  CNY: "f", // Chinese fen
  EUR: "ct", // Euro cent
  GBP: "p", // GB pence
  ILS: "ag", // Israeli agora
  NZD: "c", // New Zealand cent
  PLN: "gr", // Polish grosz
  USD: "¢", // US cent
};

export const POWER_UNIT = Object.freeze({
  W: "W",
  KW: "kW",
  MW: "MW",
  AUTO: "",
});

export default {
  data: function () {
    return {
      POWER_UNIT,
      fmtLimit: 100,
      fmtDigits: 1,
    };
  },
  methods: {
    round: function (num, precision) {
      var base = 10 ** precision;
      return (Math.round(num * base) / base).toFixed(precision);
    },
    fmt: function (val) {
      if (val === undefined || val === null) {
        return 0;
      }
      let absVal = Math.abs(val);
      return absVal >= this.fmtLimit
        ? this.round(absVal / 1e3, this.fmtDigits)
        : this.round(absVal, 0);
    },
    fmtW: function (watt = 0, format = POWER_UNIT.KW, withUnit = true, digits) {
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
        d = POWER_UNIT.KW === unit || POWER_UNIT.MW === unit || 0 === watt ? 1 : 0;
      }
      return `${new Intl.NumberFormat(this.$i18n?.locale, {
        style: "decimal",
        minimumFractionDigits: d,
        maximumFractionDigits: d,
      }).format(value)}${withUnit ? ` ${unit}` : ""}`;
    },
    fmtWh: function (watt, format = POWER_UNIT.KW, withUnit = true, digits) {
      return this.fmtW(watt, format, withUnit, digits) + (withUnit ? "h" : "");
    },
    fmtNumber: function (number, decimals, unit) {
      const style = unit ? "unit" : "decimal";
      return new Intl.NumberFormat(this.$i18n?.locale, {
        style,
        unit,
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      }).format(number);
    },
    fmtGrams: function (gramms, withUnit = true) {
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
    fmtCo2Short: function (gramms) {
      return `${this.fmtNumber(gramms, 0)} g`;
    },
    fmtCo2Medium: function (gramms) {
      return `${this.fmtNumber(gramms, 0)} g/kWh`;
    },
    fmtCo2Long: function (gramms) {
      return `${this.fmtNumber(gramms, 0)} gCO₂e/kWh`;
    },
    fmtUnit: function (val) {
      return Math.abs(val) >= this.fmtLimit ? "k" : "";
    },
    fmtNumberToLocale(val, pad = 0) {
      return val.toLocaleString(this.$i18n?.locale).padStart(pad, "0");
    },
    fmtDurationToTime(date) {
      const diff = date - new Date();
      return this.fmtDuration(diff / 1000, true, "h");
    },
    fmtDurationNs(duration = 0, withUnit = true, minUnit = "s") {
      return this.fmtDuration(duration / 1e9, withUnit, minUnit);
    },
    fmtDuration: function (duration = 0, withUnit = true, minUnit = "s") {
      if (duration <= 0) {
        return "—";
      }
      let roundedDuration = Math.round(duration);
      var seconds = roundedDuration % 60;
      var minutes = Math.floor(roundedDuration / 60) % 60;
      var hours = Math.floor(roundedDuration / 3600);
      var result = "";
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
    fmtDayString: function (date) {
      const YY = `${date.getFullYear()}`;
      const MM = `${date.getMonth() + 1}`.padStart(2, "0");
      const DD = `${date.getDate()}`.padStart(2, "0");
      return `${YY}-${MM}-${DD}`;
    },
    fmtTimeString: function (date) {
      const HH = `${date.getHours()}`.padStart(2, "0");
      const mm = `${date.getMinutes()}`.padStart(2, "0");
      return `${HH}:${mm}`;
    },
    isToday: function (date) {
      const today = new Date();
      return today.toDateString() === date.toDateString();
    },
    isTomorrow: function (date) {
      const tomorrow = new Date();
      tomorrow.setDate(tomorrow.getDate() + 1);
      return tomorrow.toDateString() === date.toDateString();
    },
    weekdayPrefix: function (date) {
      if (this.isToday(date)) {
        return "";
      }
      if (this.isTomorrow(date)) {
        try {
          const rtf = new Intl.RelativeTimeFormat(this.$i18n?.locale, { numeric: "auto" });
          return rtf.formatToParts(1, "day")[0].value;
        } catch (e) {
          console.warn("weekdayPrefix: Intl.RelativeTimeFormat not supported", e);
          return "tomorrow";
        }
      }
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        weekday: "short",
      }).format(date);
    },
    weekdayTime: function (date) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        weekday: "short",
        hour: "numeric",
        minute: "numeric",
      }).format(date);
    },
    weekdayShort: function (date) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        weekday: "short",
      }).format(date);
    },
    fmtAbsoluteDate: function (date) {
      const weekday = this.weekdayPrefix(date);
      const hour = new Intl.DateTimeFormat(this.$i18n?.locale, {
        hour: "numeric",
        minute: "numeric",
      }).format(date);

      return `${weekday} ${hour}`.trim();
    },
    fmtFullDateTime: function (date, short) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        weekday: short ? undefined : "short",
        month: short ? "numeric" : "short",
        day: "numeric",
        hour: "numeric",
        minute: "numeric",
      }).format(date);
    },
    fmtMonthYear: function (date) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        month: "long",
        year: "numeric",
      }).format(date);
    },
    fmtMonth: function (date, short) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        month: short ? "short" : "long",
      }).format(date);
    },
    fmtYear: function (date) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        year: "numeric",
      }).format(date);
    },
    fmtDayMonthYear: function (date) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        day: "numeric",
        month: "long",
        year: "numeric",
      }).format(date);
    },
    fmtDayMonth: function (date) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        weekday: "short",
        day: "numeric",
        month: "short",
      }).format(date);
    },
    fmtDayOfMonth: function (date) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        weekday: "short",
        day: "numeric",
      }).format(date);
    },
    fmtSecondUnit: function (seconds) {
      return new Intl.NumberFormat(this.$i18n?.locale, {
        style: "unit",
        unit: "second",
        unitDisplay: "long",
      })
        .formatToParts(seconds)
        .find((part) => part.type === "unit").value;
    },
    fmtMoney: function (amout = 0, currency = "EUR", decimals = true, withSymbol = false) {
      const currencyDisplay = withSymbol ? "narrowSymbol" : "code";
      const result = new Intl.NumberFormat(this.$i18n?.locale, {
        style: "currency",
        currency,
        currencyDisplay,
        maximumFractionDigits: decimals ? undefined : 0,
      }).format(amout);

      return withSymbol ? result : result.replace(currency, "").trim();
    },
    fmtCurrencySymbol: function (currency = "EUR") {
      const symbols = { EUR: "€", USD: "$" };
      return symbols[currency] || currency;
    },
    fmtPricePerKWh: function (amout = 0, currency = "EUR", short = false, withUnit = true) {
      let value = amout;
      let minimumFractionDigits = 1;
      let maximumFractionDigits = 3;
      if (ENERGY_PRICE_IN_SUBUNIT[currency]) {
        value *= 100;
        minimumFractionDigits = 1;
        maximumFractionDigits = 1;
      }
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
    pricePerKWhUnit: function (currency = "EUR", short = false) {
      const unit = ENERGY_PRICE_IN_SUBUNIT[currency] || currency;
      return `${unit}${short ? "" : "/kWh"}`;
    },
    fmtTimeAgo: function (elapsed) {
      const units = {
        day: 24 * 60 * 60 * 1000,
        hour: 60 * 60 * 1000,
        minute: 60 * 1000,
        second: 1000,
      };

      // "Math.abs" accounts for both "past" & "future" scenarios
      for (var u in units)
        if (Math.abs(elapsed) > units[u] || u == "second") {
          try {
            const rtf = new Intl.RelativeTimeFormat(this.$i18n?.locale, { numeric: "auto" });
            return rtf.format(Math.round(elapsed / units[u]), u);
          } catch (e) {
            console.warn("fmtTimeAgo: Intl.RelativeTimeFormat not supported", e);
            return `${elapsed} ${u}s ago`;
          }
        }
    },
    fmtSocOption: function (soc, rangePerSoc, distanceUnit, heating) {
      let result = heating ? this.fmtTemperature(soc) : `${this.fmtPercentage(soc)}`;
      if (rangePerSoc && distanceUnit) {
        const range = soc * rangePerSoc;
        result += ` (${this.fmtNumber(range, 0)} ${distanceUnit})`;
      }
      return result;
    },
    fmtPercentage: function (value, digits = 0) {
      return new Intl.NumberFormat(this.$i18n?.locale, {
        style: "percent",
        minimumFractionDigits: digits,
        maximumFractionDigits: digits,
      }).format(value / 100);
    },
    hasLeadingPercentageSign: function () {
      return ["tr", "ar"].includes(this.$i18n?.locale);
    },
    fmtTemperature: function (value) {
      // TODO: handle fahrenheit
      return this.fmtNumber(value, 1, "celsius");
    },
  },
};
