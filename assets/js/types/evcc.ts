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
    evccAppCapabilities?: string[];
  }
}

/** Status of configured vehicle authentication providers, keyed by provider name. */
export type AuthProviders = Record<string, { id: string; authenticated: boolean }>;

export type DeviceColors = Record<string, string>;
/** Color assigned to a device for consistent UI display. */
export type DeviceColorEntry = { title: string; color: string };

/** MQTT integration configuration. */
export interface MqttConfig {
  /** Broker address. */
  broker: string;
  /** Root topic all values are published under. */
  topic: string;
  /** Broker username. */
  user?: string;
  /** Broker password. Redacted. */
  password?: string;
  /** MQTT client id. */
  clientID?: string;
  /** Skip TLS certificate verification. */
  insecure?: boolean;
  /** CA certificate for TLS connections. */
  caCert?: string;
  /** Client certificate for TLS connections. */
  clientCert?: string;
  /** Client key for TLS connections. */
  clientKey?: string;
}

/** InfluxDB integration configuration. */
export interface InfluxConfig {
  /** InfluxDB server URL. */
  url: string;
  /** Database name (InfluxDB 1.x) or bucket (InfluxDB 2.x). */
  database: string;
  /** Organization (InfluxDB 2.x). */
  org: string;
  /** Access token (InfluxDB 2.x). Redacted. */
  token?: string;
  /** Username (InfluxDB 1.x). */
  user?: string;
  /** Password (InfluxDB 1.x). Redacted. */
  password?: string;
  /** Skip TLS certificate verification. */
  insecure?: boolean;
}

/** Network configuration. */
export interface Network {
  /** URL schema, http or https. */
  schema?: string;
  /** Hostname. */
  host: string;
  /** HTTP port. */
  port: number;
  /** Externally reachable URL. */
  externalUrl?: string;
  /** URL in the local network. */
  internalUrl?: string;
}

/** Home energy management system configuration. */
export interface HemsConfig {
  /** A home energy management system is configured. */
  configured: boolean;
}

/** Current external limits imposed by the home energy management system. */
export interface HemsStatus {
  /** Consumption is currently limited. */
  dimmed?: boolean;
  /** Allowed feed-in in %. Values below 100 mean production is curtailed. */
  curtailed?: number;
  /** Maximum allowed consumption power in W. */
  maxConsumptionPower?: number;
  /** Maximum allowed production power in W. */
  maxProductionPower?: number;
}

/** SMA Home Manager configuration. */
export interface ShmConfig {
  /** SMA vendor id. */
  vendorId: string;
  /** SMA device id. */
  deviceId: string;
  /** SMA device serial. */
  deviceSerial?: string;
}

/** A fatal startup error. */
export interface FatalError {
  /** Error message. */
  error: string;
  /** Device class the error originates from, like meter or charger. */
  class?: string;
  /** Name of the device that caused the error. */
  device?: string;
}

export type StatisticsPeriod = "30d" | "365d" | "thisYear" | "total";
export type StatisticsIndicator = "none" | "solar" | "price" | "savings" | "co2" | "co2saved";

/** Aggregated charging statistics for a time period. */
export interface StatisticsData {
  /** Average CO₂ emissions in g/kWh. */
  avgCo2: number;
  /** Average energy price per kWh in the configured currency. */
  avgPrice: number;
  /** Total charged energy in kWh. */
  chargedKWh: number;
  /** Share of solar energy in %. */
  solarPercentage: number;
}

/** Charging statistics, keyed by time period. */
export type Statistics = Record<StatisticsPeriod, StatisticsData>;

/**
 * Complete system state as returned by /api/state and pushed via websocket and MQTT.
 * This structure mirrors the internal UI state and carries no compatibility promise.
 * Fields may change or disappear between releases.
 */
