import { describe, expect, test, vi } from "vitest";
import { aggregateEnergy, type PriceSlot } from "./forecast";

const slots: PriceSlot[] = [
	{ start: "2050-03-20T10:00:00+01:00", end: "2050-03-20T10:30:00+01:00", price: 2000 }, // 1kWh
	{ start: "2050-03-20T10:30:00+01:00", end: "2050-03-20T11:00:00+01:00", price: 4000 }, // 2kWh
	{ start: "2050-03-20T11:00:00+01:00", end: "2050-03-20T11:30:00+01:00", price: 3000 }, // 1.5kWh
	{ start: "2050-03-20T11:30:00+01:00", end: "2050-03-20T12:00:00+01:00", price: 2000 }, // 1kWh
	{ start: "2050-03-20T12:00:00+01:00", end: "2050-03-20T12:30:00+01:00", price: 0 }, // 0kWh
	{ start: "2050-03-20T12:30:00+01:00", end: "2050-03-20T13:00:00+01:00", price: 0 }, // 0kWh
];

describe("aggregateEnergy", () => {
	test("aggregates energy from power values for 30min slots", () => {
		const result = aggregateEnergy(slots);
		expect(result).toBeCloseTo(5500);
	});
	test("ignore slots in the past", () => {
		// will ignore the first two slots
		vi.setSystemTime("2050-03-20T11:00:00+01:00");
		const result = aggregateEnergy(slots, true);
		expect(result).toBeCloseTo(2500);
	});
	test("correctly split in-between slots", () => {
		// start in the middle of the 4th slot
		vi.setSystemTime("2050-03-20T11:45:00+01:00");
		const result = aggregateEnergy(slots, true);
		expect(result).toBeCloseTo(500);
	});
});
