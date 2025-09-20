export const LOG_LEVELS = ["fatal", "error", "warn", "info", "debug", "trace"] as const;
export const DEFAULT_LOG_LEVEL = "debug";

export type LogLevel = (typeof LOG_LEVELS)[number];
