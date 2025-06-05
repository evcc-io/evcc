import { mount, config } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import formatter, { POWER_UNIT } from "./formatter";
import * as units from "../units";
import { defineComponent } from "vue";
import { CURRENCY } from "@/types/evcc";

const is12hSpy = vi.spyOn(units, "is12hFormat").mockReturnValue(false);

config.global.mocks["$i18n"] = { locale: "de-DE" };
config.global.mocks["$t"] = (a: string) => a;

const fmt = mount(
	defineComponent({
		render() {},
		mixins: [formatter],
	})
).vm;

describe("fmtW", () => {
	test("should format with units", () => {
		expect(fmt.fmtW(0, POWER_UNIT.AUTO)).eq("0,0 kW");
		expect(fmt.fmtW(1200000, POWER_UNIT.AUTO)).eq("1.200,0 kW");
		expect(fmt.fmtW(0, POWER_UNIT.MW)).eq("0,0 MW");
		expect(fmt.fmtW(1200000, POWER_UNIT.MW)).eq("1,2 MW");
		expect(fmt.fmtW(0, POWER_UNIT.KW)).eq("0,0 kW");
		expect(fmt.fmtW(1200000, POWER_UNIT.KW)).eq("1.200,0 kW");
		expect(fmt.fmtW(0, POWER_UNIT.W)).eq("0 W");
		expect(fmt.fmtW(1200000, POWER_UNIT.W)).eq("1.200.000 W");
	});
	test("should format without units", () => {
		expect(fmt.fmtW(0, POWER_UNIT.AUTO, false)).eq("0,0");
		expect(fmt.fmtW(1200000, POWER_UNIT.AUTO, false)).eq("1.200,0");
		expect(fmt.fmtW(0, POWER_UNIT.MW, false)).eq("0,0");
		expect(fmt.fmtW(1200000, POWER_UNIT.MW, false)).eq("1,2");
		expect(fmt.fmtW(0, POWER_UNIT.KW, false)).eq("0,0");
		expect(fmt.fmtW(1200000, POWER_UNIT.KW, false)).eq("1.200,0");
		expect(fmt.fmtW(0, POWER_UNIT.W, false)).eq("0");
		expect(fmt.fmtW(1200000, POWER_UNIT.W, false)).eq("1.200.000");
	});
	test("should format a given number of digits", () => {
		expect(fmt.fmtW(12345, POWER_UNIT.AUTO, true, 0)).eq("12 kW");
		expect(fmt.fmtW(12345, POWER_UNIT.AUTO, true, 1)).eq("12,3 kW");
		expect(fmt.fmtW(12345, POWER_UNIT.AUTO, true, 2)).eq("12,35 kW");
		expect(fmt.fmtW(12345, POWER_UNIT.MW, true, 0)).eq("0 MW");
		expect(fmt.fmtW(12345, POWER_UNIT.MW, true, 1)).eq("0,0 MW");
		expect(fmt.fmtW(12345, POWER_UNIT.MW, true, 2)).eq("0,01 MW");
		expect(fmt.fmtW(12345, POWER_UNIT.KW, true, 0)).eq("12 kW");
		expect(fmt.fmtW(12345, POWER_UNIT.KW, true, 1)).eq("12,3 kW");
		expect(fmt.fmtW(12345, POWER_UNIT.KW, true, 2)).eq("12,35 kW");
		expect(fmt.fmtW(12345, POWER_UNIT.W, true, 0)).eq("12.345 W");
		expect(fmt.fmtW(12345, POWER_UNIT.W, true, 1)).eq("12.345,0 W");
		expect(fmt.fmtW(12345, POWER_UNIT.W, true, 2)).eq("12.345,00 W");
	});
});

