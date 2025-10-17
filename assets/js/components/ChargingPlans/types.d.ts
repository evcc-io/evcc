import type { Rate } from "@/types/evcc";

export interface RepeatingPlan {
  weekdays: number[];
  time: string;
  tz: string; // timezone like "Europe/Berlin"
  soc: number;
  active: boolean;
  precondition: number;
  continuous?: boolean;
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
  data: PlanWrapper;
}

export type StaticPlan = StaticSocPlan | StaticEnergyPlan;

export interface StaticSocPlan {
  soc: number;
  time: Date;
  precondition: number;
  continuous?: boolean;
}

export interface StaticEnergyPlan {
  energy: number;
  time: Date;
  precondition: number;
  continuous?: boolean;
}
