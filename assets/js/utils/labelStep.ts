// Even category label step for a chart width, so labels thin out uniformly
// (1, 3, 5, …) instead of echarts dropping overlapping ones individually.
export const DAY_STEPS = [1, 2, 3, 4, 6, 12];
export const MONTH_STEPS = [1, 2, 5, 10];

export function labelStep(count: number, width: number, steps: number[], labelWidth = 40): number {
  const fit = Math.max(1, Math.floor((width || 1000) / labelWidth));
  return steps.find((step) => count / step <= fit) ?? (steps.at(-1) as number);
}