describe("fmtWh", () => {
	test("should format with units", () => {
		expect(fmt.fmtWh(0, POWER_UNIT.AUTO)).eq("0,0 kWh");
		expect(fmt.fmtWh(1200000, POWER_UNIT.AUTO)).eq("1.200,0 kWh");
		expect(fmt.fmtWh(0, POWER_UNIT.MW)).eq("0,0 MWh");
		expect(fmt.fmtWh(1200000, POWER_UNIT.MW)).eq("1,2 MWh");
		expect(fmt.fmtWh(0, POWER_UNIT.KW)).eq("0,0 kWh");
		expect(fmt.fmtWh(1200000, POWER_UNIT.KW)).eq("1.200,0 kWh");
		expect(fmt.fmtWh(0, POWER_UNIT.W)).eq("0 Wh");
		expect(fmt.fmtWh(1200000, POWER_UNIT.W)).eq("1.200.000 Wh");
	});
	test("should format without units", () => {
		expect(fmt.fmtWh(0, POWER_UNIT.AUTO, false)).eq("0,0");
		expect(fmt.fmtWh(1200000, POWER_UNIT.AUTO, false)).eq("1.200,0");
		expect(fmt.fmtWh(0, POWER_UNIT.MW, false)).eq("0,0");
		expect(fmt.fmtWh(1200000, POWER_UNIT.MW, false)).eq("1,2");
		expect(fmt.fmtWh(0, POWER_UNIT.KW, false)).eq("0,0");
		expect(fmt.fmtWh(1200000, POWER_UNIT.KW, false)).eq("1.200,0");
		expect(fmt.fmtWh(0, POWER_UNIT.W, false)).eq("0");
		expect(fmt.fmtWh(1200000, POWER_UNIT.W, false)).eq("1.200.000");
	});
	test("should format a given number of digits", () => {
		expect(fmt.fmtWh(12345, POWER_UNIT.AUTO, true, 0)).eq("12 kWh");
		expect(fmt.fmtWh(12345, POWER_UNIT.AUTO, true, 1)).eq("12,3 kWh");
		expect(fmt.fmtWh(12345, POWER_UNIT.AUTO, true, 2)).eq("12,35 kWh");
		expect(fmt.fmtWh(12345, POWER_UNIT.MW, true, 0)).eq("0 MWh");
		expect(fmt.fmtWh(12345, POWER_UNIT.MW, true, 1)).eq("0,0 MWh");
		expect(fmt.fmtWh(12345, POWER_UNIT.MW, true, 2)).eq("0,01 MWh");
		expect(fmt.fmtWh(12345, POWER_UNIT.KW, true, 0)).eq("12 kWh");
		expect(fmt.fmtWh(12345, POWER_UNIT.KW, true, 1)).eq("12,3 kWh");
		expect(fmt.fmtWh(12345, POWER_UNIT.KW, true, 2)).eq("12,35 kWh");
		expect(fmt.fmtWh(12345, POWER_UNIT.W, true, 0)).eq("12.345 Wh");
		expect(fmt.fmtWh(12345, POWER_UNIT.W, true, 1)).eq("12.345,0 Wh");
		expect(fmt.fmtWh(12345, POWER_UNIT.W, true, 2)).eq("12.345,00 Wh");
	});
});

describe("fmtPricePerKWh", () => {
	test("should format with units", () => {
		expect(fmt.fmtPricePerKWh(0.2, CURRENCY.EUR)).eq("20,0 ct/kWh");
		expect(fmt.fmtPricePerKWh(0.2, CURRENCY.EUR, true)).eq("20,0 ct");
		expect(fmt.fmtPricePerKWh(0.234, CURRENCY.USD)).eq("23,4 ¢/kWh");
		expect(fmt.fmtPricePerKWh(1234, CURRENCY.SEK)).eq("123.400,0 öre/kWh");
		expect(fmt.fmtPricePerKWh(0.2, CURRENCY.EUR, false, false)).eq("20,0");
		expect(fmt.fmtPricePerKWh(0.123, CURRENCY.CHF)).eq("12,3 rp/kWh");
	});
});

describe("pricePerKWhUnit", () => {
	test("should return correct unit", () => {
		expect(fmt.pricePerKWhUnit(CURRENCY.EUR)).eq("ct/kWh");
		expect(fmt.pricePerKWhUnit(CURRENCY.EUR, true)).eq("ct");
		expect(fmt.pricePerKWhUnit(CURRENCY.USD)).eq("¢/kWh");
		expect(fmt.pricePerKWhUnit(CURRENCY.SEK)).eq("öre/kWh");
		expect(fmt.pricePerKWhUnit(CURRENCY.SEK, true)).eq("öre");
		expect(fmt.pricePerKWhUnit(CURRENCY.CHF)).eq("rp/kWh");
	});
});

