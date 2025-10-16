import { test, expect, devices, type Page, type Locator } from "@playwright/test";
import { start, stop, baseUrl, restart } from "./evcc";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

const mobile = devices["iPhone 12 Mini"].viewport;

const CONFIG = "plan.evcc.yaml";
const CONFIG_NO_TARIFF = "basics.evcc.yaml";
const CONFIG_FIXED_TARIFF = "plan-fixed-tariff.evcc.yaml";

test.beforeEach(async () => {
  await start(CONFIG);
});

test.afterEach(async () => {
  await stop();
});

function getWeekday(offset = 1) {
  const date = new Date();
  date.setDate(date.getDate() + offset);
  return date.toLocaleDateString("en-US", { weekday: "long" });
}

async function setAndVerifyPlan(
  page: Page,
  lp: Locator,
  { soc, energy }: { soc?: string; energy?: string }
) {
  await lp.getByTestId("charging-plan-button").click();

  if (soc) {
    await page.getByTestId("static-plan-soc").selectOption(soc);
  }
  if (energy) {
    // select "25 kWh (+50%)" by providing "25 kWh" as option text
    const optionText = await page
      .getByTestId("static-plan-energy")
      .locator("option", { hasText: energy })
      .textContent();
    await page.getByTestId("static-plan-energy").selectOption(optionText);
  }
  await page.getByTestId("static-plan-active").click();
  await page.getByRole("button", { name: "Close" }).click();
  await expect(lp.getByTestId("charging-plan")).toContainText(soc || energy!);
}

async function verifyRepeatingPlanAvailable(page: Page, lp: Locator, expected: boolean) {
  await lp.getByTestId("charging-plan-button").click();
  if (expected) {
    await expect(page.getByTestId("repeating-plan-add")).toBeVisible();
  } else {
    await expect(page.getByTestId("repeating-plan-add")).not.toBeVisible();
  }
  await page.getByRole("button", { name: "Close" }).click();
}

test.describe("basic functionality", async () => {
  test("vehicle with soc and capacity, set and restart", async ({ page }) => {
    await page.goto("/");

    const lp1 = await page.getByTestId("loadpoint").first();

    // change vehicle
    await lp1
      .getByTestId("change-vehicle")
      .locator("select")
      .selectOption("Vehicle with SoC with Capacity");

    await expect(lp1.getByTestId("plan-marker")).not.toBeVisible();
    await expect(lp1.getByText("Loadpoint", { exact: true })).toBeVisible();

    await lp1.getByTestId("limit-soc").getByRole("combobox").selectOption("90%");
    await lp1.getByRole("button", { name: "Solar", exact: true }).click();
    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();

    await page.getByTestId("static-plan-day").selectOption({ index: 1 });
    await page.getByTestId("static-plan-time").fill("09:30");
    await page.getByTestId("static-plan-soc").selectOption("80%");
    await page.getByTestId("static-plan-precondition-lg-toggle").click();
    await page
      .getByTestId("static-plan-precondition-lg-select")
      .getByRole("combobox")
      .selectOption("1 hour");
    await page.getByTestId("static-plan-active").click();
    await page.getByRole("button", { name: "Close" }).click();

    await expect(lp1.getByTestId("plan-marker")).toBeVisible();
    await expect(lp1.getByTestId("charging-plan").getByRole("button")).toHaveText(
      "tomorrow 09:3080%"
    );

    await expect(lp1.getByTestId("vehicle-status-charger")).toHaveText("Connected.");
    await expect(lp1.getByTestId("vehicle-status-planstart")).toHaveText(/tomorrow .*/);
    await expect(lp1.getByTestId("plan-marker")).toBeVisible();
    await expect(lp1.getByTestId("charging-plan").getByRole("button")).toHaveText(
      "tomorrow 09:3080%"
    );
    await lp1.getByTestId("charging-plan").getByRole("button").click();
    await expect(page.getByTestId("static-plan-soc")).toHaveValue("80");
  });
});

