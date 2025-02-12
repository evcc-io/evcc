export interface PriceSlot {
	start: string;
	end: string;
	price: number;
}

export enum ForecastType {
	Solar = "solar",
	Price = "price",
	Co2 = "co2",
}

export function aggregateEnergy(slots: PriceSlot[], ignorePast: boolean = false): number {
	const now = new Date();
	return slots.reduce((acc: number, { start, end, price: power }: PriceSlot) => {
		let startDate = new Date(start);
		const endDate = new Date(end);
		if (ignorePast) {
			if (endDate < now) {
				return acc; // ignore past slots
			}
			if (startDate < now) {
				startDate = now; // count this slot from now on
			}
		}
		const hours = (endDate.getTime() - startDate.getTime()) / (1000 * 60 * 60); // convert ms to hours
		const energy = power * hours; // Wh
		return acc + energy;
	}, 0);
}

// return the date in local YYYY-MM-DD format
function toDayString(date: Date): string {
	const year = date.getFullYear();
	const month = String(date.getMonth() + 1).padStart(2, "0");
	const day = String(date.getDate()).padStart(2, "0");
	return `${year}-${month}-${day}`;
}

// return only slots that are on a given date, ignores slots that are in the past
function filterSlotsByDate(slots: PriceSlot[], dayString: string): PriceSlot[] {
	const now = new Date();
	return slots.filter(({ start, end }) => {
		const isPast = new Date(end) < now;
		const dateMatches = toDayString(new Date(start)) === dayString;
		return !isPast && dateMatches;
	});
}

// return the energy for a given day (0 = today, 1 = tomorrow, etc.)
export function energyByDay(slots: PriceSlot[], day: number = 0): number {
	const date = new Date();
	date.setDate(date.getDate() + day);
	const daySlots = filterSlotsByDate(slots, toDayString(date));
	return aggregateEnergy(daySlots, true);
}

export function isDayComplete(slots: PriceSlot[], day: number = 0): boolean {
	const date = new Date();
	date.setDate(date.getDate() + day);
	const daySlots = filterSlotsByDate(slots, toDayString(date));
	return daySlots.length === 24;
}

// return the highest slot for a given day (0 = today, 1 = tomorrow, etc.)
export function highestSlotIndexByDay(slots: PriceSlot[], day: number = 0): number {
	const date = new Date();
	date.setDate(date.getDate() + day);
	const daySlots = filterSlotsByDate(slots, toDayString(date));
	const sortedSlots = daySlots.sort((a, b) => b.price - a.price);
	const highestSlot = sortedSlots[0] || {};
	return slots.findIndex((slot) => slot.start === highestSlot.start);
}