describe("fmtDuration", () => {
	test("should format zero duration", () => {
		expect(fmt.fmtDuration(0)).eq("—");
		expect(fmt.fmtDuration(-100)).eq("—");
	});
	test("should format seconds", () => {
		expect(fmt.fmtDuration(1)).eq("1\u202Fs");
		expect(fmt.fmtDuration(59)).eq("59\u202Fs");
		expect(fmt.fmtDuration(59, false)).eq("59");
		expect(fmt.fmtDuration(59, true, "m")).eq("0:59\u202Fm");
		expect(fmt.fmtDuration(59, true, "h")).eq("0:00\u202Fh");
	});
	test("should format minutes", () => {
		expect(fmt.fmtDuration(60)).eq("1:00\u202Fm");
		expect(fmt.fmtDuration(150)).eq("2:30\u202Fm");
		expect(fmt.fmtDuration(150, false)).eq("2:30");
		expect(fmt.fmtDuration(150, true, "h")).eq("0:02\u202Fh");
	});
	test("should format hours", () => {
		expect(fmt.fmtDuration(60 * 60)).eq("1:00\u202Fh");
		expect(fmt.fmtDuration(60 * 60 * 2.5)).eq("2:30\u202Fh");
		expect(fmt.fmtDuration(60 * 60 * 2.5, false)).eq("2:30");
	});
	test("should format internationalized", () => {
		config.global.mocks["$i18n"].locale = "ar-EG";
		expect(fmt.fmtDuration(30, false)).eq("٣٠");
		expect(fmt.fmtDuration(90, false)).eq("١:٣٠");
		expect(fmt.fmtDuration(60 * 100, false)).eq("١:٤٠");
		config.global.mocks["$i18n"].locale = "de-DE";
	});
});

describe("getWeekdaysList", () => {
	test("should return the correct weekday-order", () => {
		expect(fmt.getWeekdaysList("long")).toEqual([
			{ name: "Montag", value: 1 },
			{ name: "Dienstag", value: 2 },
			{ name: "Mittwoch", value: 3 },
			{ name: "Donnerstag", value: 4 },
			{ name: "Freitag", value: 5 },
			{ name: "Samstag", value: 6 },
			{ name: "Sonntag", value: 0 },
		]);
	});
});

describe("getShortenedWeekdaysLabel", () => {
	test("should format single days", () => {
		expect(fmt.getShortenedWeekdaysLabel([0])).eq("So");
		expect(fmt.getShortenedWeekdaysLabel([0, 2, 4, 6])).eq("Di, Do, Sa, So");
		expect(fmt.getShortenedWeekdaysLabel([6])).eq("Sa");
		expect(fmt.getShortenedWeekdaysLabel([3, 6])).eq("Mi, Sa");
	});
	test("should format ranges", () => {
		expect(fmt.getShortenedWeekdaysLabel([1, 2])).eq("Mo, Di");
		expect(fmt.getShortenedWeekdaysLabel([0, 1, 2, 3, 4, 5, 6])).eq("Mo – So");
		expect(fmt.getShortenedWeekdaysLabel([0, 1, 3, 4, 5])).eq("Mo, Mi – Fr, So");
	});
	test("should format single days and ranges", () => {
		expect(fmt.getShortenedWeekdaysLabel([0, 1, 3, 5, 6])).eq("Mo, Mi, Fr – So");
		expect(fmt.getShortenedWeekdaysLabel([0, 2, 3, 5, 6])).eq("Di, Mi, Fr – So");
	});
});

describe("fmtPercentage", () => {
	test("should format percentage", () => {
		expect(fmt.fmtPercentage(0)).eq("0 %");
		expect(fmt.fmtPercentage(100)).eq("100 %");
		expect(fmt.fmtPercentage(100, 1)).eq("100,0 %");
	});
	test("should force signed", () => {
		expect(fmt.fmtPercentage(100, 0, true)).eq("+100 %");
		expect(fmt.fmtPercentage(100, 0, false)).eq("100 %");
		expect(fmt.fmtPercentage(-100, 0, true)).eq("-100 %");
		expect(fmt.fmtPercentage(-100, 0, false)).eq("-100 %");
	});
});

const testDate = new Date(2023, 0, 15, 15, 30); // Jan 15, 2023, 15:30 (3:30 PM)

describe("12h/24h time format", () => {
	test("12h format", () => {
		is12hSpy.mockReturnValue(true);
		expect(fmt.fmtHourMinute(testDate)).toBe("3:30 PM");
		expect(fmt.fmtFullDateTime(testDate, false)).toBe("So., 15. Jan., 3:30 PM");
		expect(fmt.fmtWeekdayTime(testDate)).toBe("So. 3:30 PM");
		expect(fmt.fmtAbsoluteDate(testDate)).toBe("So 3:30 PM");
	});

	test("24h format", () => {
		is12hSpy.mockReturnValue(false);
		expect(fmt.fmtHourMinute(testDate)).toBe("15:30");
		expect(fmt.fmtFullDateTime(testDate, false)).toBe("So., 15. Jan., 15:30");
		expect(fmt.fmtWeekdayTime(testDate)).toBe("So., 15:30");
		expect(fmt.fmtAbsoluteDate(testDate)).toBe("So 15:30");
	});
});