export interface State {
  /** @internal */
  offline: boolean;
  /** Telemetry is enabled. */
  telemetry?: boolean;
  /** Experimental UI features are enabled. */
  experimental?: boolean;
  /** Initial setup is required. */
  setupRequired?: boolean;
  /** Startup has completed. */
  startupCompleted?: boolean;
  /** HTTP API is ready to accept requests. */
  apiReady?: boolean;
  /** Charging locations. One entry per configured loadpoint. */
  loadpoints: Loadpoint[];
  /** Price, CO₂ and solar production forecasts. */
  forecast: Forecast;
  /** Configured currency for all monetary values. */
  currency?: CURRENCY;
  /** Fatal startup errors. */
  fatal?: FatalError[];
  /** Status of configured vehicle authentication providers, keyed by provider name. */
  authProviders?: AuthProviders;
  /** @internal */
  evopt?: EvOpt;
  /** Running evcc version. */
  version?: string;
  /** Latest available evcc version. */
  availableVersion?: string;
  /**
   * Operating system and architecture.
   * @example "linux/amd64"
   */
  system?: string;
  /**
   * Server time zone abbreviation and UTC offset.
   * @example "CEST +02:00"
   */
  timezone?: string;
  /** Aggregated home battery state. */
  battery?: Battery;
  /** Current operation mode of the home battery. */
  batteryMode?: BATTERY_MODE;
  /** Grid meter data. */
  grid?: Meter;
  /** A grid meter is configured. */
  gridConfigured?: boolean;
  /** Solar generation meters. One entry per configured pv meter. */
  pv?: Meter[];
  /** Total solar generation power in W. Sum of all pv meters. */
  pvPower?: number;
  /** Total solar generation energy in kWh. */
  pvEnergy?: number;
  /** Auxiliary consumption meters. Their power is treated as available for charging. */
  aux?: Meter[];
  /** Total power of all auxiliary meters in W. */
  auxPower?: number;
  /** Additional meters for monitoring purposes. */
  ext?: Meter[];
  /** Consumption meters of individual devices for monitoring purposes. */
  consumers?: Meter[];
  /** Home consumption power in W. */
  homePower?: number;
  /** Configured grid operating point in W. Positive values maintain grid import. */
  residualPower?: number;
  /** Share of green energy in home consumption, between 0 and 1. */
  greenShareHome?: number;
  /** Share of green energy used for charging, between 0 and 1. */
  greenShareLoadpoints?: number;
  /** Configured tariffs. Structure depends on configuration. */
  tariffs?: GenericConfigStatus;
  /** Current grid energy price per kWh in the configured currency. */
  tariffGrid?: number;
  /** Current feed-in rate per kWh in the configured currency. */
  tariffFeedIn?: number;
  /** Current grid CO₂ emissions in g/kWh. */
  tariffCo2?: number;
  /** Current cost of solar generation per kWh in the configured currency. */
  tariffSolar?: number;
  /** Current energy price of home consumption per kWh, taking the green energy share into account. */
  tariffPriceHome?: number;
  /** Current energy price of charging per kWh, taking the green energy share into account. */
  tariffPriceLoadpoints?: number;
  /** Current CO₂ emissions of home consumption in g/kWh, taking the green energy share into account. */
  tariffCo2Home?: number;
  /** Current CO₂ emissions of charging in g/kWh, taking the green energy share into account. */
  tariffCo2Loadpoints?: number;
  tariffTemperature?: number;
  /** MQTT integration configuration. */
  mqtt?: MqttConfig;
  /** InfluxDB integration configuration. */
  influx?: InfluxConfig;
  /** Network configuration. */
  network?: Network;
  /** Home energy management system integration. */
  hems?: Hems;
  /** SMA Home Manager configuration. */
  shm?: ShmConfig;
  /** Sponsorship status. */
  sponsor?: Sponsor;
  /** EEBus integration configuration and status. */
  eebus?: Eebus;
  /** Remote access configuration and connection status. */
  remote?: Remote;
  /** Modbus proxy configuration. */
  modbusproxy?: ModbusProxy[] | null;
  /** Messaging services configuration. Structure depends on configuration. */
  messaging?: GenericConfigStatus;
  /** Configured notification messages per event type. */
  messagingEvents?: MessagingEvents | null;
  /** Update interval of the control loop in seconds. */
  interval?: number;
  /** Load management circuits, keyed by circuit name. */
  circuits?: Record<string, Circuit>;
  /** Battery buffer SoC in %. Energy above this level may be used for charging in solar mode. */
  bufferSoc?: number;
  /** Battery priority SoC in %. Home battery is charged first while below this level. */
  prioritySoc?: number;
  /** Battery buffer start SoC in %. Solar charging starts automatically above this level. */
  bufferStartSoc?: number;
  /** Home battery discharge is prevented during fast charging and planned charging. */
  batteryDischargeControl?: boolean;
  solarAdjusted?: boolean;
  /** Price or emission limit for charging the home battery from grid. */
  batteryGridChargeLimit?: number | null;
  /** Home battery is currently charged from grid. */
  batteryGridChargeActive?: boolean;
  /** A dynamic grid price or CO₂ forecast is configured. */
  smartCostAvailable?: boolean;
  /** Type of the smart charging limit, price based or emission based. */
  smartCostType?: SMART_COST_TYPE;
  /** A feed-in tariff with forecast is configured. */
  smartFeedInPriorityAvailable?: boolean;
  /**
   * Time of the last energy history update.
   * @format date-time
   */
  historyUpdated?: string;
  /** Configured site title. */
  siteTitle?: string;
  /** Colors assigned to devices for consistent UI display. */
  deviceColors?: DeviceColorEntry[];
  /** Configured vehicles, keyed by vehicle name. */
  vehicles: Record<string, Vehicle>;
  /** Charging statistics over different time periods. */
  statistics?: Statistics;
  /** Authentication is disabled. */
  authDisabled?: boolean;
  /** Path of the evcc configuration file. */
  config?: string;
  /** Path of the evcc database. */
  database?: string;
  /** OCPP server configuration and connected stations. */
  ocpp?: Ocpp;
  /** OCPP forwarder rules and upstream connection status. */
  ocppforwarder?: OcppForwarder;
  /** Battery optimizer is enabled. */
  optimizer?: boolean;
  /** Selected battery optimizer charging strategy. */
  optimizerChargingStrategy?: string;
  /** Available battery optimizer charging strategies. */
  optimizerChargingStrategies?: string[];
  /** Built-in MCP server is enabled. */
  mcp?: boolean;
  /** Instance runs in demo mode. */
  demoMode?: boolean;
}

