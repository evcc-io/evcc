import { describe, expect, test } from "vitest";
import { formatLogEntry, type LogEntry } from "./log";

describe("formatLogEntry", () => {
  test("renders classic line format", () => {
    const entry: LogEntry = {
      time: "2026-07-21T10:00:00+02:00",
      area: "lp-1",
      level: "warn",
      message: "hello",
    };
    expect(formatLogEntry(entry)).toMatch(
      /^\[lp-1 {2}\] WARN \d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2} hello$/
    );
  });

  test("appends attributes, quoting values with spaces", () => {
    const entry: LogEntry = {
      time: "2026-07-21T10:00:00+02:00",
      area: "site",
      level: "debug",
      message: "msg",
      attrs: { component: "loadpoint", title: "Garage 1" },
    };
    expect(formatLogEntry(entry)).toMatch(/ msg component=loadpoint title="Garage 1"$/);
  });
});
