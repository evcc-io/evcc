export const LOG_LEVELS = ["fatal", "error", "warn", "info", "debug", "trace"] as const;
export const DEFAULT_LOG_LEVEL = "debug";

export type LogLevel = (typeof LOG_LEVELS)[number];

export interface LogEntry {
  time: string;
  area: string;
  level: LogLevel;
  message: string;
  attrs?: Record<string, string>;
}

const pad = (n: number) => String(n).padStart(2, "0");

// render a structured log entry in the classic console line format
export function formatLogEntry(entry: LogEntry): string {
  const d = new Date(entry.time);
  const ts = `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
  const attrs = Object.entries(entry.attrs || {})
    .map(([k, v]) => ` ${k}=${/[\s"=]/.test(v) ? JSON.stringify(v) : v}`)
    .join("");
  return `[${entry.area.padEnd(6)}] ${entry.level.toUpperCase()} ${ts} ${entry.message}${attrs}`;
}
