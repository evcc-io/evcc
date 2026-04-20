import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

const from = "2026-03-24T21:00:00+01:00";
const to = "2026-03-25T01:00:00+01:00";

test.beforeAll(async () => {
  await start(undefined, "energy-history.sql");
});
test.afterAll(async () => {
  await stop();
});

test.describe("energy history API", () => {
  test("15-minute resolution", async ({ request }) => {
    const res = await request.get(`${baseUrl()}/api/history/energy`, {
      params: { from, to },
    });
    expect(res.ok()).toBeTruthy();

    const data = await res.json();
    expect(data).toHaveLength(2);

    const grid = data.find((s: any) => s.name === "grid");
    const home = data.find((s: any) => s.name === "home");
    expect(grid).toBeDefined();
    expect(home).toBeDefined();

    expect(grid.data).toHaveLength(6);
    expect(home.data).toHaveLength(6);

    expect(home.data[0].import).toBeCloseTo(0.1, 4);
    expect(home.data[0].export).toBeCloseTo(0, 4);
    expect(grid.data[0].import).toBeCloseTo(0.5, 4);
    expect(grid.data[0].export).toBeCloseTo(0.1, 4);
  });

  test("day aggregation", async ({ request }) => {
    const res = await request.get(`${baseUrl()}/api/history/energy`, {
      params: { from, to, aggregate: "day" },
    });
    expect(res.ok()).toBeTruthy();

    const data = await res.json();
    expect(data).toHaveLength(2);

    const grid = data.find((s: any) => s.name === "grid");
    const home = data.find((s: any) => s.name === "home");
    expect(grid).toBeDefined();
    expect(home).toBeDefined();

    expect(grid.data).toHaveLength(2);
    expect(home.data).toHaveLength(2);

    const [homeDay1, homeDay2] = home.data;
    const [gridDay1, gridDay2] = grid.data;

    // day 1 (2026-03-24): 4 slots
    expect(homeDay1.import).toBeCloseTo(0.4, 4);
    expect(homeDay1.export).toBeCloseTo(0, 4);
    expect(gridDay1.import).toBeCloseTo(2.0, 4);
    expect(gridDay1.export).toBeCloseTo(0.4, 4);

    // day 2 (2026-03-25): 2 slots
    expect(homeDay2.import).toBeCloseTo(0.2, 4);
    expect(homeDay2.export).toBeCloseTo(0, 4);
    expect(gridDay2.import).toBeCloseTo(1.0, 4);
    expect(gridDay2.export).toBeCloseTo(0.2, 4);
  });
});
