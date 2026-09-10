import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.beforeEach(async () => {
  await start("battery-settings.evcc.yaml");
});
test.afterEach(async () => {
  await stop();
});

test.describe("battery settings", async () => {
  test("battery view", async ({ page }) => {
    await page.goto("/#/battery");

    await expect(page.getByTestId("header")).toContainText("Home Battery");
    await expect(page.getByRole("heading", { name: "Battery usage" })).toContainText(
      "Battery level 50%"
    );
    await expect(page.getByRole("heading", { name: "Grid charging" })).toBeVisible();
    await expect(page.getByTestId("battery-status-card")).toContainText("10.0 kWh");
  });

  test("battery usage", async ({ page }) => {
    await page.goto("/#/battery");

    const prioritySoc = page.getByTestId("battery-priority").getByRole("combobox");
    const buffer = page.getByTestId("battery-buffer");
    const bufferSoc = buffer.getByRole("combobox").first();

    await prioritySoc.selectOption("50");
    await expect(prioritySoc).toHaveValue("50");

    await bufferSoc.selectOption("80");
    await expect(bufferSoc).toHaveValue("80");

    const bufferStart = buffer.getByRole("combobox").last();
    await bufferStart.selectOption("90");
    await expect(bufferStart).toHaveValue("90");

    // persisted
    await page.reload();
    await expect(prioritySoc).toHaveValue("50");
    await expect(bufferSoc).toHaveValue("80");
    await expect(bufferStart).toHaveValue("90");
  });

  test("buffer 100% disables battery-supported charging", async ({ page }) => {
    await page.goto("/#/battery");

    const buffer = page.getByTestId("battery-buffer");
    const bufferSoc = buffer.getByRole("combobox").first();

    await expect(bufferSoc).toHaveValue("100");
    await expect(buffer).toContainText("not used for the charging points");
    await expect(buffer.getByRole("combobox")).toHaveCount(1);

    await bufferSoc.selectOption("80");
    await expect(buffer).toContainText("Start solar charging automatically");
    await expect(buffer.getByRole("combobox")).toHaveCount(2);

    await bufferSoc.selectOption("100");
    await expect(buffer).toContainText("not used for the charging points");
  });

  test("grid charging", async ({ page }) => {
    await page.goto("/#/battery");

    await page.getByLabel("Enable limit").check();
    await page.getByLabel("Price limit").selectOption({ label: "≤ 50.0 ct/kWh" });
    await expect(page.getByTestId("active-hours")).toHaveText(["Active time", "96 hr"].join(""));
    await expect(page.locator("body")).toContainText("5.0 ct – 50.0 ct");

    await page.getByRole("link", { name: "Charge" }).click();
    await page.getByTestId("energyflow").click();
    await page.getByRole("button", { name: "Grid charging: active (≤ 50.0 ct)" }).click();
    await expect(page).toHaveURL(/#\/battery/);

    await page.getByLabel("Price limit").selectOption({ label: "≤ -10.0 ct/kWh" });
    await expect(page.getByTestId("active-hours")).toHaveText("Active time");

    await page.getByRole("link", { name: "Charge" }).click();
    await expect(
      page.getByRole("button", { name: "Grid charging: when ≤ -10.0 ct" })
    ).toBeVisible();
  });

  test("hold mode display", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("energyflow").click();

    const discharge = page.getByTestId("energyflow-entry-batterydischarge");
    const charge = page.getByTestId("energyflow-entry-batterycharge");

    await expect(discharge).toContainText("Battery discharging");
    await expect(charge).toContainText("Battery charging");

    // enable discharge lock
    await page.goto("/#/battery");
    await page
      .getByLabel("Prevent home battery discharge in fast mode and during planned charging.")
      .check();
    await page.waitForLoadState("networkidle");
    await page.getByRole("link", { name: "Charge" }).click();

    await page.getByTestId("energyflow").click();
    await expect(discharge).toContainText("Battery (discharge locked)");
    await expect(charge).toContainText("Battery charging");
  });
});
