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

export default {
  data: function () {
    return {
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
      val = Math.abs(val);
      return val >= this.fmtLimit ? this.round(val / 1e3, this.fmtDigits) : this.round(val, 0);
    },
    fmtKw: function (watt = 0, kw = true, withUnit = true, digits) {
      if (digits === undefined) {
        digits = kw ? 1 : 0;
      }
      const value = kw ? watt / 1000 : watt;
      let unit = "";
      if (withUnit) {
        unit = kw ? " kW" : " W";
      }
      return `${new Intl.NumberFormat(this.$i18n?.locale, {
        style: "decimal",
        minimumFractionDigits: digits,
        maximumFractionDigits: digits,
      }).format(value)}${unit}`;
    },
    fmtKWh: function (watt, kw = true, withUnit = true, digits) {
      return this.fmtKw(watt, kw, withUnit, digits) + (withUnit ? "h" : "");
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
    fmtDurationNs(duration = 0, withUnit = true, minUnit = "s") {
      return this.fmtDuration(duration / 1e9, withUnit, minUnit);
    },
    fmtDuration: function (duration = 0, withUnit = true, minUnit = "s") {
      if (duration <= 0) {
        return "—";
      }
      duration = Math.round(duration);
      var seconds = duration % 60;
      var minutes = Math.floor(duration / 60) % 60;
      var hours = Math.floor(duration / 3600);
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
      const rtf = new Intl.RelativeTimeFormat(this.$i18n?.locale, { numeric: "auto" });

      if (this.isToday(date)) {
        return rtf.formatToParts(0, "day")[0].value;
      }
      if (this.isTomorrow(date)) {
        return rtf.formatToParts(1, "day")[0].value;
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

      return `${weekday} ${hour}`;
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
    fmtDayMonthYear: function (date) {
      return new Intl.DateTimeFormat(this.$i18n?.locale, {
        day: "numeric",
        month: "long",
        year: "numeric",
      }).format(date);
    },
    fmtMoney: function (amout = 0, currency = "EUR", decimals = true) {
      return new Intl.NumberFormat(this.$i18n?.locale, {
        style: "currency",
        currency,
        currencyDisplay: "code",
        maximumFractionDigits: decimals ? undefined : 0,
      })
        .format(amout)
        .replace(currency, "")
        .trim();
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

      const rtf = new Intl.RelativeTimeFormat(this.$i18n?.locale, { numeric: "auto" });

      // "Math.abs" accounts for both "past" & "future" scenarios
      for (var u in units)
        if (Math.abs(elapsed) > units[u] || u == "second")
          return rtf.format(Math.round(elapsed / units[u]), u);
    },
    fmtSocOption: function (soc, rangePerSoc, distanceUnit, heating) {
      let result = heating ? this.fmtTemperature(soc) : `${this.fmtNumber(soc, 0)}%`;
      if (rangePerSoc && distanceUnit) {
        const range = soc * rangePerSoc;
        result += ` (${this.fmtNumber(range, 0)} ${distanceUnit})`;
      }
      return result;
    },
    fmtTemperature: function (value) {
      // TODO: handle fahrenheit
      return this.fmtNumber(value, 1, "celsius");
    },
  },
};
