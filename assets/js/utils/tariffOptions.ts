import { type SelectOption, type LimitDirection } from "@/types/evcc";

export interface TariffLimitOptionsConfig {
  /** currently selected value */
  selectedValue?: number | null;
  /** include negative values in range */
  includeNegatives?: boolean;
  /** include extra high values (+50% above max) */
  extraHigh?: boolean;
  /** include extra low values (+50% below min) */
  extraLow?: boolean;
  /** format function for display values */
  formatValue: (value: number) => string;
  /** direction to determine operator */
  direction?: LimitDirection;
}

export const generateTariffLimitOptions = (
  values: number[],
  config: TariffLimitOptionsConfig
): SelectOption<number>[] => {
  const {
    selectedValue,
    includeNegatives = true,
    extraHigh = false,
    extraLow = false,
    formatValue,
    direction = "below",
  } = config;

  if (!values.length) {
    const operator = direction === "below" ? "≤" : "≥";
    return [{ value: 0, name: `${operator} ${formatValue(0)}` }];
  }

  // calculate data range
  const validValues = values.filter((v) => v !== undefined && !isNaN(v));
  const min = Math.min(...validValues);
  const max = Math.max(...validValues);

  // calculate step size
  const stepSize = calculateStepSize(min, max);

  // calculate start value with extra low range
  const baseStart = includeNegatives ? stepSize * -11 : 0;
  const extraLowBase = Math.min(min, 0);
  const extraLowValue = extraLow ? extraLowBase - (max - extraLowBase) * 0.5 : min;
  const startValue = Math.min(baseStart, extraLowValue);
  const adjustedStartValue = Math.floor(startValue / stepSize) * stepSize;

  // calculate end value with extra high range
  const extraHighValue = extraHigh ? max + (max - min) * 0.5 : max;
  const endValue = extraHighValue;

  // generate option values
  const optionValues = [] as number[];
  for (let i = 1; i <= 100; i++) {
    const value = adjustedStartValue + stepSize * i;
    if (value > endValue + stepSize) break;
    optionValues.push(roundLimit(value));
  }

  // add selected value if not in scale
  if (
    selectedValue !== null &&
    selectedValue !== undefined &&
    !optionValues.includes(selectedValue)
  ) {
    optionValues.push(selectedValue);
  }

  // always ensure 0 is included
  if (!optionValues.includes(0)) {
    optionValues.push(0);
  }

  optionValues.sort((a, b) => a - b);

  const operator = direction === "below" ? "≤" : "≥";

  return optionValues.map((value) => ({
    value,
    name: `${operator} ${formatValue(value)}`,
  }));
};

const calculateStepSize = (min: number, max: number): number => {
  const baseSteps = [0.001, 0.002, 0.005];
  const range = max - Math.min(0, min);

  for (let scale = 1; scale <= 10000; scale *= 10) {
    for (const baseStep of baseSteps) {
      const step = baseStep * scale;
      if (range < step * 100) return step;
    }
  }
  return 1;
};

const roundLimit = (limit: number): number => {
  return Math.round(limit * 1000) / 1000;
};