test.describe("vehicle variations", async () => {
  test.describe("guest vehicle", async () => {
    test("kWh based plan and limit, no repeats", async ({ page }) => {
      await page.goto("/");

      const lp1 = await page.getByTestId("loadpoint").first();

      await expect(lp1.getByTestId("vehicle-name")).toHaveText("Guest vehicle");

      // kWh based limit
      await lp1.getByTestId("limit-energy").getByRole("combobox").selectOption("50 kWh");

      // kWh based plan
      await setAndVerifyPlan(page, lp1, { energy: "25 kWh" });

      // no repeating plans option
      await verifyRepeatingPlanAvailable(page, lp1, false);
    });

    test("non-standard energy values are visible in dropdown", async ({ page }) => {
      await page.goto("/");
      const lp1 = await page.getByTestId("loadpoint").first();

      // verify guest vehicle
      await expect(lp1.getByTestId("vehicle-name")).toHaveText("Guest vehicle");

      // create plan with non-standard energy value via API
      const response = await page.request.post(
        `/api/loadpoints/1/plan/energy/32/2050-09-04T05:00:00.000Z`
      );
      expect(response.status()).toBe(200);

      // verify plan is shown
      const plan = await lp1.getByTestId("charging-plan");
      await expect(plan).toContainText("32 kWh");

      // open modal and verify dropdown
      await plan.getByRole("button").click();
      const modal = await page.getByTestId("charging-plan-modal").first();
      const energySelect = modal.getByTestId("static-plan-energy");
      await expect(energySelect).toBeVisible();

      // verify non-standard value is selected
      await expect(energySelect).toHaveValue("32");
      await energySelect.selectOption("70");
      await expect(energySelect).toHaveValue("70");
    });
  });

  test.describe("vehicle no soc no capacity", async () => {
    test("kWh based plan and limit, no repeats", async ({ page }) => {
      await page.goto("/");

      const lp1 = await page.getByTestId("loadpoint").first();

      // change vehicle
      await lp1
        .getByTestId("change-vehicle")
        .locator("select")
        .selectOption("Vehicle no SoC no Capacity");

      // kWh based limit
      await lp1.getByTestId("limit-energy").getByRole("combobox").selectOption("50 kWh");

      // kWh based plan
      await setAndVerifyPlan(page, lp1, { energy: "25 kWh" });

      // no repeating plans option
      await verifyRepeatingPlanAvailable(page, lp1, false);
    });
  });

  test.describe("vehicle no soc with capacity", async () => {
    test("kWh based plan and limit, no repeats", async ({ page }) => {
      await page.goto("/");

      const lp1 = await page.getByTestId("loadpoint").first();

      // change vehicle
      await lp1
        .getByTestId("change-vehicle")
        .locator("select")
        .selectOption("Vehicle no SoC with Capacity");

      // kWh based limit
      await lp1.getByTestId("limit-energy").getByRole("combobox").selectOption("50 kWh (+50%)");

      // kWh based plan
      await setAndVerifyPlan(page, lp1, { energy: "25 kWh" });

      // no repeating plans option
      await verifyRepeatingPlanAvailable(page, lp1, false);
    });
  });

  test.describe("vehicle with soc no capacity", async () => {
    test("kWh based plan and soc based limit, no repeats", async ({ page }) => {
      await page.goto("/");

      const lp1 = await page.getByTestId("loadpoint").first();

      // change vehicle
      await lp1
        .getByTestId("change-vehicle")
        .locator("select")
        .selectOption("Vehicle with SoC no Capacity");

      // soc based limit
      await lp1.getByTestId("limit-soc").getByRole("combobox").selectOption("80%");

      // soc based plan
      await setAndVerifyPlan(page, lp1, { energy: "50 kWh" });

      // no repeating plans option
      await verifyRepeatingPlanAvailable(page, lp1, false);
    });
  });

  test.describe("vehicle with soc with capacity", async () => {
    test("soc based plan and limit, with repeats", async ({ page }) => {
      await page.goto("/");

      const lp1 = await page.getByTestId("loadpoint").first();

      // change vehicle
      await lp1
        .getByTestId("change-vehicle")
        .locator("select")
        .selectOption("Vehicle with SoC with Capacity");

      // soc based limit
      await lp1.getByTestId("limit-soc").getByRole("combobox").selectOption("80%");

      // soc based plan
      await setAndVerifyPlan(page, lp1, { soc: "60%" });

      // repeating plans option
      await verifyRepeatingPlanAvailable(page, lp1, true);
    });

    test("non-standard soc values are visible in dropdown", async ({ page }) => {
      await page.goto("/");
      const lp1 = await page.getByTestId("loadpoint").first();
      await lp1
        .getByTestId("change-vehicle")
        .locator("select")
        .selectOption("Vehicle with SoC with Capacity");

      // create plan with non-standard SoC value via API
      const response = await page.request.post(
        `/api/vehicles/vehicleSocCapacity/plan/soc/72/2050-09-04T05:00:00.000Z`
      );
      expect(response.status()).toBe(200);

      // verify plan is shown
      const plan = await lp1.getByTestId("charging-plan");
      await expect(plan).toContainText("72%");

      // open modal and verify dropdown
      plan.getByRole("button").click();
      const modal = await page.getByTestId("charging-plan-modal").first();
      const socSelect = modal.getByTestId("static-plan-soc");

      // verify non-standard value is selected
      await expect(socSelect).toHaveValue("72");
      await socSelect.selectOption("80%");
      await expect(socSelect).toHaveValue("80");
    });
  });

  test.describe("loadpoint with soc, guest vehicle", async () => {
    test("kWh based plan and soc based limit, no repeats", async ({ page }) => {
      await page.goto("/");

      const lp2 = await page.getByTestId("loadpoint").last();

      // change vehicle
      await expect(lp2.getByTestId("vehicle-name")).toHaveText("Guest vehicle");

      // soc based limit
      await lp2.getByTestId("limit-soc").getByRole("combobox").selectOption("80%");

      // soc based plan
      await setAndVerifyPlan(page, lp2, { energy: "50 kWh" });

      // no repeating plans option
      await verifyRepeatingPlanAvailable(page, lp2, false);
    });
  });

  test.describe("loadpoint with soc, vehicle with capacity", async () => {
    test("soc based plan and limit, with repeats", async ({ page }) => {
      await page.goto("/");

      const lp2 = await page.getByTestId("loadpoint").last();

      // change vehicle
      await lp2
        .getByTestId("change-vehicle")
        .locator("select")
        .selectOption("Vehicle no SoC with Capacity");

      // soc based limit
      await lp2.getByTestId("limit-soc").getByRole("combobox").selectOption("80%");

      // soc based plan
      await setAndVerifyPlan(page, lp2, { soc: "60%" });

      // repeating plans option
      await verifyRepeatingPlanAvailable(page, lp2, true);
    });
  });
});

