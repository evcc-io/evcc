import type { StaticPlan, RepeatingPlan } from "../components/ChargingPlans/types";
import type { ForecastSlot, SolarDetails } from "../components/Forecast/types";

declare global {
	interface Window {
		app: any;
	}
}

export interface FatalError {
	error: any;
	class?: any;
}

export interface State {
	offline: boolean;
	startup?: boolean;
	loadpoints: [];
	forecast?: Forecast;
	currency?: CURRENCY;
	fatal?: FatalError;
}

export interface LoadpointCompact {
	icon: string;
	title: string;
	charging: boolean;
	soc?: number;
	power: number;
	heating?: boolean;
	chargePower: number;
	connected: boolean;
	index: number;
}

export enum CURRENCY {
	EUR = "EUR",
	USD = "USD",
	DKK = "DKK",
}

export enum CHARGE_MODE {
	OFF = "off",
	NOW = "now",
	MINPV = "minpv",
	PV = "pv",
}

export enum PHASES {
	AUTO = 0,
	ONE_PHASE = 1,
	TWO_PHASES = 2,
	THREE_PHASES = 3,
}

export interface Sponsor {
	name: string;
	expiresAt: string;
	expiresSoon: boolean;
}

export interface Battery {
	power: number;
	soc: number;
	controllable: boolean;
	capacity: number; // 0 when not specified
	title?: string;
}

export interface Vehicle {
	name: string;
	minSoc?: number;
	limitSoc?: number;
	plan?: StaticPlan;
	repeatingPlans: RepeatingPlan[];
	title: string;
	features?: string[];
	capacity?: number;
}

export type Timeout = ReturnType<typeof setInterval> | null;

export interface Tariff {
	rates: Rate[];
	lastUpdate: Date;
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

export interface Forecast {
	grid?: ForecastSlot[];
	co2?: ForecastSlot[];
	solar?: SolarDetails;
	planner?: ForecastSlot[];
}

export interface SelectOption<T> {
	name: string;
	value: T;
	count?: number;
	disabled?: boolean;
}

export type DeviceType = "charger" | "meter" | "vehicle";

// see https://stackoverflow.com/a/54178819
type Omit<T, K extends keyof T> = Pick<T, Exclude<keyof T, K>>;
export type PartialBy<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;

export type ValueOf<T> = T[keyof T];
