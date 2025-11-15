/**
 * Dynamic chart scaling utilities for price visualizations.
 * These functions ensure that price differences are visible even when values are similar.
 */

export interface ScalingResult {
  min: number;
  max: number;
  range: number;
}

/**
 * Calculate dynamic y-axis boundaries for price data with visual margins.
 * This ensures bars remain visually distinguishable even when prices vary slightly.
 *
 * @param values - Array of numeric values to scale
 * @param marginPercent - Percentage of range to add as margin (default: 10%)
 * @param minMargin - Minimum absolute margin to ensure visibility (default: 0.01)
 * @returns Object with min, max, and range values
 */
export const calculateDynamicScale = (
  values: (number | undefined)[],
  marginPercent: number = 10,
  minMargin: number = 0.01
): ScalingResult => {
  // Filter out undefined values
  const validValues = values.filter((v): v is number => v !== undefined && !isNaN(v));

  if (validValues.length === 0) {
    return { min: 0, max: 1, range: 1 };
  }

  const dataMin = Math.min(...validValues);
  const dataMax = Math.max(...validValues);
  const dataRange = dataMax - dataMin;

  // Calculate margin as percentage of range, but at least minMargin
  const margin = Math.max(dataRange * (marginPercent / 100), minMargin);

  // Apply margins
  const min = dataMin - margin;
  const max = dataMax + margin;
  const range = max - min;

  return { min, max, range };
};

/**
 * Normalize a value to a percentage (0-100) based on dynamic scale.
 * Used for calculating bar heights relative to min/max boundaries.
 *
 * @param value - The value to normalize
 * @param min - Minimum boundary
 * @param range - Range (max - min)
 * @param minHeight - Minimum height percentage (default: 10%)
 * @param maxHeight - Maximum height percentage (default: 90%)
 * @returns Percentage height (minHeight to maxHeight)
 */
export const normalizeToHeight = (
  value: number | undefined,
  min: number,
  range: number,
  minHeight: number = 10,
  maxHeight: number = 90
): number => {
  if (value === undefined || isNaN(value)) {
    return (minHeight + maxHeight) / 2; // Return middle height for undefined values
  }

  if (range === 0) {
    return (minHeight + maxHeight) / 2; // Return middle height if all values are the same
  }

  // Calculate normalized position (0 to 1)
  const normalized = (value - min) / range;

  // Map to height range (minHeight% to maxHeight%)
  return minHeight + normalized * (maxHeight - minHeight);
};