/** Configuration and runtime status of an integration. */
export interface ConfigStatus<C, S> {
  /** Static configuration. */
  config?: C;
  /** Runtime status. */
  status?: S;
  /** Source of the configuration, evcc.yaml file or database. */
  yamlSource?: YamlSource;
}

/** Configuration and runtime status of an integration. Structure depends on configuration. */
export interface GenericConfigStatus extends ConfigStatus<unknown, unknown> {}

/** Home energy management system integration. */
export interface Hems extends ConfigStatus<HemsConfig, HemsStatus> {}

export type YamlSource = "file" | "db" | undefined;

/** OCPP server configuration. */
export interface OcppConfig {
  /** Port the OCPP server listens on. */
  port: number;
}

/** A rule forwarding an OCPP station to an upstream backend. */
export interface OcppForwarderRule {
  /** Station id to forward. */
  stationId: string;
  /** URL of the upstream OCPP backend. */
  upstreamUrl: string;
  /** Password for upstream authentication. Redacted. */
  password?: string;
  /** Station id used towards the upstream backend. */
  upstreamStationId?: string;
  /** Username for upstream authentication. */
  username?: string;
  /** Skip TLS certificate verification. */
  insecure?: boolean;
  /** CA certificate for TLS connections. */
  caCert?: string;
  /** Only forward read operations to the station. */
  readOnly?: boolean;
}

/** OCPP forwarder rules and upstream connection status. */
export interface OcppForwarder extends ConfigStatus<OcppForwarderRule[], OcppForwarderSession[]> {}

/** Connection status of an OCPP forwarder rule. */
export interface OcppForwarderSession {
  /** Charger id. */
  chargerId: string;
  /** URL of the upstream OCPP backend. */
  upstreamUrl: string;
  /** Connection to the upstream backend is established. */
  upstreamConnected: boolean;
  /** Last connection error. */
  error?: string;
}

/** OCPP server status. */
export interface OcppStatus {
  /** Externally reachable URL of the OCPP server. */
  externalUrl?: string;
  /** Connection status per configured station. */
  stations: OcppStationStatus[];
}

/** OCPP server configuration and status. */
export interface Ocpp {
  config: OcppConfig;
  status: OcppStatus;
}

/** Connection status of an OCPP station. */
export interface OcppStationStatus {
  /** Station id. */
  id: string;
  /** Connection status. */
  status: OCPP_STATION_STATUS;
}

/** Connection status of an OCPP station. */
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

/** A load management circuit limiting power and current of its assigned loadpoints. */
export interface Circuit {
  /** Circuit title for UI display. */
  title?: string;
  /** Circuit icon name for UI display. */
  icon?: string;
  /** Name of the parent circuit. */
  parent?: string;
  /** Current power in W. */
  power: number;
  /** Current in A. */
  current?: number;
  /** Maximum allowed power in W. */
  maxPower?: number;
  /** Maximum allowed current in A. */
  maxCurrent?: number;
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

export interface ConfigHems extends Entity {
  deviceProduct?: string;
}

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
  ui: LoadpointUi;
}

/** Display-only UI settings of a loadpoint. */
export interface LoadpointUi {
  /** Lower bound of the temperature scale for heating devices, in degrees. */
  minTemp: number;
  /** Upper bound of the temperature scale for heating devices, in degrees. */
  maxTemp: number;
}

/** Type of the smart charging limit. */
export enum SMART_COST_TYPE {
  CO2 = "co2",
  PRICE_DYNAMIC = "pricedynamic",
  PRICE_FORECAST = "priceforecast",
}

