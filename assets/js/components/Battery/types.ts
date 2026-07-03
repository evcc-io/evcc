// Shared view-model types for the experimental battery page.

import type { BatteryForecast } from "@/types/evcc";

export interface SocPoint {
  t: number; // epoch ms
  soc: number; // 0..100
}

export interface BatterySeries {
  id: string;
  title: string;
  capacity: number; // kWh
  currentSoc: number; // 0..100
  history: SocPoint[]; // ascending t, <= now
  forecast: SocPoint[]; // ascending t, >= now (may be empty)
}

export interface BatterySuggestion {
  action: "normal" | "hold" | "charge" | "holdcharge";
}

export interface BatteryStatusCardModel {
  id: string;
  title: string;
  soc: number;
  power: number;
  capacity: number;
  color: string;
  suggestion: BatterySuggestion | null;
  forecast: BatteryForecast | null;
}

export interface BatteryHistorySlot {
  start: string;
  socTemp?: number | null;
}
export interface BatteryHistorySeries {
  title?: string;
  group: string;
  data: BatteryHistorySlot[];
}
