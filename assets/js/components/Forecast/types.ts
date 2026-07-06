export function isForecastSlot(obj?: TimeseriesEntry | ForecastSlot): obj is ForecastSlot {
  return (obj as ForecastSlot).start !== undefined;
}

/** A forecast value at a point in time. */
export interface TimeseriesEntry {
  /** Forecast power in W. */
  val: number;
  /**
   * Time of the forecast value.
   * @format date-time
   */
  ts: string;
}

/** A forecast value for a time slot. */
export interface ForecastSlot {
  /**
   * Start of the time slot.
   * @format date-time
   */
  start: string;
  /**
   * End of the time slot.
   * @format date-time
   */
  end: string;
  /** Forecast value of the time slot. Unit depends on the forecast type. */
  value: number;
}

/** Expected solar production energy of a day. */
export interface EnergyByDay {
  /** Expected production energy in kWh. */
  energy: number;
  /** Forecast data covers the whole day. */
  complete: boolean;
}

/** Solar production forecast. */
export interface SolarDetails {
  /** Correction factor applied to the forecast based on past production. */
  scale?: number;
  /** Expected production today. */
  today?: EnergyByDay;
  /** Expected production tomorrow. */
  tomorrow?: EnergyByDay;
  /** Expected production the day after tomorrow. */
  dayAfterTomorrow?: EnergyByDay;
  /** Expected production power over time. */
  timeseries?: TimeseriesEntry[];
}
