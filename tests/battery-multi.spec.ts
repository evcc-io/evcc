import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

const CONFIG = "battery-multi.evcc.yaml";
const SQL = "battery-multi.sql";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.beforeEach(async () => {
  await start(CONFIG, SQL);
});
test.afterEach(async () => {
  await stop();
});

test.describe("battery page with multiple batteries", async () => {
  test("status cards: combined aggregate plus one per battery", async ({ page }) => {
    await page.goto("/#/battery");

    const cards = page.getByTestId("battery-status-card");
    await expect(cards).toHaveCount(3); // combined + battery1 + battery2

    // combined card first, showing the capacity-weighted site soc (76%@13.5 + 40%@7.5 -> 63%)
    await expect(cards.first()).toContainText("Combined");
    await expect(cards.first()).toContainText("63%");

    // per-battery: soc, charge/discharge state and energy of total
    const charging = cards.filter({ hasText: "76%" });
    await expect(charging).toContainText("charging"); // battery1 power -800 W
    await expect(charging).toContainText("13.5 kWh"); // of total capacity

    const discharging = cards.filter({ hasText: "40%" });
    await expect(discharging).toContainText("discharging"); // battery2 power 1200 W
  });

  test("history chart: unit toggle persists and window pages", async ({ page }) => {
    await page.goto("/#/battery");

    const energy = page.getByTestId("batteryUnit-energy");
    await expect(energy).toBeEnabled(); // both batteries have capacity
    await energy.click();
    await expect(energy).toHaveAttribute("aria-checked", "true");

    // unit choice persists across a reload
    await page.reload();
    await expect(page.getByTestId("batteryUnit-energy")).toHaveAttribute("aria-checked", "true");

    // paging: cannot go into the future at offset 0, prev enables next
    const prev = page.getByTestId("battery-chart-prev");
    const next = page.getByTestId("battery-chart-next");
    await expect(next).toBeDisabled();
    await expect(prev).toBeEnabled();
    await prev.click();
    await expect(next).toBeEnabled();
  });

  test("usage configuration reflects and updates the stored thresholds", async ({ page }) => {
    await page.goto("/#/battery");

    // section headings render
    await expect(page.getByText("Where does the surplus go first?")).toBeVisible();
    await expect(page.getByText("Battery as charging buffer")).toBeVisible();

    // stored thresholds shown in the inline pickers (prioritySoc 50, bufferSoc 80)
    const prioritySoc = page.getByTestId("battery-priority").getByRole("combobox");
    const bufferSoc = page.getByTestId("battery-buffer").getByRole("combobox").first();
    await expect(prioritySoc).toHaveValue("50");
    await expect(bufferSoc).toHaveValue("80");

    // changing priority updates the picker value
    await prioritySoc.selectOption("30");
    await expect(prioritySoc).toHaveValue("30");

    // values crossing the other threshold are not selectable (prioritySoc 30, bufferSoc 80)
    await expect(prioritySoc.getByRole("option", { name: "85%", exact: true })).toHaveAttribute(
      "disabled",
      ""
    );
    await expect(bufferSoc.getByRole("option", { name: "25%", exact: true })).toHaveAttribute(
      "disabled",
      ""
    );

    // equal values unlock higher priorities; selecting one raises the buffer instead
    await prioritySoc.selectOption("80");
    await prioritySoc.selectOption("85");
    await expect(bufferSoc).toHaveValue("85");
    await expect(prioritySoc).toHaveValue("80");

    // discharge control is offered for the controllable battery and toggles on
    const discharge = page.getByRole("switch", { name: /Prevent home battery/ });
    await expect(discharge).not.toBeChecked();
    await discharge.click();
    await expect(discharge).toBeChecked();
  });

  // the limit is inert while the experimental grid discharge switch is off, so its
  // control only shows once the switch is on
  test("battery config: grid discharge limit follows the grid discharge switch", async ({
    page,
  }) => {
    await page.goto("/#/battery");
    const gridDischarge = page.getByRole("switch", { name: /discharge to the grid/ });
    const limit = page.getByTestId("battery-grid-discharge-limit");
    await expect(gridDischarge).not.toBeChecked();
    await expect(limit).not.toBeVisible();

    await gridDischarge.click();
    await expect(gridDischarge).toBeChecked();

    // the card carries a working limit control, not just its heading
    await expect(limit).toBeVisible();
    await expect(limit.getByRole("combobox")).toBeVisible();

    // 6 ct catches the base feed-in rate, so the whole forecast window reacts
    await limit.getByLabel("Enable limit").check();
    await limit.getByRole("combobox").selectOption("0.06");
    await expect(limit.getByTestId("active-hours")).toContainText("Active time");

    // price range follows the limit: full tariff span at 6 ct, only the
    // 28 ct zone once the limit excludes the base rate
    const range = limit.getByTestId("price-range");
    await expect(range).toContainText("6.0 ct – 28.0 ct");
    await limit.getByRole("combobox").selectOption("0.28");
    await expect(range).toContainText("28.0 ct – 28.0 ct");

    // switch and limit survive a reload
    await page.reload();
    await expect(gridDischarge).toBeChecked();
    await expect(limit.getByRole("combobox")).toHaveValue("0.28");
  });
});