export enum LENGTH_UNIT {
  KM = "km",
  MILES = "mi",
}

export enum TIME_FORMAT {
  H12 = "12",
  H24 = "24",
}

/** A charging location with a charger, an optional dedicated meter and an optional vehicle. */
export interface Loadpoint {
  /** Unique loadpoint identifier used in API routes and configuration. */
  name: string;
  /** Battery boost is active. When enabled, home battery power is used for fast charging. */
  batteryBoost: boolean;
  /** Charging current per phase in A. */
  chargeCurrents?: number[];
  /** Duration of the current charging session in seconds. */
  chargeDuration: number;
  /** Current charging power in W. */
  chargePower: number;
  /** Estimated remaining charging time in seconds. */
  chargeRemainingDuration?: number;
  /** Estimated remaining energy in kWh. */
  chargeRemainingEnergy?: number;
  /** Total energy imported by the charge meter in kWh. */
  chargeTotalImport?: number;
  /** Charging voltage per phase in V. */
  chargeVoltages?: number[];
  /** Energy charged in the current charging session in kWh. */
  chargedEnergy: number;
  /** The connected vehicle is not identified automatically by its status. */
  chargerFeatureAutodetectDisabled: boolean;
  /** Values are averaged. */
  chargerFeatureAverage: boolean;
  /** Values are cached. */
  chargerFeatureCacheable: boolean;
  /** The climater state of the connected vehicle is ignored for charge control. */
  chargerFeatureClimaterDisabled: boolean;
  /** Charger only supports full ampere steps for current control. */
  chargerFeatureCoarseCurrent: boolean;
  /** Charger is a heating device where disabled means normal operation. */
  chargerFeatureContinuous: boolean;
  /** Charger is a heating device. SoC values represent temperature in degrees. */
  chargerFeatureHeating: boolean;
  /** Charger is an always-connected device without vehicles and charging sessions, like a heat pump. */
  chargerFeatureIntegratedDevice: boolean;
  /** The connected vehicle only provides data while charging. */
  chargerFeatureOffline: boolean;
  /** Vehicle API calls are retried on failure. */
  chargerFeatureRetryable: boolean;
  /** The connected vehicle pushes data updates instead of being polled. */
  chargerFeatureStreaming: boolean;
  /** Charger is a switchable device without current control, like a heat pump or switch socket. */
  chargerFeatureSwitchDevice: boolean;
  /** No wake-up calls are sent to the connected vehicle. */
  chargerFeatureWakeUpDisabled: boolean;
  /** The connected vehicle may need a short welcome charge to start communicating. */
  chargerFeatureWelcomeCharge: boolean;
  /** Charger icon name for UI display. */
  chargerIcon: string | null;
  /** Charger supports automatic switching between 1-phase and 3-phase charging. */
  chargerPhases1p3p: boolean;
  /** Charger is physically connected with a single phase. */
  chargerSinglePhase: boolean;
  /** Reason why the charger is waiting, like a pending authorization or a required disconnect. */
  chargerStatusReason?: CHARGER_STATUS_REASON | null;
  /** A vehicle is connected and charging. */
  charging: boolean;
  /** A vehicle is connected to the charger. */
  connected: boolean;
  /** Duration since the vehicle was connected, in seconds. */
  connectedDuration: number;
  /** Delay before charging stops in solar mode, in seconds. */
  disableDelay: number;
  /** Grid draw power above which charging stops in solar mode, in W. */
  disableThreshold: number;
  /** Currently applied SoC limit in %. */
  effectiveLimitSoc: number;
  /** Currently applied maximum charging current in A. */
  effectiveMaxCurrent: number;
  /** Currently applied minimum charging current in A. */
  effectiveMinCurrent: number;
  /** Currently applied minimum SoC in %. */
  effectiveMinSoc: number;
  /** Id of the currently applied charging plan. Zero if no plan is active. */
  effectivePlanId: number;
  /** SoC goal in % of the currently applied charging plan. */
  effectivePlanSoc: number;
  /**
   * Target time of the currently applied charging plan.
   * @format date-time
   */
  effectivePlanTime: string | null;
  /** Currently applied charging plan strategy. */
  effectivePlanStrategy: PlanStrategy;
  /** Currently applied priority. */
  effectivePriority: number;
  /** Delay before charging starts in solar mode, in seconds. */
  enableDelay: number;
  /** Available surplus power above which charging starts in solar mode, in W. */
  enableThreshold: number;
  /** Charger is currently allowed to charge. */
  enabled: boolean;
  /** Session energy limit in kWh. Zero means no limit. */
  limitEnergy: number;
  /** Session SoC limit in %. Zero means no limit. */
  limitSoc: number;
  /** Maximum charging current in A. */
  maxCurrent: number;
  /** Minimum charging current in A. */
  minCurrent: number;
  /** Minimum SoC in %. Vehicle is fast-charged until this level is reached. */
  minSoc: number;
  /** Vehicle SoC is below the configured minimum SoC. */
  minSocNotReached: boolean;
  /** Charging behavior. */
  mode: CHARGE_MODE;
  /** Current offered to the vehicle in A. */
  offeredCurrent: number;
  /** Pending phase switching action in solar mode. */
  phaseAction: PHASE_ACTION;
  /** Remaining time until the pending phase switching action executes, in seconds. */
  phaseRemaining: number;
  /** Number of phases expected to be used for charging. */
  phasesActive: number;
  /** Configured phase mode. 0 is automatic switching, 1 and 3 select a fixed phase count. */
  phasesConfigured: number;
  /** Time slots of the current charging plan. */
  plan: Rate[] | null;
  /** The current time slot is an active charging slot of the plan. */
  planActive: boolean;
  /** Energy goal of the charging plan in kWh. */
  planEnergy: number;
  /** Duration in seconds the charging plan is projected to miss its target time. */
  planOverrun: number;
  /** Charging plan strategy. */
  planStrategy: PlanStrategy;
  /**
   * Projected end of the charging plan. End of the last planned slot.
   * @format date-time
   */
  planProjectedEnd: string | null;
  /**
   * Projected start of the charging plan. Start of the earliest planned slot.
   * @format date-time
   */
  planProjectedStart: string | null;
  /**
   * Target time of the charging plan.
   * @format date-time
   */
  planTime: string | null;
  /** Priority for surplus distribution. Higher number means higher priority. */
  priority: number;
  /** Pending charge start or stop action in solar mode. */
  pvAction: PV_ACTION;
  /** Remaining time until the pending charge start or stop action executes, in seconds. */
  pvRemaining: number;
  last24hEnergy?: number;
  last7dEnergy?: number;
  /** Average CO₂ emissions of the current charging session in g/kWh. */
  sessionCo2PerKWh: number | null;
  /** Energy charged in the current charging session in kWh. */
  sessionEnergy: number;
  /** Total cost of the current charging session in the configured currency. */
  sessionPrice: number | null;
  /** Average energy price of the current charging session per kWh in the configured currency. */
  sessionPricePerKWh: number | null;
  /** Share of solar energy in the current charging session in %. */
  sessionSolarPercentage: number;
  /** Fast charging with cheap or clean grid energy is currently active. */
  smartCostActive: boolean;
  /** Price or emission limit for fast charging with grid energy. */
  smartCostLimit: number | null;
  /**
   * Start of the next fast charging period with cheap or clean grid energy.
   * @format date-time
   */
  smartCostNextStart: string | null;
  /** Charging pause for prioritized feed-in is currently active. */
  smartFeedInPriorityActive: boolean;
  /** Feed-in rate limit above which charging is paused. */
  smartFeedInPriorityLimit: number | null;
  /**
   * Start of the next charging pause for prioritized feed-in.
   * @format date-time
   */
  smartFeedInPriorityNextStart: string | null;
  /** Charging suggestion from the battery optimizer. */
  suggestion?: LoadpointSuggestion | null;
  /** Loadpoint title for UI display. */
  title: string;
  todayEnergy?: number;
  /** Climater of the connected vehicle is active. */
  vehicleClimaterActive: boolean | null;
  /** Automatic vehicle detection is running. */
  vehicleDetectionActive: boolean;
  /** SoC limit configured in the vehicle itself, in %. */
  vehicleLimitSoc: number;
  /** Unique name of the connected vehicle. Empty for guest vehicles. */
  vehicleName: string;
  /** Odometer reading of the connected vehicle in km. */
  vehicleOdometer: number;
  /** Range of the connected vehicle in km. */
  vehicleRange: number;
  /** Charge level of the connected vehicle in %. Temperature in degrees for heating devices. */
  vehicleSoc: number;
  /** Title of the connected vehicle for UI display. */
  vehicleTitle: string;
  /** A short welcome charge is active to enable vehicle communication. */
  vehicleWelcomeActive: boolean;
  /** SoC limit for battery boost in %. */
  batteryBoostLimit: number;
  /** Display-only UI settings. */
  ui?: LoadpointUi;
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
  range: number;
  vehicleRange: number;
  vehicleSoc: number;
  capacity: number;
  vehicleKnown: boolean;
  vehicleHasSoc: boolean;
  socBasedCharging: boolean;
  socBasedPlanning: boolean;
  sessionInfo: SessionInfoKey | undefined;
  rangePerSoc: number | undefined;
  socPerKwh: number;
  vehicleNotReachable: boolean;
}

