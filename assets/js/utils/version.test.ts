import { describe, expect, test } from "vite-plus/test";
import {
  isDevelopment,
  isNightly,
  commitFromVersion,
  getReleaseName,
  getShortVersion,
  isNewVersionAvailable,
} from "./version";

const DEV = "0.0.0";
const NIGHTLY = "0.304.0-dev+abc1234";
const STABLE = "0.303.1";

describe("isDevelopment", () => {
  test("only dev builds", () => {
    expect(isDevelopment(DEV)).toBe(true);
    expect(isDevelopment(NIGHTLY)).toBe(false);
    expect(isDevelopment(STABLE)).toBe(false);
  });
});

describe("isNightly", () => {
  test("only versions with build metadata", () => {
    expect(isNightly(DEV)).toBe(false);
    expect(isNightly(NIGHTLY)).toBe(true);
    expect(isNightly(STABLE)).toBe(false);
  });
});

describe("commitFromVersion", () => {
  test("extracts build metadata", () => {
    expect(commitFromVersion(NIGHTLY)).toBe("abc1234");
    expect(commitFromVersion(STABLE)).toBe("");
  });
});

describe("getReleaseName", () => {
  test("maps version to release name", () => {
    expect(getReleaseName(DEV)).toBe("development");
    expect(getReleaseName(NIGHTLY)).toBe("nightly");
    expect(getReleaseName(STABLE)).toBe("stable");
  });
});

describe("getShortVersion", () => {
  test("formats version", () => {
    expect(getShortVersion(DEV)).toBe("dev build");
    expect(getShortVersion(NIGHTLY)).toBe("v0.304.0-dev+abc1234");
    expect(getShortVersion(STABLE)).toBe("v0.303.1");
  });
});

describe("isNewVersionAvailable", () => {
  test("never for dev builds", () => {
    expect(isNewVersionAvailable(DEV, "0.303.1")).toBe(false);
  });
  test("only when available differs", () => {
    expect(isNewVersionAvailable(STABLE, "0.303.1")).toBe(false);
    expect(isNewVersionAvailable("0.303.0", "0.303.1")).toBe(true);
    expect(isNewVersionAvailable(STABLE, undefined)).toBe(false);
  });
});
