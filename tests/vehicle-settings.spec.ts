import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorUrl, simulatorConfig } from "./simulator";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await startSimulator();
});
test.afterAll(async () => {
  await stopSimulator();
});

test.beforeEach(async ({ page }) => {
  await start(simulatorConfig());

  await page.goto(simulatorUrl());
  await page.getByLabel("Grid Power").fill("500");
  await page.getByTestId("vehicle0").getByLabel("SoC").fill("20");
  await page.getByTestId("loadpoint0").getByText("B (connected)").click();
  await page.getByRole("button", { name: "Apply changes" }).click();
});

test.afterEach(async () => {
  await stop();
});

test.describe("minSoc", async () => {
  test("apply and restart", async ({ page }) => {
    await page.goto("/");

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();

    await expect(page.getByText("charged to x in solar mode")).toBeVisible();
    await page.getByRole("combobox", { name: "Min. charge %" }).selectOption("20%");
    await expect(page.getByText("charged to 20% in solar mode")).toBeVisible();

    await restart(simulatorConfig());
    await page.reload();

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByText("charged to 20% in solar mode")).toBeVisible();
  });

  test("show minsoc indicator when minsoc is active", async ({ page }) => {
    await page.goto("/");

    await expect(page.getByTestId("charging-plan")).toContainText("Plan");
    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await page.getByRole("combobox", { name: "Min. charge %" }).selectOption("50%");
    await page.getByRole("button", { name: "Close" }).click();

    await expect(page.getByTestId("vehicle-status-minsoc")).toBeVisible();
    await expect(page.getByTestId("vehicle-status-minsoc")).toHaveText("50%");

    await page.getByTestId("vehicle-status-minsoc").click();
    await page.getByRole("combobox", { name: "Min. charge %" }).selectOption("---");
    await page.getByRole("button", { name: "Close" }).click();

    await expect(page.getByTestId("vehicle-status-minsoc")).not.toBeVisible();
  });
});

test.describe("limitSoc", async () => {
  test("apply and restart", async ({ page }) => {
    await page.goto("/");

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();

    await page.getByRole("combobox", { name: "Default limit" }).selectOption("80%");
    await page.getByRole("button", { name: "Close" }).click();
    await expect(page.getByTestId("limit-soc-value")).toContainText("80%");

    await restart(simulatorConfig());
    await page.reload();

    await expect(page.getByTestId("limit-soc-value")).toContainText("80%");

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByRole("combobox", { name: "Default limit" })).toHaveValue("80");
  });
});

test.describe("minSoc and limitSoc", async () => {
  test("disabled for offline vehicles", async ({ page }) => {
    await page.goto("/");

    // switch to offline vehicle
    await page.getByTestId("change-vehicle").locator("select").selectOption("grüner Honda e");

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByRole("combobox", { name: "Min. charge %" })).toBeDisabled();
    await expect(page.getByRole("combobox", { name: "Default limit" })).toBeDisabled();
  });

  test("disabled for guest vehicles", async ({ page }) => {
    await page.goto("/");

    // switch to offline vehicle
    await page.getByTestId("change-vehicle").locator("select").selectOption("Guest vehicle");

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByRole("combobox", { name: "Min. charge %" })).toBeDisabled();
    await expect(page.getByRole("combobox", { name: "Default limit" })).toBeDisabled();
  });
});
