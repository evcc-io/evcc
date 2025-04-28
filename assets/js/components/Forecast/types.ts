export function isForecastSlot(obj?: TimeseriesEntry | ForecastSlot): obj is ForecastSlot {
	return (obj as ForecastSlot).start !== undefined;
}

export interface TimeseriesEntry {
	val: number;
	ts: string;
}

export interface ForecastSlot {
	start: string;
	end: string;
	value: number;
}

export interface EnergyByDay {
	energy: number;
	complete: boolean;
}

export interface SolarDetails {
	scale?: number;
	today?: EnergyByDay;
	tomorrow?: EnergyByDay;
	dayAfterTomorrow?: EnergyByDay;
	timeseries?: TimeseriesEntry[];
}
