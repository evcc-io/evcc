import type { StaticPlan, RepeatingPlan } from "../components/ChargingPlans/types";

declare global {
	interface Window {
		app: any;
	}
}

export interface State {
	offline: boolean;
	loadpoints: [];
	forecast?: any;
	currency?: CURRENCY;
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

export interface Rate {
	start: Date;
	end: Date;
	price: number;
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
