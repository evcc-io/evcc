import { test, expect, devices, type Page } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import {
  expectModalVisible,
  expectModalHidden,
  openTopNavigation,
  expectTopNavigationClosed,
} from "./utils";

test.use({ baseURL: baseUrl() });

const mobile = devices["iPhone 12 Mini"].viewport;
const desktop = devices["Desktop Chrome"].viewport;

test.beforeAll(async () => {
  await start("sessions.evcc.yaml", "sessions.sql");
});
test.afterAll(async () => {
  await stop();
});

async function selectLoadpointFilter(page: Page, value: string) {
  await page
    .getByTestId("filter-loadpoint")
    .locator("visible=true")
    .getByRole("combobox")
    .selectOption(value);
}

async function selectVehicleFilter(page: Page, value: string) {
  await page
    .getByTestId("filter-vehicle")
    .locator("visible=true")
    .getByRole("combobox")
    .selectOption(value);
}

test.describe("basics", async () => {
  test("navigation to sessions", async ({ page }) => {
    await page.goto("/");
    await openTopNavigation(page);
    await page
      .getByTestId("topnavigation-dropdown")
      .getByRole("link", { name: "Charging sessions" })
      .click();
    await expectTopNavigationClosed(page);
    await expect(page.getByRole("heading", { name: "Charging Sessions" })).toBeVisible();
  });
  test("month without data", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=6");
    await expect(page.getByTestId("sessions-nodata")).toHaveCount(1);
    await expect(page.getByTestId("sessions-entry")).toHaveCount(0);
  });
  test("month with data", async ({ page }) => {
    await page.setViewportSize(desktop);
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("navigate-month")).toHaveText("May");
    await expect(page.getByTestId("navigate-year")).toHaveText("2023");
    await expect(page.getByTestId("sessions-nodata")).toHaveCount(0);
    await expect(page.getByRole("table")).toBeVisible();
    await expect(page.getByTestId("sessions-head")).toHaveCount(1);
    await expect(page.getByTestId("sessions-head").locator("th")).toHaveCount(9);

    await expect(page.getByTestId("sessions-head-energy")).toContainText("ChargedkWh");
    await expect(page.getByTestId("sessions-foot-energy")).toBeVisible();
    await expect(page.getByTestId("sessions-foot-energy")).toHaveText("20.0");

    await expect(page.getByTestId("sessions-head-solar")).toContainText("Solar%");
    await expect(page.getByTestId("sessions-foot-solar")).toBeVisible();
    await expect(page.getByTestId("sessions-foot-solar")).toHaveText("67.3");

    await expect(page.getByTestId("sessions-head-price")).toContainText("Cost€");
    await expect(page.getByTestId("sessions-foot-price")).toBeVisible();
    await expect(page.getByTestId("sessions-foot-price")).toHaveText("5.50");

    await expect(page.getByTestId("sessions-head-avgPrice")).toContainText("⌀ Pricect/kWh");
    await expect(page.getByTestId("sessions-foot-avgPrice")).toBeVisible();
    await expect(page.getByTestId("sessions-foot-avgPrice")).toHaveText("27.5");

    await expect(page.getByTestId("sessions-head-chargeDuration")).toContainText("Durationh:mm");
    await expect(page.getByTestId("sessions-foot-chargeDuration")).toBeVisible();
    await expect(page.getByTestId("sessions-foot-chargeDuration")).toHaveText("1:30");

    await page
      .getByTestId("sessions-head-chargeDuration")
      .getByRole("combobox")
      .selectOption("⌀ Power");
    await expect(page.getByTestId("sessions-head-avgPower")).toContainText("⌀ PowerkW");
    await expect(page.getByTestId("sessions-foot-avgPower")).toBeVisible();
    await expect(page.getByTestId("sessions-foot-avgPower")).toHaveText("10.0");

    await expect(page.getByTestId("sessions-entry")).toHaveCount(4);
  });
});

