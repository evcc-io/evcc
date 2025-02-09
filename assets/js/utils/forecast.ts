export interface PriceSlot {
	start: string;
	end: string;
	price: number;
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
		console.log(start, end, isPast, dateMatches, dayString);
		return !isPast && dateMatches;
	});
}

// return the energy for today from now on
export function todaysEnergy(slots: PriceSlot[]): number {
	const now = new Date();
	const todaysSlots = filterSlotsByDate(slots, toDayString(now));
	return aggregateEnergy(todaysSlots, true);
}

// return the energy for tomorrow
export function tomorrowsEnergy(slots: PriceSlot[]): number {
	const tomorrow = new Date();
	tomorrow.setDate(tomorrow.getDate() + 1);
	const tomorrowsSlots = filterSlotsByDate(slots, toDayString(tomorrow));
	return aggregateEnergy(tomorrowsSlots, true);
}
