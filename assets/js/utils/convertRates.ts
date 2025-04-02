import type { RateRaw, Rate } from "../types/evcc";

function convertRate(rate: RateRaw): Rate {
	return {
		start: new Date(rate.start),
		end: new Date(rate.end),
		value: rate.value,
	};
}

export default function convertRates(rates: RateRaw[] | null): Rate[] {
	if (!rates) return [];
	return rates.map(convertRate);
}
