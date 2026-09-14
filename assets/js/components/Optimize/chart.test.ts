import { describe, expect, test } from "vite-plus/test";
import { dayBoundarySeries, isMidnight, slotXAxis } from "./chart";

// 15min slots starting at the given local hour
function times(startHour: number, count: number): number[] {
  const start = new Date(2025, 0, 1, startHour).getTime();
  return Array.from({ length: count }, (_, i) => start + i * 15 * 60 * 1000);
}

describe("slotXAxis", () => {
  const weekdayShort = () => "Thu";
  // 20:00 to 08:00, midnight at index 16
  const slots = times(20, 48);
  const axis = slotXAxis(slots, weekdayShort);
  const label = (i: number) => axis.axisLabel.formatter(String(slots[i]));

  test("labels full hours only", () => {
    expect(label(0)).toBe("20"); // 20:00
    expect(label(1)).toBe(""); // 20:15
    expect(label(4)).toBe(""); // 21:00, not a step hour
    expect(label(32)).toBe("4"); // 04:00
  });

  test("shows the weekday at midnight", () => {
    expect(label(16)).toBe("0\nThu");
  });

  test("marks day boundaries just inside the bar edge", () => {
    expect(isMidnight(slots[16])).toBe(true);
    expect(dayBoundarySeries(slots).markLine.data).toEqual([{ xAxis: 15.85 }]);
    // no line at the axis start, even if the first slot is midnight
    expect(dayBoundarySeries(times(0, 4)).markLine.data).toEqual([]);
  });
});