test.describe("mobile basics", async () => {
  test("column select", async ({ page }) => {
    await page.setViewportSize(mobile);
    await page.goto("/#/sessions?year=2023&month=5");

    await expect(page.getByTestId("sessions-head").locator("th")).toHaveCount(5);

    await expect(page.getByTestId("sessions-head-energy")).toContainText("ChargedkWh");
    await expect(page.getByTestId("sessions-foot-energy")).toBeVisible();
    await expect(page.getByTestId("sessions-foot-energy")).toHaveText("20.0");

    await page.getByTestId("sessions-head-energy").getByRole("combobox").selectOption("Solar");
    await expect(page.getByTestId("sessions-head-solar")).toContainText("Solar%");
    await expect(page.getByTestId("sessions-foot-solar")).toBeVisible();
    await expect(page.getByTestId("sessions-foot-solar")).toHaveText("67.3");

    await page.getByTestId("sessions-head-solar").getByRole("combobox").selectOption("Cost");
    await expect(page.getByTestId("sessions-head-price")).toContainText("Cost€");
    await expect(page.getByTestId("sessions-foot-price")).toBeVisible();
    await expect(page.getByTestId("sessions-foot-price")).toHaveText("5.50");

    await page.getByTestId("sessions-head-price").getByRole("combobox").selectOption("⌀ Price");
    await expect(page.getByTestId("sessions-head-avgPrice")).toContainText("⌀ Pricect/kWh");
    await expect(page.getByTestId("sessions-foot-avgPrice")).toBeVisible();
    await expect(page.getByTestId("sessions-foot-avgPrice")).toHaveText("27.5");
  });

  test("keep selection when paging", async ({ page }) => {
    await page.setViewportSize(mobile);
    await page.goto("/#/sessions?year=2023&month=5");

    await page.getByTestId("sessions-head-energy").getByRole("combobox").selectOption("Solar");
    await page.getByTestId("navigate-next-year-month").click();
    await page.getByTestId("navigate-prev-year-month").click();
    await expect(page.getByTestId("sessions-head-solar")).toContainText("Solar%");
  });
});

test.describe("paging", async () => {
  test("prev/next links", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("navigate-next-month")).not.toBeDisabled();
    await expect(page.getByTestId("navigate-prev-month")).not.toBeDisabled();
  });
  test("next month", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await page.getByTestId("navigate-next-month").click();
    await expect(page.getByTestId("navigate-month")).toHaveText("June");
    await expect(page.getByTestId("navigate-year")).toHaveText("2023");
    await expect(page.getByTestId("navigate-next-month")).not.toBeDisabled();
    await expect(page.getByTestId("navigate-prev-month")).not.toBeDisabled();
  });
  test("prev month", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=6");
    await page.getByTestId("navigate-prev-month").click();
    await expect(page.getByTestId("navigate-month")).toHaveText("May");
    await expect(page.getByTestId("navigate-year")).toHaveText("2023");
  });
});

