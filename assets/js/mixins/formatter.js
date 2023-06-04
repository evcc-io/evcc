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
      return `${new Intl.NumberFormat(this.$i18n.locale, {
        style: "decimal",
        minimumFractionDigits: digits,
        maximumFractionDigits: digits,
      }).format(value)}${unit}`;
    },
    fmtKWh: function (watt, kw = true, withUnit = true, digits) {
      return this.fmtKw(watt, kw, withUnit, digits) + (withUnit ? "h" : "");
    },
    fmtNumber: function (number, decimals) {
      return new Intl.NumberFormat(this.$i18n.locale, {
        style: "decimal",
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
    fmtDuration: function (d) {
      if (d <= 0 || d == null) {
        return "—";
      }
      var seconds = "0" + (d % 60);
      var minutes = "0" + (Math.floor(d / 60) % 60);
      var hours = "" + Math.floor(d / 3600);
      if (hours.length < 2) {
        hours = "0" + hours;
      }
      return hours + ":" + minutes.substr(-2) + ":" + seconds.substr(-2);
    },
    fmtShortDuration: function (duration = 0, withUnit = false) {
      if (duration <= 0) {
        return "—";
      }
      duration = Math.round(duration);
      var seconds = duration % 60;
      var minutes = Math.floor(duration / 60) % 60;
      var hours = Math.floor(duration / 3600);
      var result = "";
      if (hours >= 1) {
        result = hours + ":" + `${minutes}`.padStart(2, "0");
      } else if (minutes >= 1) {
        result = minutes + ":" + `${seconds}`.padStart(2, "0");
      } else {
        result = `${seconds}`;
      }
      if (withUnit) {
        result += this.fmtShortDurationUnit(duration);
      }
      return result;
    },
    fmtShortDurationUnit: function (duration = 0) {
      if (duration <= 0) {
        return "";
      }
      var minutes = Math.floor(duration / 60) % 60;
      var hours = Math.floor(duration / 3600);
      if (hours >= 1) {
        return "h";
      }
      if (minutes >= 1) {
        return "m";
      }
      return "s";
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
      const rtf = new Intl.RelativeTimeFormat(this.$i18n.locale, { numeric: "auto" });

      if (this.isToday(date)) {
        return rtf.formatToParts(0, "day")[0].value;
      }
      if (this.isTomorrow(date)) {
        return rtf.formatToParts(1, "day")[0].value;
      }
      return new Intl.DateTimeFormat(this.$i18n.locale, {
        weekday: "short",
      }).format(date);
    },
    weekdayTime: function (date) {
      return new Intl.DateTimeFormat(this.$i18n.locale, {
        weekday: "short",
        hour: "numeric",
        minute: "numeric",
      }).format(date);
    },
    weekdayShort: function (date) {
      return new Intl.DateTimeFormat(this.$i18n.locale, {
        weekday: "short",
      }).format(date);
    },
    fmtAbsoluteDate: function (date) {
      const weekday = this.weekdayPrefix(date);
      const hour = new Intl.DateTimeFormat(this.$i18n.locale, {
        hour: "numeric",
        minute: "numeric",
      }).format(date);

      return `${weekday} ${hour}`;
    },
    fmtFullDateTime: function (date, short) {
      return new Intl.DateTimeFormat(this.$i18n.locale, {
        weekday: short ? undefined : "short",
        month: short ? "numeric" : "short",
        day: "numeric",
        hour: "numeric",
        minute: "numeric",
      }).format(date);
    },
    fmtMonthYear: function (date) {
      return new Intl.DateTimeFormat(this.$i18n.locale, {
        month: "long",
        year: "numeric",
      }).format(date);
    },
    fmtMonth: function (date, short) {
      return new Intl.DateTimeFormat(this.$i18n.locale, {
        month: short ? "short" : "long",
      }).format(date);
    },
    fmtDayMonthYear: function (date) {
      return new Intl.DateTimeFormat(this.$i18n.locale, {
        day: "numeric",
        month: "long",
        year: "numeric",
      }).format(date);
    },
    fmtMoney: function (amout = 0, currency = "EUR", decimals = true) {
      return new Intl.NumberFormat(this.$i18n.locale, {
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
    fmtPricePerKWh: function (amout = 0, currency = "EUR", short = false) {
      let unit = currency;
      let value = amout;
      let minimumFractionDigits = 1;
      let maximumFractionDigits = 3;
      if (["EUR", "USD"].includes(currency)) {
        value *= 100;
        unit = "ct";
        minimumFractionDigits = 1;
        maximumFractionDigits = 1;
      }
      return `${new Intl.NumberFormat(this.$i18n.locale, {
        style: "decimal",
        minimumFractionDigits,
        maximumFractionDigits,
      }).format(value)} ${unit}${short ? "" : "/kWh"}`;
    },
    fmtTimeAgo: function (elapsed) {
      const units = {
        day: 24 * 60 * 60 * 1000,
        hour: 60 * 60 * 1000,
        minute: 60 * 1000,
        second: 1000,
      };

      const rtf = new Intl.RelativeTimeFormat(this.$i18n.locale, { numeric: "auto" });

      // "Math.abs" accounts for both "past" & "future" scenarios
      for (var u in units)
        if (Math.abs(elapsed) > units[u] || u == "second")
          return rtf.format(Math.round(elapsed / units[u]), u);
    },
  },
};
