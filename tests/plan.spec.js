import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

const CONFIG = "plan.evcc.yaml";

test.beforeEach(async () => {
  await start(CONFIG, "password.sql");
});

test.afterEach(async () => {
  await stop();
});

async function setAndVerifyPlan(page, lp, { soc, energy }) {
  await lp.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();

  if (soc) {
    await page.getByTestId("plan-soc").selectOption(soc);
  }
  if (energy) {
    // select "25 kWh (+50%)" by providing "25 kWh" as option text
    const optionText = await page
      .getByTestId("plan-energy")
      .locator("option", { hasText: energy })
      .textContent();
    await page.getByTestId("plan-energy").selectOption(optionText);
  }
  await page.getByTestId("plan-active").click();
  await page.getByRole("button", { name: "Close" }).click();
  await expect(lp.getByTestId("charging-plan")).toContainText(soc || energy);
}

test.describe("basic functionality", async () => {
  test("vehicle with soc and capacity, set and restart", async ({ page }) => {
    await page.goto("/");

    const lp1 = await page.getByTestId("loadpoint").first();

    // change vehicle
    await lp1.getByRole("button", { name: "Guest vehicle" }).click();
    await lp1.getByRole("button", { name: "Vehicle with SoC with Capacity" }).click();

    await expect(lp1.getByTestId("plan-marker")).not.toBeVisible();
    await expect(lp1.getByText("Loadpoint", { exact: true })).toBeVisible();

    await lp1.getByTestId("limit-soc").getByRole("combobox").selectOption("90%");
    await lp1.getByRole("button", { name: "Solar", exact: true }).click();
    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();

    await page.getByTestId("plan-day").selectOption({ index: 1 });
    await page.getByTestId("plan-time").fill("09:30");
    await page.getByTestId("plan-soc").selectOption("80%");
    await page.getByTestId("plan-active").click();
    await page.getByRole("button", { name: "Close" }).click();

    await expect(lp1.getByTestId("plan-marker")).toBeVisible();
    await expect(lp1.getByTestId("charging-plan").getByRole("button")).toHaveText(
      "tomorrow 9:30 AM80%"
    );

    await expect(lp1.getByTestId("vehicle-status")).toContainText("Charging plan starts at");
    await expect(lp1.getByTestId("plan-marker")).toBeVisible();
    await expect(lp1.getByTestId("charging-plan").getByRole("button")).toHaveText(
      "tomorrow 9:30 AM80%"
    );
    await lp1.getByTestId("charging-plan").getByRole("button").click();
    await expect(page.getByTestId("plan-soc")).toHaveValue("80");
  });
});

test.describe("vehicle variations", async () => {
  test.describe("guest vehicle", async () => {
    test("kWh based plan and limit", async ({ page }) => {
      await page.goto("/");

      const lp1 = await page.getByTestId("loadpoint").first();

      // change vehicle
      await expect(lp1.getByRole("button", { name: "Guest vehicle" })).toBeVisible();

      // kWh based limit
      await lp1.getByTestId("limit-energy").getByRole("combobox").selectOption("50 kWh");

      // kWh based plan
      await setAndVerifyPlan(page, lp1, { energy: "25 kWh" });
    });
  });

  test.describe("vehicle no soc no capacity", async () => {
    test("kWh based plan and limit", async ({ page }) => {
      await page.goto("/");

      const lp1 = await page.getByTestId("loadpoint").first();

      // change vehicle
      await lp1.getByRole("button", { name: "Guest vehicle" }).click();
      await lp1.getByRole("button", { name: "Vehicle no SoC no Capacity" }).click();

      // kWh based limit
      await lp1.getByTestId("limit-energy").getByRole("combobox").selectOption("50 kWh");

      // kWh based plan
      await setAndVerifyPlan(page, lp1, { energy: "25 kWh" });
    });
  });

  test.describe("vehicle no soc with capacity", async () => {
    test("kWh based plan and limit", async ({ page }) => {
      await page.goto("/");

      const lp1 = await page.getByTestId("loadpoint").first();

      // change vehicle
      await lp1.getByRole("button", { name: "Guest vehicle" }).click();
      await lp1.getByRole("button", { name: "Vehicle no SoC with Capacity" }).click();

      // kWh based limit
      await lp1.getByTestId("limit-energy").getByRole("combobox").selectOption("50 kWh (+50%)");

      // kWh based plan
      await setAndVerifyPlan(page, lp1, { energy: "25 kWh" });
    });
  });

  test.describe("vehicle with soc no capacity", async () => {
    test("kWh based plan and soc based limit", async ({ page }) => {
      await page.goto("/");

      const lp1 = await page.getByTestId("loadpoint").first();

      // change vehicle
      await lp1.getByRole("button", { name: "Guest vehicle" }).click();
      await lp1.getByRole("button", { name: "Vehicle with SoC no Capacity" }).click();

      // soc based limit
      await lp1.getByTestId("limit-soc").getByRole("combobox").selectOption("80%");

      // soc based plan
      await setAndVerifyPlan(page, lp1, { energy: "50 kWh" });
    });
  });

  test.describe("vehicle with soc with capacity", async () => {
    test("soc based plan and limit", async ({ page }) => {
      await page.goto("/");

      const lp1 = await page.getByTestId("loadpoint").first();

      // change vehicle
      await lp1.getByRole("button", { name: "Guest vehicle" }).click();
      await lp1.getByRole("button", { name: "Vehicle with SoC with Capacity" }).click();

      // soc based limit
      await lp1.getByTestId("limit-soc").getByRole("combobox").selectOption("80%");

      // soc based plan
      await setAndVerifyPlan(page, lp1, { soc: "60%" });
    });
  });

  test.describe("loadpoint with soc, guest vehicle", async () => {
    test("kWh based plan and soc based limit", async ({ page }) => {
      await page.goto("/");

      const lp2 = await page.getByTestId("loadpoint").last();

      // change vehicle
      await expect(lp2.getByRole("button", { name: "Guest vehicle" })).toBeVisible();

      // soc based limit
      await lp2.getByTestId("limit-soc").getByRole("combobox").selectOption("80%");

      // soc based plan
      await setAndVerifyPlan(page, lp2, { energy: "50 kWh" });
    });
  });

  test.describe("loadpoint with soc, vehicle with capacity", async () => {
    test("soc based plan and limit", async ({ page }) => {
      await page.goto("/");

      const lp2 = await page.getByTestId("loadpoint").last();

      // change vehicle
      await lp2.getByRole("button", { name: "Guest vehicle" }).click();
      await lp2.getByRole("button", { name: "Vehicle no SoC with Capacity" }).click();

      // soc based limit
      await lp2.getByTestId("limit-soc").getByRole("combobox").selectOption("80%");

      // soc based plan
      await setAndVerifyPlan(page, lp2, { soc: "60%" });
    });
  });
});