export enum THEME {
  AUTO = "auto",
  LIGHT = "light",
  DARK = "dark",
}

/** Currency in ISO 4217 format. */
export enum CURRENCY {
  AUD = "AUD",
  BGN = "BGN",
  BRL = "BRL",
  CAD = "CAD",
  CHF = "CHF",
  CNY = "CNY",
  CZK = "CZK",
  EUR = "EUR",
  GBP = "GBP",
  HUF = "HUF",
  ILS = "ILS",
  JPY = "JPY",
  NZD = "NZD",
  NOK = "NOK",
  PLN = "PLN",
  RON = "RON",
  USD = "USD",
  DKK = "DKK",
  SEK = "SEK",
  ZAR = "ZAR",
  TRY = "TRY",
  MYR = "MYR",
}

export enum ICON_SIZE {
  XS = "xs",
  S = "s",
  M = "m",
  L = "l",
  XL = "xl",
}

/** Charging mode. */
export enum CHARGE_MODE {
  OFF = "off",
  NOW = "now",
  MINPV = "minpv",
  PV = "pv",
}

/** Battery operation mode. */
export enum BATTERY_MODE {
  UNKNOWN = "unknown",
  NORMAL = "normal",
  HOLD = "hold",
  CHARGE = "charge",
  HOLDCHARGE = "holdcharge",
}

