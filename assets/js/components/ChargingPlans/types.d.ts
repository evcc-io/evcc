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

export type StaticPlan = StaticSocPlan | StaticEnergyPlan;

export interface StaticSocPlan {
	soc: number;
	time: Date;
}

export interface StaticEnergyPlan {
	energy: number;
	time: Date;
}
