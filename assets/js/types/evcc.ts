import type { StaticPlan, RepeatingPlan } from "../components/ChargingPlans/types";
import type { ForecastSlot, SolarDetails } from "../components/Forecast/types";

// react-native-webview
interface WebView {
  postMessage: (message: string) => void;
}

declare global {
  interface Window {
    app: any;
    evcc: {
      version: string;
      commit: string;
      customCss: string;
    };
  }
  interface Window {
    ReactNativeWebView?: WebView;
  }
}

export interface Auth {
  vehicles: VehicleLogins;
}

export type VehicleLogins = Record<string, { authenticated: boolean; uri: string }>;

export interface FatalError {
  error: string;
  class?: string;
  device?: string;
}

export interface State {
  offline: boolean;
  startup?: boolean;
  loadpoints: [];
  forecast?: Forecast;
  currency?: CURRENCY;
  fatal?: FatalError[];
  auth?: Auth;
  version?: string;
  battery?: Battery[];
  tariffGrid?: number;
  tariffFeedIn?: number;
  tariffCo2?: number;
  tariffSolar?: number;
  mqtt?: {
    broker: string;
    topic: string;
  };
  influx?: {
    url: string;
    database: any;
    org: any;
  };
  hems?: {
    type: any;
  };
  sponsor?: Sponsor;
  eebus?: any;
  modbusproxy?: [];
  messaging?: any;
  interval?: number;
  circuits?: Record<string, Circuit>;
  siteTitle?: string;
  vehicles: Record<string, Vehicle>;
}

export interface Config {
  template?: string;
  title?: string;
  icon?: string;
  [key: string]: number | string | undefined;
}

export interface Circuit {
  name: string;
  maxPower: number;
  power?: number;
  maxCurrent: number;
  current?: number;
  config?: Config;
}

export interface Entity {
  name: string;
  type: string;
  id: number;
  config: Config;
}

export interface Meter extends Entity {
  deviceTitle?: string;
  deviceIcon?: string;
}
export type ConfigVehicle = Entity;
export type Charger = Entity;

export interface LoadpointThreshold {
  delay: number;
  threshold: number;
}

export interface Loadpoint {
  id: number;
  name: string;
  charger: string;
  meter: string;
  vehicle: string;
  title: string;
  defaultMode: string;
  priority: number;
  phasesConfigured: number;
  minCurrent: number;
  maxCurrent: number;
  smartCostLimit: number | null;
  planEnergy: number;
  planTime: string;
  planPrecondition: number;
  limitEnergy: number;
  limitSoc: number;
  thresholds: {
    enable: LoadpointThreshold;
    disable: LoadpointThreshold;
  };
  soc: {
    poll: {
      mode: string;
      interval: number;
    };
    estimate: boolean;
  };
}

export enum SMART_COST_TYPE {
  CO2 = "co2",
  PRICE_DYNAMIC = "pricedynamic",
  PRICE_FORECAST = "priceforecast",
}

export enum LENGTH_UNIT {
  KM = "km",
  MILES = "mi",
}

export interface LoadpointCompact {
  icon: string;
  title: string;
  charging: boolean;
  soc?: number;
  power: number;
  heating?: boolean;
  chargePower: number;
  connected: boolean;
  index: number;
  vehicleName: string;
  chargerIcon?: string;
  vehicleSoc: number;
  chargerFeatureHeating: boolean;
}

export enum THEME {
  AUTO = "auto",
  LIGHT = "light",
  DARK = "dark",
}

export enum CURRENCY {
  AUD = "AUD",
  BGN = "BGN",
  BRL = "BRL",
  CAD = "CAD",
  CHF = "CHF",
  CNY = "CNY",
  EUR = "EUR",
  GBP = "GBP",
  ILS = "ILS",
  NZD = "NZD",
  PLN = "PLN",
  USD = "USD",
  DKK = "DKK",
  SEK = "SEK",
}

export enum ICON_SIZE {
  XS = "xs",
  S = "s",
  M = "m",
  L = "l",
  XL = "xl",
}

export enum CHARGE_MODE {
  OFF = "off",
  NOW = "now",
  MINPV = "minpv",
  PV = "pv",
}

export enum PHASES {
  AUTO = 0,
  ONE_PHASE = 1,
  TWO_PHASES = 2,
  THREE_PHASES = 3,
}

export enum LOADPOINT_TYPE {
  CHARGING = "charging",
  HEATING = "heating",
}

export type SessionInfoKey =
  | "remaining"
  | "finished"
  | "duration"
  | "solar"
  | "avgPrice"
  | "price"
  | "co2";

export interface Sponsor {
  name: string;
  expiresAt: string;
  expiresSoon: boolean;
}

export interface Notification {
  message: string;
  time: Date;
  level: string;
  lp: number;
  count: number;
}

export interface Battery {
  power: number;
  soc: number;
  controllable: boolean;
  capacity: number; // 0 when not specified
  title?: string;
}

export interface Vehicle {
  name: string;
  minSoc?: number;
  limitSoc?: number;
  plan?: StaticPlan;
  repeatingPlans: RepeatingPlan[];
  title: string;
  features?: string[];
  capacity?: number;
  icon?: string;
}

export type Timeout = ReturnType<typeof setInterval> | null;

export interface Tariff {
  rates: Rate[];
  lastUpdate: Date;
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

export interface Forecast {
  grid?: ForecastSlot[];
  co2?: ForecastSlot[];
  solar?: SolarDetails;
  planner?: ForecastSlot[];
  feedin?: ForecastSlot[];
}

export interface SelectOption<T> {
  name: string;
  value: T;
  count?: number;
  disabled?: boolean;
}

export type DeviceType = "charger" | "meter" | "vehicle";
export type SelectedMeterType = "grid" | "pv" | "battery" | "charge" | "aux";

// see https://stackoverflow.com/a/54178819
type Omit<T, K extends keyof T> = Pick<T, Exclude<keyof T, K>>;
export type PartialBy<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;

export type ValueOf<T> = T[keyof T];