for (const [name, viewport] of Object.entries({ desktop, mobile })) {
  test.describe(`filter on ${name}`, async () => {
    test("by vehicle", async ({ page }) => {
      await page.setViewportSize(viewport);
      await page.goto("/#/sessions?year=2023&month=5");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(4);

      await selectVehicleFilter(page, "blauer e-Golf (2)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(2);
      await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/blauer e-Golf/);
      await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/blauer e-Golf/);

      await selectVehicleFilter(page, "weißes Model 3 (2)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(2);
      await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/weißes Model 3/);
      await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/weißes Model 3/);

      await selectVehicleFilter(page, "all vehicles (4)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(4);
    });

    test("by loadpoint", async ({ page }) => {
      await page.setViewportSize(viewport);
      await page.goto("/#/sessions?year=2023&month=5");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(4);

      await selectLoadpointFilter(page, "Carport (3)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(3);
      await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/Carport/);
      await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/Carport/);
      await expect(page.getByTestId("sessions-entry").nth(2)).toHaveText(/Carport/);

      await selectLoadpointFilter(page, "Garage (1)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(1);
      await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/Garage/);

      await selectLoadpointFilter(page, "all charging points (4)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(4);
    });

    test("by vehicle and loadpoint", async ({ page }) => {
      await page.setViewportSize(viewport);
      await page.goto("/#/sessions?year=2023&month=5");

      await selectLoadpointFilter(page, "Carport (3)");
      await selectVehicleFilter(page, "weißes Model 3 (1)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(1);
      await expect(page.getByTestId("sessions-entry")).toHaveText(/Carport/);
      await expect(page.getByTestId("sessions-entry")).toHaveText(/weißes Model 3/);

      await selectVehicleFilter(page, "blauer e-Golf (2)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(2);
      await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/Carport/);
      await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/blauer e-Golf/);
      await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/Carport/);
      await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/blauer e-Golf/);
    });

    test("by vehicle and loadpoint disabled options", async ({ page }) => {
      await page.setViewportSize(viewport);
      await page.goto("/#/sessions?year=2023&month=5");

      await selectVehicleFilter(page, "blauer e-Golf (2)");
      const option = page
        .getByTestId("filter-loadpoint")
        .locator("visible=true")
        .locator("option[value=Garage]");
      await expect(option).toHaveAttribute("disabled", "");
      await expect(option).toHaveText("Garage (0)");
    });

    test("keep filter when paging", async ({ page }) => {
      await page.setViewportSize(viewport);
      await page.goto("/#/sessions?year=2023&month=5");

      await selectLoadpointFilter(page, "Carport (3)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(3);
      if (name === "mobile") {
        await page.getByTestId("navigate-next-year-month").click();
        await page.getByTestId("navigate-prev-year-month").click();
      } else {
        await page.getByTestId("navigate-next-month").click();
        await page.getByTestId("navigate-prev-month").click();
      }
      await expect(page.getByTestId("sessions-entry")).toHaveCount(3);
    });
  });
}

test.describe("columns desktop", async () => {
  test("show vehicle column if multiple different exist", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("vehicle")).toBeVisible();
  });
  test("show loadpoint column if multiple different exist", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("loadpoint")).toBeVisible();
  });
  test("show co2 column it has values", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=3");
    await expect(page.getByTestId("sessions-head-co2")).toBeVisible();
  });
  test("hide co2 column if it doesnt have values", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("sessions-head-co2")).toHaveCount(0);
  });
});

const date = new Date();
const YEAR = date.getFullYear();
const MONTH = date.getMonth() + 1;
const MONTH_YEAR = new Intl.DateTimeFormat("en", { month: "long", year: "numeric" }).format(date);

test.describe("csv export", async () => {
  test("total export", async ({ page }) => {
    await page.goto("/#/sessions?period=total");
    await expect(page.getByRole("link", { name: "Download total CSV" })).toHaveAttribute(
      "href",
      "./api/sessions?format=csv&lang=en"
    );
  });
  test("year export", async ({ page }) => {
    // fixed year
    await page.goto("/#/sessions?period=year&year=2023");
    await expect(page.getByRole("link", { name: "Download 2023 CSV" })).toHaveAttribute(
      "href",
      "./api/sessions?format=csv&lang=en&year=2023"
    );

    // current year
    await page.goto(`/#/sessions?period=year`);
    await expect(page.getByRole("link", { name: `Download ${YEAR} CSV` })).toHaveAttribute(
      "href",
      `./api/sessions?format=csv&lang=en&year=${YEAR}`
    );
  });
  test("monthly export", async ({ page }) => {
    await page.goto("/#/sessions?&year=2023&month=5");
    await expect(page.getByRole("link", { name: "Download May 2023 CSV" })).toHaveAttribute(
      "href",
      "./api/sessions?format=csv&lang=en&year=2023&month=5"
    );

    // current month
    await page.goto(`/#/sessions`);
    await expect(page.getByRole("link", { name: `Download ${MONTH_YEAR} CSV` })).toHaveAttribute(
      "href",
      `./api/sessions?format=csv&lang=en&year=${YEAR}&month=${MONTH}`
    );
  });
});

