export interface LiveCommunityData {
  totalClients: number;
  activeClients: number;
  totalInstances: number;
  activeInstances: number;
  chargePower: number;
  greenPower: number;
  maxChargePower: number;
  maxGreenPower: number;
  chargeEnergy: number;
  greenEnergy: number;
}
export type Period = "30d" | "365d" | "thisYear" | "total";
