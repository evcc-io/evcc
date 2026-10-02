export type FlowSource = "pv" | "battery" | "grid";
export type FlowSink = "home" | "battery" | "loadpoint" | "export";

export interface Flow {
  from: FlowSource;
  to: FlowSink;
  energy: number; // kWh
}

export interface FlowCost {
  import: number;
  export: number;
  importEnergy: number; // kWh priced
  exportEnergy: number; // kWh priced
  consumption: number; // grid share at grid price, self-produced share at feed-in price
  consumptionEnergy: number; // kWh priced
  home: number; // share of consumption cost for home only
  homeEnergy: number; // kWh priced
  baseline: number;
  avgGrid: number;
}

export interface FlowCo2 {
  consumption: number; // kg, grid share to home and loadpoints
  consumptionEnergy: number; // kWh
  baseline: number; // kg
  avgCo2: number; // g/kWh
}

export interface FlowResult {
  flows: Flow[];
  cost?: FlowCost;
  co2?: FlowCo2;
}

export interface TariffSlot {
  start: string;
  grid?: number; // bucket average when aggregated
  gridMin?: number;
  gridMax?: number;
  feedin?: number;
  feedinMin?: number;
  feedinMax?: number;
  co2?: number;
  temperature?: number;
}

// price per chart category: bucket average with the slot range
export interface PriceBand {
  avg: (number | null)[];
  lo: (number | null)[];
  hi: (number | null)[];
}

export interface PriceOverlay {
  import?: PriceBand;
  feedin?: PriceBand;
}

export interface ConsumerEnergy {
  title: string;
  energy: number; // kWh
}

// loadpoint and consumer charts
export enum ENTITY_CHART {
  BARS = "bars",
  PATTERN = "pattern",
}

// the overview card
export enum OVERVIEW_VIEW {
  OVERVIEW = "overview",
  USAGE = "usage",
  FLOW = "flow",
}

// one Stat card, a slot named after the key renders below the number
export interface StatItem {
  key: string;
  label?: string;
  number?: number; // animated, formatted with `format`
  format?: (n: number) => string;
  sub?: string;
  tooltip?: string;
  accent?: string; // text color class for the value
  empty?: string; // shown instead of a value, with a link to the tariff settings
}
