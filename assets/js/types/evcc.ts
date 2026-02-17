import type { StaticPlan, RepeatingPlan, PlanStrategy } from "../components/ChargingPlans/types";
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

export type AuthProviders = Record<string, { id: string; authenticated: boolean }>;

export interface MqttConfig {
  broker: string;
  topic: string;
}

export interface InfluxConfig {
  url: string;
  database: any;
  org: any;
}

export interface HemsConfig {
  type: string;
}

export interface HemsStatus {
  maxPower: number;
}

export type Hems = ConfigStatus<HemsConfig, HemsStatus>;

export interface ShmConfig {
  vendorId: string;
  deviceId: string;
}

export interface FatalError {
  error: string;
  class?: string;
  device?: string;
}

export interface State {
  offline: boolean;
  telemetry?: boolean;
  experimental?: boolean;
  setupRequired?: boolean;
  startupCompleted?: boolean;
  loadpoints: Loadpoint[];
  forecast: Forecast;
  currency?: CURRENCY;
  fatal?: FatalError[];
  authProviders?: AuthProviders;
  evopt?: EvOpt;
  version?: string;
  system?: string;
  timezone?: string;
  battery?: Battery;
  pv?: Meter[];
  aux?: Meter[];
  ext?: Meter[];
  tariffGrid?: number;
  tariffFeedIn?: number;
  tariffCo2?: number;
  tariffSolar?: number;
  mqtt?: MqttConfig;
  influx?: InfluxConfig;
  hems?: Hems;
  shm?: ShmConfig;
  sponsor?: Sponsor;
  eebus?: Eebus;
  modbusproxy?: ModbusProxy[];
  messaging?: ConfigStatus<unknown, unknown>;
  messagingEvents?: MessagingEvents;
  interval?: number;
  circuits?: Record<string, Circuit>;
  siteTitle?: string;
  vehicles: Record<string, Vehicle>;
  authDisabled?: boolean;
  config?: string;
  database?: string;
  ocpp?: Ocpp;
}

export interface ConfigStatus<C, S> {
  config?: C;
  status?: S;
  yamlSource?: YamlSource;
}

export type YamlSource = "file" | "db" | undefined;

export interface OcppConfig {
  port: number;
}

export interface OcppStatus {
  externalUrl?: string;
  stations: OcppStationStatus[];
}

export interface Ocpp {
  config: OcppConfig;
  status: OcppStatus;
}

export interface OcppStationStatus {
  id: string;
  status: OCPP_STATION_STATUS;
}

