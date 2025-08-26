import { describe, expect, test } from "vitest";
import { generateTariffLimitOptions } from "./tariffOptions";

describe("generateTariffLimitOptions", () => {
  const mockFormatValue = (value: number) => `${value.toFixed(3)} €/kWh`;

  const getValues = (values: number[], config = {}) => {
    const result = generateTariffLimitOptions(values, {
      formatValue: mockFormatValue,
      ...config,
    });
    return result.map((option) => option.value);
  };

  const expectRange = (values: number[], expectedStart: number[], expectedEnd: number[]) => {
    expect(values.slice(0, 3)).toEqual(expectedStart);
    expect(values.slice(-3)).toEqual(expectedEnd);
  };

  test("returns single option with value 0 for empty values", () => {
    const options = generateTariffLimitOptions([], {
      formatValue: mockFormatValue,
    });
    expect(options).toEqual([{ value: 0, name: "≤ 0.000 €/kWh" }]);
  });

  test("selectedValue: includes custom selected value", () => {
    const values = getValues([0.1, 0.2], { selectedValue: 0.999 });
    expectRange(values, [-0.05, -0.045, -0.04], [0.2, 0.205, 0.999]);
  });

  test("includeNegatives: includes negative range by default", () => {
    const values = getValues([0.1, 0.2]);
    expectRange(values, [-0.05, -0.045, -0.04], [0.195, 0.2, 0.205]);
  });

  test("includeNegatives: excludes negative values when false", () => {
    const values = getValues([0.1, 0.2], { includeNegatives: false });
    expectRange(values, [0, 0.005, 0.01], [0.195, 0.2, 0.205]);
  });

  test("extraHigh: extends range above maximum", () => {
    const values = getValues([0.1, 0.2], { extraHigh: true });
    expectRange(values, [-0.05, -0.045, -0.04], [0.245, 0.25, 0.255]);
  });

  test("extraLow: extends range below minimum", () => {
    const values = getValues([0.1, 0.2], { extraLow: true });
    expectRange(values, [-0.095, -0.09, -0.085], [0.195, 0.2, 0.205]);
  });

  test("extraLow: handles negative input values", () => {
    const values = getValues([-0.05, 0.02, 0.08], { extraLow: true });
    expectRange(values, [-0.114, -0.112, -0.11], [0.078, 0.08, 0.082]);
  });

  test("rounds values to 3 decimal places", () => {
    const values = getValues([0.1234567, 0.2345678]);
    values.forEach((value) => {
      const rounded = Math.round(value * 1000) / 1000;
      expect(value).toBe(rounded);
    });
  });

  test("stepSize: fine steps for small decimal ranges", () => {
    const values = getValues([0.23, 0.27, 0.31, 0.38]);
    expectRange(values, [-0.05, -0.045, -0.04], [0.375, 0.38, 0.385]);
  });

  test("stepSize: medium steps for medium decimal ranges", () => {
    const values = getValues([0.12, 0.18, 0.24, 0.29]);
    expectRange(values, [-0.05, -0.045, -0.04], [0.28, 0.285, 0.29]);
  });

  test("stepSize: coarse steps for large integer ranges", () => {
    const values = getValues([22, 28, 33, 37]);
    expectRange(values, [-5, -4.5, -4], [36.5, 37, 37.5]);
  });

  test("stepSize: very fine steps for micro decimal ranges", () => {
    const values = getValues([0.0013, 0.0027, 0.0034, 0.0041]);
    expectRange(values, [-0.01, -0.009, -0.008], [0.003, 0.004, 0.005]);
  });

  test("formatValue: converts euro to cents with 1 decimal", () => {
    const options = generateTariffLimitOptions([0.25, 0.35], {
      formatValue: (value: number) => `${(value * 100).toFixed(1)} ct/kWh`,
    });

    expect(options[0]).toEqual({ value: -0.05, name: "≤ -5.0 ct/kWh" });
    expect(options[1]).toEqual({ value: -0.045, name: "≤ -4.5 ct/kWh" });
    expect(options[options.length - 1]).toEqual({ value: 0.35, name: "≤ 35.0 ct/kWh" });
  });

  test("formatValue: works with CO2 emissions formatter", () => {
    const options = generateTariffLimitOptions([200, 400], {
      formatValue: (value: number) => `${Math.round(value)} gCO₂/kWh`,
      includeNegatives: false,
    });

    expect(options[0]).toEqual({ value: 0, name: "≤ 0 gCO₂/kWh" });
    expect(options[1]).toEqual({ value: 5, name: "≤ 5 gCO₂/kWh" });
    expect(options[options.length - 1]).toEqual({ value: 405, name: "≤ 405 gCO₂/kWh" });
  });

  test("direction: uses correct operator in formatted names", () => {
    const options = generateTariffLimitOptions([0.1, 0.2], {
      formatValue: (value: number) => `${(value * 100).toFixed(1)} ct/kWh`,
      direction: "above",
    });

    expect(options[0]).toEqual({ value: -0.05, name: "≥ -5.0 ct/kWh" });
    expect(options[10]).toEqual({ value: 0, name: "≥ 0.0 ct/kWh" });
    expect(options[options.length - 1]).toEqual({ value: 0.205, name: "≥ 20.5 ct/kWh" });
  });
});
