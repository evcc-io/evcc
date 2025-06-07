import { defineComponent } from "vue";
import { is12hFormat } from "@/units";
import { CURRENCY } from "../types/evcc";

const CURRENCY_SYMBOLS: Record<CURRENCY, string> = {
	AUD: "$",
	BGN: "лв",
	BRL: "R$",
	CAD: "$",
	CHF: "Fr.",
	CNY: "¥",
	EUR: "€",
	GBP: "£",
	ILS: "₪",
	NZD: "$",
	PLN: "zł",
	USD: "$",
	DKK: "kr",
	SEK: "kr",
};

// list of currencies where energy price should be displayed in subunits (factor 100)
const ENERGY_PRICE_IN_SUBUNIT: Record<CURRENCY, string> = {
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
	methods: {
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
					POWER_UNIT.KW === unit ||
					POWER_UNIT.MW === unit ||
					(POWER_UNIT.W !== unit && 0 === watt)
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
		fmtNumber(number: number, decimals: number, unit?: string) {
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
		fmtDurationLong(seconds: number) {
			// @ts-expect-error - Intl.DurationFormat is a new API not yet in TS types, see https://github.com/microsoft/TypeScript/issues/60608
			if (!Intl.DurationFormat) {
				// old browser fallback
				return this.fmtDuration(seconds);
			}
			const hours = Math.floor(seconds / 3600);
			const minutes = Math.floor((seconds % 3600) / 60);

			// @ts-expect-error - Intl.DurationFormat is a new API not yet in TS types, see https://github.com/microsoft/TypeScript/issues/60608
			const formatter = new Intl.DurationFormat(this.$i18n?.locale, { style: "long" });
			return formatter.format({ minutes, hours });
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
		isTomorrow(date: Date) {
			const tomorrow = new Date();
			tomorrow.setDate(tomorrow.getDate() + 1);
			return tomorrow.toDateString() === date.toDateString();
		},
		weekdayPrefix(date: Date) {
			if (this.isToday(date)) {
				return "";
			}
			if (this.isTomorrow(date)) {
				try {
					const rtf = new Intl.RelativeTimeFormat(this.$i18n?.locale, {
						numeric: "auto",
					});
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
			return new Intl.DateTimeFormat(this.$i18n?.locale, {
				weekday: short ? undefined : "short",
				month: short ? "numeric" : "short",
				day: "numeric",
				hour: "numeric",
				minute: "numeric",
				hour12: is12hFormat(),
			}).format(date);
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
			return new Intl.DateTimeFormat(this.$i18n?.locale, {
				weekday: "short",
				day: "numeric",
				month: "short",
			}).format(date);
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
		fmtPricePerKWh(amout = 0, currency = CURRENCY.EUR, short = false, withUnit = true) {
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
		timezone() {
			return Intl?.DateTimeFormat?.().resolvedOptions?.().timeZone || "UTC";
		},
		pricePerKWhUnit(currency = CURRENCY.EUR, short = false) {
			const unit = ENERGY_PRICE_IN_SUBUNIT[currency] || currency;
			return `${unit}${short ? "" : "/kWh"}`;
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
		getWeekdaysList(
			weekdayFormat: Intl.DateTimeFormatOptions["weekday"]
		): { name: string; value: number }[] {
			const { format } = new Intl.DateTimeFormat(this.$i18n?.locale, {
				weekday: weekdayFormat,
			});
			const mondayToSaturday = [7, 8, 9, 10, 11, 12].map((day, index) => {
				return { name: format(new Date(Date.UTC(2021, 5, day))), value: index + 1 };
			});
			const sunday = { name: format(new Date(Date.UTC(2021, 5, 6))), value: 0 };
			return [...mondayToSaturday, sunday];
		},
		getShortenedWeekdaysLabel(selectedWeekdays: number[]): string {
			if (0 === selectedWeekdays.length) {
				return "–";
			}

			const weekdays = this.getWeekdaysList("short");
			let label = "";

			// the week in the input-parameter starts with 0 for sunday and ends with 6 for saturday
			// this algorithms works only if the week starts with 1 for monday and ends with 7 for sunday because
			// then we are able to count from 1 to 7 by incrementing the number
			// so we have to transform the input accordingly
			const selectedWeekdaysTransformed = selectedWeekdays.map(function (dayIndex) {
				return 0 === dayIndex ? 7 : dayIndex;
			});
			function getWeekdayName(dayIndex: number) {
				return weekdays.find((day) => day.value === (7 === dayIndex ? 0 : dayIndex))?.name;
			}

			const maxWeekday = Math.max(...selectedWeekdaysTransformed);

			for (let weekdayRangeStart = 1; weekdayRangeStart < 8; weekdayRangeStart++) {
				if (selectedWeekdaysTransformed.includes(weekdayRangeStart)) {
					label += getWeekdayName(weekdayRangeStart);

					let weekdayRangeEnd = weekdayRangeStart;
					while (selectedWeekdaysTransformed.includes(weekdayRangeEnd + 1)) {
						weekdayRangeEnd++;
					}

					if (weekdayRangeEnd - weekdayRangeStart > 1) {
						// more than 2 consecutive weekdays selected
						label += " – " + getWeekdayName(weekdayRangeEnd);
						weekdayRangeStart = weekdayRangeEnd;
						if (maxWeekday !== weekdayRangeEnd) {
							label += ", ";
						}
					} else if (weekdayRangeStart !== weekdayRangeEnd) {
						// exactly 2 consecutive weekdays selected
						label += ", ";
					} else {
						// exactly 1 single day selected
						if (maxWeekday !== weekdayRangeEnd) {
							label += ", ";
						}
					}
				}
			}
			return label;
		},
	},
});
