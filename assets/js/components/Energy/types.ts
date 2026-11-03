export type FlowSource = "pv" | "battery" | "grid";
export type FlowSink = "home" | "battery" | "loadpoint" | "export";

export interface Flow {
  from: FlowSource;
  to: FlowSink;
  energy: number; // kWh
  cost?: number; // home and loadpoint only
  pricedEnergy?: number; // kWh with a price
}

export interface FlowCost {
  import: number;
  export: number;
  importEnergy: number; // kWh with a price
  exportEnergy: number; // kWh with a price
  avgGrid: number;
}

export interface FlowCo2 {
  consumption: number; // kg, grid share to home and loadpoints
  consumptionEnergy: number; // kWh
  baseline: number; // kg
  avgCo2: number; // g/kWh
}

// one consumer or loadpoint: its sink's split per slot, scaled to its energy and priced per slot
export interface EntityFlows {
  group: "consumer" | "loadpoint";
  title: string;
  virtual?: boolean; // home minus the tracked consumers
  flows: Flow[];
}

export interface FlowResult {
  flows: Flow[];
  entities?: EntityFlows[];
  cost?: FlowCost;
  co2?: FlowCo2;
}

// key of the virtual consumer: home minus the tracked consumers
export const OTHERS = "others";

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
  cost?: (number | null)[]; // what the category's energy cost or earned
}

export interface PriceOverlay {
  import?: PriceBand;
  feedin?: PriceBand;
}

export interface ConsumerEnergy {
  title: string;
  energy: number; // kWh
  cost?: number; // undefined without price data
}

// loadpoint and consumer charts
export enum ENTITY_CHART {
  BARS = "bars",
  PATTERN = "pattern",
}

// the consumers card: shares of the period, or the consumers over time
export enum CONSUMER_VIEW {
  TREEMAP = "treemap",
  RIVER = "river",
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
