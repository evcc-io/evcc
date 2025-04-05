import type { Rate } from "assets/js/types/evcc";

export interface RepeatingPlan {
	weekdays: number[];
	time: string;
	tz: string; // timezone like "Europe/Berlin"
	soc: number;
	active: boolean;
}

export interface PlanWrapper {
	planId: number;
	planTime: Date;
	duration: number;
	plan: Rate[];
	power: number;
}

export interface PlanResponse {
	status: number;
	data: { result: PlanWrapper };
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