test.describe("preview", async () => {
  const cases = [
    {
      szenario: "kWh based plan",
      vehicle: "Vehicle no SoC with Capacity",
      goalId: "static-plan-energy",
    },
    {
      szenario: "soc based plan",
      vehicle: "Vehicle with SoC with Capacity",
      goalId: "static-plan-soc",
    },
  ];

  cases.forEach((c) => {
    test(c.szenario, async ({ page }) => {
      await page.goto("/");

      const lp1 = await page.getByTestId("loadpoint").first();

      // change vehicle
      await lp1.getByTestId("change-vehicle").locator("select").selectOption(c.vehicle);

      await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();

      // initial set -> preview plan
      await page.getByTestId("static-plan-day").selectOption({ index: 1 });
      await page.getByTestId("static-plan-time").fill("09:30");
      await page.getByTestId(c.goalId).selectOption("80");
      await expect(page.getByTestId("plan-preview-title")).toHaveText("Preview plan");

      // activate -> active plan
      await page.getByTestId("static-plan-active").click();
      await expect(page.getByTestId("plan-preview-title")).toHaveText("Active plan");

      await page.getByTestId(c.goalId).selectOption("90");
      await expect(page.getByTestId("plan-preview-title")).toHaveText("Active plan");

      // apply -> active plan
      await expect(page.getByTestId("static-plan-apply")).toBeVisible();
      await page.getByTestId("static-plan-apply").click();
      await expect(page.getByTestId("static-plan-apply")).not.toBeVisible();
      await expect(page.getByTestId("plan-preview-title")).toHaveText("Active plan");

      await page.getByTestId("static-plan-time").fill("23:30");
      await expect(page.getByTestId("plan-preview-title")).toHaveText("Active plan");
      await expect(page.getByTestId("static-plan-apply")).toBeVisible();

      // deactivate
      await page.getByTestId("static-plan-active").click();
      await expect(page.getByTestId("plan-preview-title")).toHaveText("Preview plan");
    });
  });
  test("fixed tariff: show prices", async ({ page }) => {
    await restart(CONFIG_FIXED_TARIFF);
    await page.goto("/");

    const lp1 = await page.getByTestId("loadpoint").first();
    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();

    const modal = await page.getByTestId("charging-plan-modal");
    await expect(modal.getByTestId("tariff-value")).toHaveText(
      ["Energy price", "40.0 ct/kWh"].join("")
    );
  });
});