export enum PHASES {
  AUTO = 0,
  ONE_PHASE = 1,
  TWO_PHASES = 2,
  THREE_PHASES = 3,
}

/** Pending phase switching action in solar mode. */
export enum PHASE_ACTION {
  INACTIVE = "inactive",
  SCALE_1P = "scale1p",
  SCALE_3P = "scale3p",
}

/** Pending charge start or stop action in solar mode. */
export enum PV_ACTION {
  INACTIVE = "inactive",
  ENABLE = "enable",
  DISABLE = "disable",
}

/** Reason why the charger is waiting. */
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
  | "co2"
  | "last24hEnergy"
  | "last7dEnergy";

/** Sponsorship status. */
export interface SponsorStatus {
  /** Name of the sponsor. */
  name?: string;
  /**
   * Expiry time of the sponsor token.
   * @format date-time
   */
  expiresAt?: string;
  /** Sponsor token expires soon. */
  expiresSoon?: boolean;
  /** Sponsor token. Redacted. */
  token?: string;
}

/** Sponsorship status. */
export interface Sponsor extends ConfigStatus<unknown, SponsorStatus> {}

export type VehicleOption = {
  key?: string | null;
  name: string | null;
};

/** Serial baud rate. */
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

/** Serial communication parameters. */
export enum MODBUS_COMSET {
  _8N1 = "8N1",
  _8E1 = "8E1",
  _8N2 = "8N2",
}

/** How the Modbus proxy handles write requests. */
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

/** A TLS certificate key pair. */
export type Certificate = {
  /** Public certificate in PEM format. */
  public: string;
  /** Private key in PEM format. Redacted. */
  private: string;
};

/** Remote access configuration and connection status. */
export interface Remote extends ConfigStatus<RemoteConfig, RemoteStatus> {}

/** Remote access configuration. */
export type RemoteConfig = {
  /** Remote access is enabled. */
  enabled: boolean;
};

/** Remote access connection status. */
export type RemoteStatus = {
  /** Connection to the remote access server is established. */
  connected: boolean;
  /** Remote access URL. */
  url?: string;
  /** Remote login attempts are currently blocked by the rate limiter. */
  loginBlocked: boolean;
  /** Last remote activity per client, keyed by username. RFC3339 timestamps. */
  lastSeen?: Record<string, string>;
};

export type RemoteClient = {
  username: string;
  createdAt: string;
  expiresAt?: string;
};

export type RemoteClientCreated = RemoteClient & {
  password: string;
};

/** EEBus integration configuration and status. */
export interface Eebus extends ConfigStatus<EebusConfig, EebusStatus> {}

/** EEBus integration configuration. */
export type EebusConfig = {
  /** URI the EEBus service listens on. */
  uri?: string;
  /** Port the EEBus service listens on. */
  port: number;
  /** SHIP id. */
  shipid: string;
  /** Network interfaces the EEBus service uses. */
  interfaces?: string[];
  /** TLS certificate. */
  certificate?: Certificate;
};

/** EEBus runtime status. */
export type EebusStatus = {
  /** SKI (subject key identifier) used for pairing. */
  ski: string;
  /** QR code payload for pairing. */
  qr?: string;
};