test.describe("preview", async () => {
  const cases = [
    {
      szenario: "kWh based plan",
      vehicle: "Vehicle no SoC with Capacity",
      goalId: "plan-energy",
    },
    {
      szenario: "soc based plan",
      vehicle: "Vehicle with SoC with Capacity",
      goalId: "plan-soc",
    },
  ];

  cases.forEach((c) => {
    test(c.szenario, async ({ page }) => {
      await page.goto("/");

      const lp1 = await page.getByTestId("loadpoint").first();

      // change vehicle
      await lp1.getByRole("button", { name: "Guest vehicle" }).click();
      await lp1.getByRole("button", { name: c.vehicle }).click();

      await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();

      // initial set -> preview plan
      await page.getByTestId("plan-day").selectOption({ index: 1 });
      await page.getByTestId("plan-time").fill("09:30");
      await page.getByTestId(c.goalId).selectOption("80");
      await expect(page.getByTestId("plan-preview-title")).toHaveText("Preview plan");

      // activate -> active plan
      await page.getByTestId("plan-active").click();
      await expect(page.getByTestId("plan-preview-title")).toHaveText("Active plan");

      // change -> preview plan
      await page.getByTestId(c.goalId).selectOption("90");
      await expect(page.getByTestId("plan-preview-title")).toHaveText("Preview plan");

      // apply -> active plan
      await expect(page.getByTestId("plan-apply")).toBeVisible();
      await page.getByTestId("plan-apply").click();
      await expect(page.getByTestId("plan-preview-title")).toHaveText("Active plan");

      // deactivate -> stay in preview
      await page.getByTestId("plan-time").fill("23:30");
      await expect(page.getByTestId("plan-preview-title")).toHaveText("Preview plan");
      await expect(page.getByTestId("plan-apply")).toBeVisible();
      await page.getByTestId("plan-active").click();
      await expect(page.getByTestId("plan-preview-title")).toHaveText("Preview plan");
    });
  });
});

test.describe("warnings", async () => {
  test("goal not reachable in time", async ({ page }) => {
    await page.goto("/");

    const lp1 = await page.getByTestId("loadpoint").first();

    // change vehicle
    await lp1.getByRole("button", { name: "Guest vehicle" }).click();
    await lp1.getByRole("button", { name: "Vehicle with SoC with massive Capacity" }).click();
    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();

    await page.getByTestId("plan-active").click();

    await expect(page.getByTestId("plan-warnings")).toContainText(
      "Goal not reachable in time. Estimated finish"
    );
  });
  test("time in the past", async ({ page }) => {
    await page.goto("/");

    const lp1 = await page.getByTestId("loadpoint").first();

    // change vehicle
    await lp1.getByRole("button", { name: "Guest vehicle" }).click();
    await lp1.getByRole("button", { name: "Vehicle with SoC with Capacity" }).click();
    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();

    await expect(page.getByTestId("plan-entry-warnings")).not.toBeVisible();

    await page.getByTestId("plan-day").selectOption({ index: 0 });
    await page.getByTestId("plan-time").fill("00:01");

    await expect(page.getByTestId("plan-entry-warnings")).toContainText(
      "Pick a time in the future, Marty."
    );
    await page.getByTestId("plan-time").fill("00:01");
  });
});
