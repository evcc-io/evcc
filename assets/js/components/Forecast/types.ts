export function isForecastSlot(obj?: TimeseriesEntry | ForecastSlot): obj is ForecastSlot {
  return (obj as ForecastSlot).start !== undefined;
}

export interface TimeseriesEntry {
  val: number;
  ts: number;
}

export interface ForecastSlot {
  start: number;
  end: number;
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