export type EebusPairing = {
  ski: string;
  shipID: string;
  source: "paired" | "ski";
};

/** A Modbus proxy forwarding requests to a physical device. */
export type ModbusProxy = {
  /** Port the proxy listens on. */
  port: number;
  /** How write requests are handled. */
  readonly: MODBUS_PROXY_READONLY;
  /** Connection settings of the physical device. */
  settings: ModbusProxySettings;
};

/** Configured notification messages, keyed by event type. */
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
  SUGGESTION = "suggestion",
}

/** A configured notification message. */
export interface MessagingEvent {
  /** Message title template. */
  title: string;
  /** Message body template. */
  msg: string;
  /** Notifications for this event are disabled. */
  disabled: boolean;
}

/** Connection settings of a Modbus device. */
export interface ModbusProxySettings {
  /** Device address for TCP connections. */
  uri?: string;
  /** Use RTU protocol over TCP. */
  rtu?: boolean;
  /** Serial device path. */
  device?: string;
  /** Serial baud rate. */
  baudrate?: MODBUS_BAUDRATE;
  /** Serial communication parameters. */
  comset?: MODBUS_COMSET;
}

export interface Notification {
  message: string;
  time: Date;
  level: string;
  lp: number;
  count: number;
}

/** Measurement data of a single meter. */
export interface Meter {
  /** Unique device name of the meter. */
  name?: string;
  /** Current power in W. */
  power: number;
  /** Meter title for UI display. */
  title?: string;
  /** Meter icon name for UI display. */
  icon?: string;
  /** Total energy in kWh. */
  energy?: number;
  /** Total exported energy in kWh. */
  returnEnergy?: number;
  /** Power per phase in W. */
  powers?: number[];
  /** Current per phase in A. */
  currents?: number[];
}

/** Projected home battery charge levels based on the solar and price forecast. */
export interface BatteryForecast {
  /** Projected highest charge level. */
  highest?: BatteryForecastPoint;
  /** Projected lowest charge level. */
  lowest?: BatteryForecastPoint;
}

/** A projected home battery charge level at a point in time. */
export interface BatteryForecastPoint {
  /** Projected charge level in %. */
  soc: number;
  /**
   * Time of the projected charge level.
   * @format date-time
   */
  time: string;
  /** The projection hits the upper or lower battery limit. */
  limit?: boolean;
}

/** Home battery state. Aggregated across all battery meters. */
export interface Battery {
  /** Current battery power in W. Positive means discharging, negative means charging. */
  power: number;
  /** Total battery capacity in kWh. */
  capacity: number;
  /** Charge level in %. Weighted by capacity across all batteries. */
  soc: number;
  /** Total battery energy in kWh. */
  energy?: number;
  /** Measurement data per battery meter. */
  devices?: BatteryMeter[];
  /** Projected charge levels based on the solar and price forecast. */
  forecast?: BatteryForecast;
}

/** Battery optimizer suggestion for a home battery. */
export interface BatterySuggestion {
  /** Suggested operation mode. */
  action: "normal" | "hold" | "charge" | "holdcharge";
  /** Recommended charge power in W. */
  charge?: number;
  /** Recommended discharge power in W. */
  discharge?: number;
  /** Suggestion differs from the current operating mode. */
  actionable?: boolean;
}

/** Battery optimizer suggestion for a loadpoint. */
export interface LoadpointSuggestion {
  /** Suggested charging action. */
  action: "charge" | "stop";
  /** Recommended charge power in W. */
  charge?: number;
  /** Recommended discharge power in W. */
  discharge?: number;
  /** Suggestion differs from the current operating mode. */
  actionable?: boolean;
}

/** Measurement data of a single home battery meter. */
export interface BatteryMeter extends Meter {
  /** Charge level in %. */
  soc: number;
  /** Battery supports external control of its operation mode. */
  controllable: boolean;
  /** Battery capacity in kWh. Zero when not specified. */
  capacity: number;
  /** Battery optimizer suggestion. */
  suggestion?: BatterySuggestion;
}

/** A repeating charging plan. */
export interface RepeatingPlan {
  /** Weekdays the plan is active. 0 is Sunday, 6 is Saturday. */
  weekdays: number[];
  /** Target time in HH:MM format. */
  time: string;
  /** Time zone of the target time, like Europe/Berlin. */
  tz: string;
  /** SoC goal in %. */
  soc: number;
  /** Plan is active. */
  active: boolean;
}

export interface PlanWrapper {
  planId: number;
  planTime: Date;
  duration: number;
  plan: Rate[] | null;
  power: number;
}

export interface PlanResponse {
  status: number;
  data: PlanWrapper;
}