test.describe("warnings", async () => {
  test("goal not reachable in time", async ({ page }) => {
    await page.goto("/");

    const lp1 = await page.getByTestId("loadpoint").first();

    // change vehicle
    await lp1
      .getByTestId("change-vehicle")
      .locator("select")
      .selectOption("Vehicle with SoC with Massive Capacity");

    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();

    await page.getByTestId("static-plan-active").click();

    // match this text but with fuzzy date "getByText('Goal will be reached 52:10 h')"
    await expect(page.getByTestId("plan-warnings")).toHaveText(/Goal will be reached .* later/);
  });
  test("time in the past", async ({ page }) => {
    await page.goto("/");

    const lp1 = await page.getByTestId("loadpoint").first();

    // change vehicle
    await lp1
      .getByTestId("change-vehicle")
      .locator("select")
      .selectOption("Vehicle with SoC with Capacity");

    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();

    await expect(page.getByTestId("plan-entry-warnings")).not.toBeVisible();

    await page.getByTestId("static-plan-day").selectOption({ index: 0 });
    await page.getByTestId("static-plan-time").fill("00:01");

    await expect(page.getByTestId("plan-entry-warnings")).toContainText(
      "Pick a time in the future, Marty."
    );
    await page.getByTestId("static-plan-time").fill("00:01");
  });
});

