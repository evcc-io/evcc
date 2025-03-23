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
}

export interface Vehicle {
	name: string;
	minSoc?: number;
	limitSoc?: number;
	plan?: StaticPlan;
	repeatingPlans: RepeatingPlan[];
	title: string;
}

export interface PlanWrapper {
	planId: number;
	planTime: Date;
	duration: number;
	plan: Rate[];
	power: number;
}

export type StaticPlan = StaticSocPlan | StaticEnergyPlan;

export interface StaticSocPlan {
	soc: number;
	time: Date;
}

export interface StaticEnergyPlan {
	energy: number;
	time: Date;
}

export type Timeout = ReturnType<typeof setInterval> | null;

export interface RepeatingPlan {
	weekdays: number[];
	time: string;
	tz: string; // timezone like "Europe/Berlin"
	soc: number;
	active: boolean;
}
export interface Tariff {
	rates: Rate[];
	lastUpdate: Date;
}

export interface Rate {
	start: Date;
	end: Date;
	price: number;
}

// TODO: add comments to props
export interface CustomSelectOption {
	name: string;
	value: string | number;
	count?: number;
	disabled?: boolean;
}

// see https://stackoverflow.com/a/54178819
type Omit<T, K extends keyof T> = Pick<T, Exclude<keyof T, K>>;
export type PartialBy<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;

export type ValueOf<T> = T[keyof T];