/** Charging plan with a fixed target time and a SoC or energy goal. */
export type StaticPlan = StaticSocPlan | StaticEnergyPlan;

/** Charging plan with a SoC goal. */
export interface StaticSocPlan {
  /** SoC goal in %. */
  soc: number;
  /** Target time. */
  time: Date;
}

/** Charging plan with an energy goal. */
export interface StaticEnergyPlan {
  /** Energy goal in kWh. */
  energy: number;
  /** Target time. */
  time: Date;
}

/** Charging plan strategy. */
export interface PlanStrategy {
  /** Force continuous charging without gaps. */
  continuous: boolean;
  /** Duration of uninterrupted charging directly before the target time, in seconds. */
  precondition: number;
}

/** A configured vehicle. */
export interface Vehicle {
  /** Unique vehicle name used in API routes and configuration. */
  name?: string;
  /** Charge mode applied when the vehicle connects. */
  mode?: CHARGE_MODE | "";
  /** Minimum SoC in %. Vehicle is fast-charged until this level is reached. */
  minSoc?: number;
  /** SoC limit in %. Charging stops when reached. */
  limitSoc?: number;
  /** Charging plan with a fixed target time and SoC or energy goal. */
  plan?: StaticPlan;
  /** Repeating charging plans. */
  repeatingPlans: RepeatingPlan[] | null;
  /** Charging plan strategy. */
  planStrategy: PlanStrategy;
  /** Vehicle title for UI display. */
  title: string;
  /** Feature flags of the vehicle implementation. */
  features?: string[];
  /** Usable battery capacity in kWh. */
  capacity?: number;
  /** Vehicle icon name for UI display. */
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

/** A time slot with an associated price or emission value. */
export interface Rate {
  /** Start of the time slot. */
  start: Date;
  /** End of the time slot. */
  end: Date;
  /** Price per kWh in the configured currency or emissions in g/kWh. */
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

/** A forecast value at a point in time. */
export interface TimeseriesEntry {
  /** Forecast power in W. */
  val: number;
  /**
   * Time of the forecast value.
   * @format date-time
   */
  ts: string;
}

/** A forecast value for a time slot. */
export interface ForecastSlot {
  /**
   * Start of the time slot.
   * @format date-time
   */
  start: string;
  /**
   * End of the time slot.
   * @format date-time
   */
  end: string;
  /** Forecast value of the time slot. Unit depends on the forecast type. */
  value: number;
}

/** Expected solar production energy of a day. */
export interface EnergyByDay {
  /** Expected production energy in kWh. */
  energy: number;
  /** Forecast data covers the whole day. */
  complete: boolean;
}

/** Solar production forecast. */
export interface SolarDetails {
  /** Correction factor applied to the forecast based on past production. */
  scale?: number;
  /** Expected production today. */
  today?: EnergyByDay;
  /** Expected production tomorrow. */
  tomorrow?: EnergyByDay;
  /** Expected production the day after tomorrow. */
  dayAfterTomorrow?: EnergyByDay;
  /** Expected production power over time. */
  timeseries?: TimeseriesEntry[];
}

/** Price, CO₂ and solar production forecasts. */
export interface Forecast {
  /** Grid price forecast. Price per kWh in the configured currency per time slot. */
  grid?: ForecastSlot[];
  /** CO₂ emission forecast in g/kWh per time slot. */
  co2?: ForecastSlot[];
  /** Solar production forecast. */
  solar?: SolarDetails;
  /** Charging cost forecast used by the plan optimizer per time slot. */
  planner?: ForecastSlot[];
  /** Feed-in rate forecast. Rate per kWh in the configured currency per time slot. */
  feedin?: ForecastSlot[];
  temperature?: ForecastSlot[];
}

export interface SelectOption<T> {
  name: string;
  value: T;
  count?: number;
  disabled?: boolean;
}

export type DeviceType =
  | "charger"
  | "meter"
  | "vehicle"
  | "loadpoint"
  | "messenger"
  | "tariff"
  | "hems";
export type MeterType = "grid" | "pv" | "battery" | "charge" | "aux" | "ext" | "consumer";
export type MeterTemplateUsage = "grid" | "pv" | "battery" | "charge" | "aux";
export type TariffType = "grid" | "feedIn" | "co2" | "planner" | "solar" | "temperature";

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
  consumer: string[] | null;
}

export type ValueOf<T> = T[keyof T];

// EvOpt interfaces matching OpenAPI spec exactly
export interface EvOpt {
  updated: string;
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

// Tariff zone configuration
export interface Zone {
  price: number | null;
  days: string;
  months: string;
  hours: string;
}
