import type { StaticPlan, RepeatingPlan } from "../components/ChargingPlans/types";

declare global {
	interface Window {
		app: any;
	}
}

export interface State {
	offline: boolean;
	startup?: boolean;
	loadpoints: [];
	forecast?: any;
	currency?: CURRENCY;
	fatal?: {
		error: any;
	};
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