export enum OCPP_STATION_STATUS {
  UNKNOWN = "unknown",
  CONFIGURED = "configured",
  CONNECTED = "connected",
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

export enum ConfigType {
  Template = "template",
  Custom = "custom",
  Heatpump = "heatpump",
  SwitchSocket = "switchsocket",
  SgReady = "sgready",
  SgReadyRelay = "sgready-relay",
  SgReadyBoost = "sgready-boost", // deprecated
}

export type ConfigVehicle = Entity;
export type ConfigMessenger = Entity;

// Configuration-specific types for device setup/configuration contexts
export interface ConfigCharger extends Omit<Entity, "type"> {
  deviceProduct: string;
  type: ConfigType;
}

export interface ConfigMeter extends Entity {
  deviceProduct: string;
  deviceTitle?: string;
  deviceIcon?: string;
}

export type ConfigCircuit = Entity;

export interface LoadpointThreshold {
  delay: number;
  threshold: number;
}

export interface ConfigLoadpoint {
  id?: number;
  name?: string;
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
  planEnergy?: number;
  planTime?: string;
  planStrategy?: PlanStrategy;
  limitEnergy?: number;
  limitSoc?: number;
  circuit?: string;
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

export interface Loadpoint {
  batteryBoost: boolean;
  chargeCurrents?: number[];
  chargeDuration: number;
  chargePower: number;
  chargeRemainingDuration?: number;
  chargeRemainingEnergy?: number;
  chargeTotalImport?: number;
  chargeVoltages?: number[];
  chargedEnergy: number;
  chargerFeatureHeating: boolean;
  chargerFeatureIntegratedDevice: boolean;
  chargerIcon: string | null;
  chargerPhases1p3p: boolean;
  chargerSinglePhase: boolean;
  chargerStatusReason: CHARGER_STATUS_REASON | null;
  charging: boolean;
  connected: boolean;
  connectedDuration: number;
  disableDelay: number;
  disableThreshold: number;
  effectiveLimitSoc: number;
  effectiveMaxCurrent: number;
  effectiveMinCurrent: number;
  effectivePlanId: number;
  effectivePlanSoc: number;
  effectivePlanTime: string | null;
  effectivePlanStrategy: PlanStrategy;
  effectivePriority: number;
  enableDelay: number;
  enableThreshold: number;
  enabled: boolean;
  limitEnergy: number;
  limitSoc: number;
  maxCurrent: number;
  minCurrent: number;
  mode: CHARGE_MODE;
  offeredCurrent: number;
  phaseAction: PHASE_ACTION;
  phaseRemaining: number;
  phasesActive: number;
  phasesConfigured: number;
  planActive: boolean;
  planEnergy: number;
  planOverrun: number;
  planStrategy: PlanStrategy;
  planProjectedEnd: string | null;
  planProjectedStart: string | null;
  planTime: string | null;
  priority: number;
  pvAction: PV_ACTION;
  pvRemaining: number;
  sessionCo2PerKWh: number | null;
  sessionEnergy: number;
  sessionPrice: number | null;
  sessionPricePerKWh: number | null;
  sessionSolarPercentage: number;
  smartCostActive: boolean;
  smartCostLimit: number | null;
  smartCostNextStart: string | null;
  smartFeedInPriorityActive: boolean;
  smartFeedInPriorityLimit: number | null;
  smartFeedInPriorityNextStart: string | null;
  title: string;
  vehicleClimaterActive: boolean | null;
  vehicleDetectionActive: boolean;
  vehicleLimitSoc: number;
  vehicleName: string;
  vehicleOdometer: number;
  vehicleRange: number;
  vehicleSoc: number;
  vehicleTitle: string;
  vehicleWelcomeActive: boolean;
}

export interface UiLoadpoint extends Loadpoint {
  // Derived/computed fields for UI display
  id: string;
  displayTitle: string;
  icon: string;
  order: number | null;
  visible: boolean;
  lastSmartCostLimit: number | undefined;
  lastSmartFeedInPriorityLimit: number | undefined;
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

export enum PHASE_ACTION {
  INACTIVE = "inactive",
  SCALE_1P = "scale1p",
  SCALE_3P = "scale3p",
}

export enum PV_ACTION {
  INACTIVE = "inactive",
  ENABLE = "enable",
  DISABLE = "disable",
}

export enum CHARGER_STATUS_REASON {
  UNKNOWN = "unknown",
  WAITING_FOR_AUTHORIZATION = "waitingforauthorization",
  DISCONNECT_REQUIRED = "disconnectrequired",
}

export enum LOADPOINT_TYPE {
  CHARGING = "charging",
  HEATING = "heating",
}

export type LoadpointType = ValueOf<typeof LOADPOINT_TYPE>;

export type SessionInfoKey =
  | "remaining"
  | "finished"
  | "duration"
  | "solar"
  | "avgPrice"
  | "price"
  | "emission"
  | "co2";

export interface SponsorStatus {
  name?: string;
  expiresAt?: string;
  expiresSoon?: boolean;
  token?: string;
}

export type Sponsor = ConfigStatus<any, SponsorStatus>;

export type VehicleOption = {
  key?: string | null;
  name: string | null;
};

export enum MODBUS_BAUDRATE {
  _1200 = 1200,
  _9600 = 9600,
  _19200 = 19200,
  _38400 = 38400,
  _57600 = 57600,
  _115200 = 115200,
}

export enum MODBUS_TYPE {
  RS485_SERIAL = "rs485serial",
  RS485_TCPIP = "rs485tcpip",
  TCPIP = "tcpip",
}

export enum MODBUS_COMSET {
  _8N1 = "8N1",
  _8E1 = "8E1",
  _8N2 = "8N2",
}

export enum MODBUS_PROXY_READONLY {
  FALSE = "false",
  TRUE = "true",
  DENY = "deny",
}

export enum MODBUS_CONNECTION {
  TCPIP = "tcpip",
  SERIAL = "serial",
}

export enum MODBUS_PROTOCOL {
  TCP = "tcp",
  RTU = "rtu",
}

export type Certificate = {
  public: string;
  private: string;
};

export type Eebus = ConfigStatus<EebusConfig, EebusStatus>;

export type EebusConfig = {
  uri: string;
  port: number;
  shipid: string;
  interfaces?: string[];
  certificate?: Certificate;
};

export type EebusStatus = {
  ski: string;
};

export type ModbusProxy = {
  port: number;
  readonly: MODBUS_PROXY_READONLY;
  settings: ModbusProxySettings;
};

export type MessagingEvents = Record<MESSAGING_EVENTS, MessagingEvent>;

export enum MESSAGING_EVENTS {
  START = "start",
  STOP = "stop",
  CONNECT = "connect",
  DISCONNECT = "disconnect",
  SOC = "soc",
  GUEST = "guest",
  ASLEEP = "asleep",
  PLANOVERRUN = "planoverrun",
}

export interface MessagingEvent {
  title: string;
  msg: string;
  disabled: boolean;
}

export interface ModbusProxySettings {
  uri?: string;
  rtu?: boolean;
  device?: string;
  baudrate?: MODBUS_BAUDRATE;
  comset?: MODBUS_COMSET;
}

export interface Notification {
  message: string;
  time: Date;
  level: string;
  lp: number;
  count: number;
}

export interface Meter {
  power: number;
  title?: string;
  icon?: string;
  energy?: number;
}

export interface Battery {
  power: number;
  capacity: number;
  soc: number;
  devices: BatteryMeter[];
}

export interface BatteryMeter extends Meter {
  soc: number;
  controllable: boolean;
  capacity: number; // 0 when not specified
}

export interface Vehicle {
  name: string;
  minSoc?: number;
  limitSoc?: number;
  plan?: StaticPlan;
  repeatingPlans: RepeatingPlan[] | null;
  planStrategy: PlanStrategy;
  title: string;
  features?: string[];
  capacity?: number;
  icon?: string;
}

export type Timeout = ReturnType<typeof setInterval> | null;

export interface VehicleStatus {
  message: string;
  type?: string;
}

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
  start: Date;
  end: Date;
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

export type DeviceType = "charger" | "meter" | "vehicle" | "loadpoint" | "messenger";
export type MeterType = "grid" | "pv" | "battery" | "charge" | "aux" | "ext";
export type MeterTemplateUsage = "grid" | "pv" | "battery" | "charge" | "aux";

// see https://stackoverflow.com/a/54178819
type Omit<T, K extends keyof T> = Pick<T, Exclude<keyof T, K>>;
export type PartialBy<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;

export interface SiteConfig {
  grid: string;
  pv: string[];
  battery: string[];
  title: string;
  aux: string[] | null;
  ext: string[] | null;
}

export type ValueOf<T> = T[keyof T];

// EvOpt interfaces matching OpenAPI spec exactly
export interface EvOpt {
  req: OptimizationInput;
  res: OptimizationResult;
  details: OptimizationDetails;
}

// Request payload for /optimize/charge-schedule
export interface OptimizationInput {
  batteries: BatteryConfig[]; // Battery configurations
  time_series: TimeSeries; // Time series data
  eta_c?: number; // Charging efficiency (0-1), default 0.95
  eta_d?: number; // Discharging efficiency (0-1), default 0.95
  M?: number; // Big M value for MILP constraints
}

// Battery configuration
export interface BatteryConfig {
  s_min: number; // Min state of charge (Wh)
  s_max: number; // Max state of charge (Wh)
  s_initial: number; // Initial state of charge (Wh)
  c_min: number; // Min charge power (W)
  c_max: number; // Max charge power (W)
  d_max: number; // Max discharge power (W)
  p_a: number; // Energy value per Wh at end
  charge_from_grid?: boolean; // Can charge from grid
  discharge_to_grid?: boolean; // Can discharge to grid
  p_demand?: number[]; // Min charge demand per step (Wh)
  s_goal?: number[]; // Goal state of charge per step (Wh)
}

// Time series data
export interface TimeSeries {
  dt: number[]; // Duration per time step (seconds)
  gt: number[]; // Household demand per step (Wh)
  ft: number[]; // Energy generation forecast per step (Wh)
  p_N: number[]; // Grid import price per step (currency/Wh)
  p_E: number[]; // Grid export price per step (currency/Wh)
}

// Solver status enum
export enum OptimizationStatus {
  OPTIMAL = "Optimal",
  INFEASIBLE = "Infeasible",
  UNBOUNDED = "Unbounded",
  UNDEFINED = "Undefined",
  NOT_SOLVED = "Not Solved",
}

// Flow direction enum
export enum FlowDirection {
  IMPORT = 0, // Import from grid
  EXPORT = 1, // Export to grid
}

// Response from /optimize/charge-schedule
export interface OptimizationResult {
  status: OptimizationStatus; // Solver status
  objective_value: number | null; // Economic benefit (null if not optimal)
  batteries: BatteryResult[]; // Results per battery
  grid_import: number[]; // Grid import per step (Wh)
  grid_export: number[]; // Grid export per step (Wh)
  flow_direction: FlowDirection[]; // Flow direction per step (0=import, 1=export)
}

// Battery optimization results
export interface BatteryResult {
  charging_power: number[]; // Charging energy per step (Wh)
  discharging_power: number[]; // Discharging energy per step (Wh)
  state_of_charge: number[]; // State of charge per step (Wh)
}

// Battery detail information for optimization
export interface BatteryDetail {
  type: "vehicle" | "battery"; // Type of battery
  title: string; // Display title
  name: string; // Internal name/identifier
  capacity: number; // Battery capacity (kWh)
}

// Optimization details with timestamps and battery information
export interface OptimizationDetails {
  timestamp: string[]; // Array of ISO timestamp strings
  batteryDetails: BatteryDetail[]; // Array of battery detail objects
}

// Error response
export interface Error {
  message: string; // Error description
}