test.describe("repeating", async () => {
  test("add and remove plans", async ({ page }) => {
    await page.goto("/");

    const lp1 = await page.getByTestId("loadpoint").first();
    await lp1
      .getByTestId("change-vehicle")
      .locator("select")
      .selectOption("Vehicle with SoC with Capacity");

    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    const modal = await page.getByTestId("charging-plan-modal");

    // one static plan, no number
    await expect(modal.getByTestId("plan-entry")).toHaveCount(1);
    await expect(modal.getByTestId("plan-entry").first()).not.toContainText("#1");

    // add plan
    await modal.getByRole("button", { name: "Add repeating plan" }).click();
    await expect(modal.getByTestId("plan-entry")).toHaveCount(2);
    await expect(modal.getByTestId("plan-entry").first()).toContainText("#1");
    await expect(modal.getByTestId("plan-entry").last()).toContainText("#2");

    // remove plan
    await modal.getByTestId("plan-entry").last().getByRole("button", { name: "Remove" }).click();
    await expect(modal.getByTestId("plan-entry")).toHaveCount(1);
    await expect(modal.getByTestId("plan-entry").first()).not.toContainText("#1");
  });

  test("preview static and repeating plan", async ({ page }) => {
    await page.goto("/");

    const lp1 = await page.getByTestId("loadpoint").first();
    await lp1
      .getByTestId("change-vehicle")
      .locator("select")
      .selectOption("Vehicle with SoC with Capacity");

    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    const modal = await page.getByTestId("charging-plan-modal");

    // one static plan
    await expect(modal.getByTestId("plan-entry")).toHaveCount(1);

    await modal.getByTestId("static-plan-day").selectOption({ index: 1 }); // tomorrow
    await modal.getByTestId("static-plan-time").fill("09:00");

    await expect(modal.getByTestId("plan-preview-title")).toHaveText("Preview plan");
    await expect(modal.getByTestId("target-text")).toContainText("09:00");

    // add repeating plan
    await modal.getByRole("button", { name: "Add repeating plan" }).click();
    await modal.getByTestId("repeating-plan-weekdays").click();
    await modal.getByRole("checkbox", { name: "Select all" }).check();
    await modal.getByTestId("repeating-plan-time").fill("11:11");

    // switch between previews
    await page.waitForLoadState("networkidle");
    await modal
      .getByTestId("plan-preview-title")
      .getByRole("combobox")
      .selectOption("Preview plan #2");
    await expect(modal.getByTestId("target-text")).toContainText("11:11");

    await modal
      .getByTestId("plan-preview-title")
      .getByRole("combobox")
      .selectOption("Preview plan #1");
    await expect(modal.getByTestId("target-text")).toContainText("09:00");

    // activate #1
    await modal.getByTestId("static-plan-active").click();
    await expect(modal.getByTestId("plan-preview-title")).toHaveText("Next plan #1");
    await expect(modal.getByTestId("target-text")).toContainText("09:00");

    // activate #2
    await modal.getByTestId("static-plan-active").click();
    await modal.getByTestId("repeating-plan-active").click();
    await expect(modal.getByTestId("plan-preview-title")).toHaveText("Next plan #2");
    await expect(modal.getByTestId("target-text")).toContainText("11:11");

    // back to preview if no active plan
    await modal.getByTestId("repeating-plan-active").click();
    await expect(modal.getByTestId("plan-preview-title").locator("option:checked")).toHaveText(
      "Preview plan #1"
    );
    await expect(modal.getByTestId("target-text")).toContainText("9:00");
  });

  test("weekday selection", async ({ page }) => {
    await page.goto("/");

    const lp1 = await page.getByTestId("loadpoint").first();
    await lp1
      .getByTestId("change-vehicle")
      .locator("select")
      .selectOption("Vehicle with SoC with Capacity");

    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    const modal = await page.getByTestId("charging-plan-modal");

    await modal.getByRole("button", { name: "Add repeating plan" }).click();

    // weekday select should have value "Mo-Fr"
    await expect(modal.getByTestId("repeating-plan-weekdays").getByRole("button")).toHaveText(
      "Mon – Fri"
    );

    // select all weekdays
    await modal.getByTestId("repeating-plan-weekdays").click();
    await modal.getByRole("checkbox", { name: "Select all" }).click();
    await expect(modal.getByTestId("repeating-plan-weekdays").getByRole("button")).toHaveText(
      "Mon – Sun"
    );

    // select none
    await modal.getByRole("checkbox", { name: "Select all" }).click();
    await expect(modal.getByTestId("repeating-plan-weekdays").getByRole("button")).toHaveText("–");

    // select specific weekdays
    await modal.getByRole("checkbox", { name: "Thursday" }).check();
    await expect(modal.getByTestId("repeating-plan-weekdays").getByRole("button")).toHaveText(
      "Thu"
    );
    await modal.getByTestId("repeating-plan-weekdays").click(); // close

    // activate
    await modal.getByTestId("repeating-plan-time").fill("02:22");
    await modal.getByTestId("repeating-plan-active").click();
    await page.waitForLoadState("networkidle");

    // specific weekday and time
    await expect(modal.getByTestId("plan-preview-title")).toHaveText("Next plan #2");
    await expect(modal.getByTestId("target-text")).toContainText("Thu 02:22");
  });

  test("next plan", async ({ page }) => {
    await page.goto("/");

    const yesterday = getWeekday(-1);
    const tomorrow = getWeekday(1);

    const lp1 = await page.getByTestId("loadpoint").first();
    await lp1
      .getByTestId("change-vehicle")
      .locator("select")
      .selectOption("Vehicle with SoC with Capacity");

    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    const modal = await page.getByTestId("charging-plan-modal");

    // configure static plan for tomorrow
    const plan1 = modal.getByTestId("plan-entry").nth(0);
    await plan1.getByTestId("static-plan-day").selectOption({ index: 1 });
    await plan1.getByTestId("static-plan-time").fill("09:30");
    await plan1.getByTestId("static-plan-soc").selectOption("80%");
    await plan1.getByTestId("static-plan-active").click();

    // add repeating plan for tomorrow
    await modal.getByRole("button", { name: "Add repeating plan" }).click();
    const plan2 = modal.getByTestId("plan-entry").nth(1);
    const days2 = plan2.getByTestId("repeating-plan-weekdays");
    await days2.click();
    await days2.getByRole("checkbox", { name: "Select all" }).click();
    await days2.getByRole("checkbox", { name: "Select all" }).click();
    await days2.getByRole("checkbox", { name: tomorrow }).check();
    await days2.click(); // close
    await plan2.getByTestId("repeating-plan-time").fill("09:20");
    await plan2.getByTestId("repeating-plan-active").click();

    // add repeating plan for every day
    await modal.getByRole("button", { name: "Add repeating plan" }).click();
    const plan3 = modal.getByTestId("plan-entry").nth(2);
    const days3 = plan3.getByTestId("repeating-plan-weekdays");
    await days3.click();
    await days3.getByRole("checkbox", { name: "Select all" }).check();
    await days3.click(); // close
    await plan3.getByTestId("repeating-plan-time").fill("09:10");
    await plan3.getByTestId("repeating-plan-active").click();

    // check next plans
    await expect(modal.getByTestId("plan-preview-title")).toHaveText("Next plan #3");
    await expect(modal.getByTestId("target-text")).toContainText("9:10");

    // disable plan #3
    await plan3.getByTestId("repeating-plan-active").click();
    await expect(modal.getByTestId("plan-preview-title")).toHaveText("Next plan #2");
    await expect(modal.getByTestId("target-text")).toContainText("9:20");

    // change plan #2 to yesterday
    await days2.click();
    await days2.getByRole("checkbox", { name: tomorrow }).click(); // uncheck
    await days2.getByRole("checkbox", { name: yesterday }).click(); // check
    await days2.click(); // close
    // no changes yet
    await expect(modal.getByTestId("plan-preview-title")).toHaveText("Next plan #2");
    await expect(modal.getByTestId("target-text")).toContainText("9:20");

    // apply
    await plan2.getByTestId("repeating-plan-apply").click();
    await expect(plan2.getByTestId("repeating-plan-apply")).not.toBeVisible();
    await expect(modal.getByTestId("plan-preview-title")).toHaveText("Next plan #1");
    await expect(modal.getByTestId("target-text")).toContainText("9:30");

    // set lower targets than vehicle soc (50%)
    await plan1.getByTestId("static-plan-soc").selectOption("40%");
    await plan1.getByTestId("static-plan-apply").click();
    await expect(plan1.getByTestId("static-plan-apply")).not.toBeVisible();
    await plan2.getByTestId("repeating-plan-soc").selectOption("40%");
    await plan2.getByTestId("repeating-plan-apply").click();
    await expect(plan2.getByTestId("repeating-plan-apply")).not.toBeVisible();
    await expect(modal.getByTestId("plan-preview-title")).toHaveText("Goal already reached");
  });

  test("repeating plan persistence", async ({ page }) => {
    await page.goto("/");

    const tomorrow = getWeekday(1);

    let lp1 = await page.getByTestId("loadpoint").first();
    await lp1
      .getByTestId("change-vehicle")
      .locator("select")
      .selectOption("Vehicle with SoC with Capacity");

    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    const modal = await page.getByTestId("charging-plan-modal").first();

    await modal.getByRole("button", { name: "Add repeating plan" }).click();
    const plan = modal.getByTestId("plan-entry").nth(1);
    await plan.getByTestId("repeating-plan-weekdays").click();
    await plan.getByRole("checkbox", { name: "Select all" }).click(); // check all
    await plan.getByRole("checkbox", { name: "Select all" }).click(); // uncheck all
    await plan.getByRole("checkbox", { name: tomorrow }).check();
    await plan.getByTestId("repeating-plan-time").fill("09:20");
    await plan.getByTestId("repeating-plan-precondition-lg-toggle").click();
    await plan
      .getByTestId("repeating-plan-precondition-lg-select")
      .getByRole("combobox")
      .selectOption("2 hours");
    await plan.getByTestId("repeating-plan-active").click();
    await expect(modal.getByTestId("plan-preview-title")).toHaveText("Next plan #2");
    await expect(modal.getByTestId("target-text")).toContainText("09:20");
    await modal.getByRole("button", { name: "Close" }).click();
    await expect(
      lp1.getByTestId("charging-plan").getByRole("button", { name: "tomorrow 09:20" })
    ).toBeVisible();

    await restart(CONFIG);
    await page.goto("/");

    lp1 = await page.getByTestId("loadpoint").first();
    await lp1
      .getByTestId("change-vehicle")
      .locator("select")
      .selectOption("Vehicle with SoC with Capacity");

    await lp1.getByTestId("charging-plan").getByRole("button", { name: "tomorrow 09:20" }).click();
    await expect(modal.getByTestId("plan-entry")).toHaveCount(2);
    await expect(modal.getByTestId("plan-preview-title")).toHaveText("Next plan #2");
    await expect(modal.getByTestId("target-text")).toContainText("09:20");
    await expect(modal.getByTestId("repeating-plan-precondition-lg-toggle")).toBeChecked();
    await expect(
      modal.getByTestId("repeating-plan-precondition-lg-select").locator("option:checked")
    ).toHaveText("2 hours");
  });
});

// add test for precondition, start with basic.evcc.yaml and verify that precondition toggle element is not visible. make dedicated describe block
test.describe("precondition", async () => {
  test("only if dynamic tariff exists", async ({ page }) => {
    await restart(CONFIG_NO_TARIFF);
    await page.goto("/");
    const lp1 = await page.getByTestId("loadpoint").first();
    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await expect(page.getByTestId("static-plan-active")).toBeVisible();
    await expect(page.getByTestId("static-plan-precondition-lg-toggle")).not.toBeVisible();
    await expect(page.getByTestId("static-plan-precondition-lg-select")).not.toBeVisible();

    // verify small viewport
    await page.setViewportSize(mobile);
    await expect(page.getByTestId("static-plan-precondition-select")).not.toBeVisible();
  });
});
