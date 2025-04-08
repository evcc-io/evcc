import type { StaticPlan, RepeatingPlan } from "../components/ChargingPlans/types";

declare global {
	interface Window {
		app: any;
	}
}

export interface State {
	offline: boolean;
	loadpoints: [];
	forecast?: Forecast;
	currency?: CURRENCY;
}
export interface LoadpointCompact {
	icon: string;
	title: string;
	charging: boolean;
	soc: number;
	power: number;
	chargerFeatureHeating: boolean
}

export enum CURRENCY {
	EUR = "EUR",
	USD = "USD",
	DKK = "DKK",
}

export interface Vehicle {
	name: string;
	minSoc?: number;
	limitSoc?: number;
	plan?: StaticPlan;
	repeatingPlans: RepeatingPlan[];
	title: string;
}

export type Timeout = ReturnType<typeof setInterval> | null;

export interface Tariff {
	rates: Rate[];
	lastUpdate: Date;
}

// data from api with string-based timestamps
export interface RateRaw {
	start: string;
	end: string;
	value: number;
}

export interface Rate {
	start: Date;
	end: Date;
	value: number;
}

export interface Slot {
	day: string;
	value?: number;
	startHour: number;
	endHour: number;
	charging: boolean;
	toLate?: boolean | null;
	warning?: boolean | null;
	isTarget?: boolean | null;
	selectable?: boolean | null;
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
export function isForecastSlot(obj?: TimeseriesEntry | ForecastSlot): obj is ForecastSlot {
	return (obj as ForecastSlot).start !== undefined;
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

export interface Forecast {
	grid?: ForecastSlot[];
	co2?: ForecastSlot[];
	solar?: SolarDetails;
}

export interface SelectOption<T> {
	name: string;
	value: T;
	count?: number;
	disabled?: boolean;
}

// see https://stackoverflow.com/a/54178819
type Omit<T, K extends keyof T> = Pick<T, Exclude<keyof T, K>>;
export type PartialBy<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;

export type ValueOf<T> = T[keyof T];