test.describe("session details", async () => {
  test("show session details (session 5)", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await page.getByTestId("sessions-entry").nth(0).click();
    const modal = page.getByTestId("session-details");
    await expectModalVisible(modal);

    await expect(modal.getByRole("heading", { name: "Charging Session" })).toBeVisible();
    await expect(modal.getByLabel("Charging point")).toHaveValue("Garage");
    await expect(modal.getByLabel("Vehicle")).toHaveValue("weißes Model 3");
    await expect(modal.getByTestId("session-details-date")).toContainText(
      ["Thu, May 4, 22:00", "Fri, May 5, 06:00"].join("")
    );
    await expect(modal.getByTestId("session-details-energy")).toContainText("5.0 kWh");
    await expect(modal.getByTestId("session-details-energy")).toContainText("1:00");
    await expect(modal.getByTestId("session-details-solar")).toContainText("0.0% (0.0 kWh)");
    await expect(modal.getByTestId("session-details-price")).toContainText("2.50 € 50.0 ct/kWh");
    await expect(modal.getByTestId("session-details-co2")).toHaveCount(0);
    await expect(modal.getByTestId("session-details-odometer")).toHaveCount(0);
    await expect(modal.getByTestId("session-details-meter")).toHaveCount(0);
    await expect(modal.getByTestId("session-details-delete")).toContainText("Delete");
  });

  test("show session details with CO2 data (session 1)", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=3");
    await page.getByTestId("sessions-entry").nth(0).click();
    const modal = page.getByTestId("session-details");
    await expectModalVisible(modal);

    await expect(modal.getByLabel("Charging point")).toHaveValue("Carport");
    await expect(modal.getByLabel("Vehicle")).toHaveValue("blauer e-Golf");
    await expect(modal.getByTestId("session-details-date")).toContainText(
      ["Wed, Mar 1, 07:00", "Tue, May 2, 12:00"].join("")
    );
    await expect(modal.getByTestId("session-details-energy")).toContainText("10.0 kWh");
    await expect(modal.getByTestId("session-details-energy")).toContainText("1:00");
    await expect(modal.getByTestId("session-details-solar")).toContainText("100.0% (10.0 kWh)");
    await expect(modal.getByTestId("session-details-price")).toContainText("2.00 € 20.0 ct/kWh");
    await expect(modal.getByTestId("session-details-co2")).toContainText("300 g/kWh");
  });

  test("edit session (session 5)", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await page.getByTestId("sessions-entry").nth(0).click();
    const modal = page.getByTestId("session-details");
    await expectModalVisible(modal);

    await modal.getByLabel("Charging point").selectOption("Carport");
    await modal.getByLabel("Vehicle").selectOption("blauer e-Golf");
    await modal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(modal);

    page.reload();
    await page.getByTestId("sessions-entry").nth(0).click();
    await expectModalVisible(modal);
    await expect(modal.getByLabel("Charging point")).toHaveValue("Carport");
    await expect(modal.getByLabel("Vehicle")).toHaveValue("blauer e-Golf");
  });

  test("delete session (session 5)", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(4);
    await page.getByTestId("sessions-entry").nth(0).click();
    const modal = page.getByTestId("session-details");
    await expectModalVisible(modal);
    await modal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(modal);

    // confirm: cancel
    const confirmModal = page.getByTestId("session-details-confirm");
    await expectModalVisible(confirmModal);
    await confirmModal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(confirmModal);
    await expectModalVisible(modal);

    // confirm: delete
    await modal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(modal);
    await expectModalVisible(confirmModal);
    await confirmModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(confirmModal);
    await expectModalHidden(modal);

    // item removed
    await expect(page.getByTestId("sessions-entry")).toHaveCount(3);
    page.reload();
    await expect(page.getByTestId("sessions-entry")).toHaveCount(3);
  });
});
