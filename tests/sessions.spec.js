import { test, expect, devices } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

const mobile = devices["iPhone 12 Mini"].viewport;
const desktop = devices["Desktop Chrome"].viewport;

test.beforeAll(async () => {
  await start("basics.evcc.yaml", "sessions.sql");
});
test.afterAll(async () => {
  await stop();
});

test.describe("basics", async () => {
  test("navigation to sessions", async ({ page }) => {
    await page.goto("/");
    const topNavigationButton = await page.getByTestId("topnavigation-button");
    await topNavigationButton.click();
    await page
      .getByTestId("topnavigation-dropdown")
      .getByRole("link", { name: "Charging sessions" })
      .click();
    await expect(page.getByRole("heading", { name: "Charging Sessions" })).toBeVisible();
  });
  test("month without data", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=6");
    await expect(page.getByTestId("sessions-nodata")).toHaveCount(1);
    await expect(page.getByTestId("sessions-entry")).toHaveCount(0);
  });
  test("month with data", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("navigate-month")).toHaveText("May");
    await expect(page.getByTestId("navigate-year")).toHaveText("2023");
    await expect(page.getByTestId("sessions-nodata")).toHaveCount(0);
    await expect(page.getByRole("table")).toBeVisible();
    await expect(page.getByTestId("sessions-head")).toHaveCount(1);
    await expect(page.getByTestId("sessions-head").locator("th")).toHaveCount(9);

    await expect(page.getByTestId("sessions-head-energy")).toContainText("ChargedkWh");
    await expect(page.getByTestId("sessions-foot-energy")).toBeVisible("20.0");

    await expect(page.getByTestId("sessions-head-solar")).toContainText("Solar%");
    await expect(page.getByTestId("sessions-foot-solar")).toBeVisible("67.3");

    await expect(page.getByTestId("sessions-head-price")).toContainText("Cost€");
    await expect(page.getByTestId("sessions-foot-price")).toBeVisible("5.50");

    await expect(page.getByTestId("sessions-head-avgPrice")).toContainText("⌀ Pricect/kWh");
    await expect(page.getByTestId("sessions-foot-avgPrice")).toBeVisible("27.5");

    await expect(page.getByTestId("sessions-head-chargeDuration")).toContainText("Durationh:mm");
    await expect(page.getByTestId("sessions-foot-chargeDuration")).toBeVisible("1:30");

    await page
      .getByTestId("sessions-head-chargeDuration")
      .getByRole("combobox")
      .selectOption("⌀ Power");
    await expect(page.getByTestId("sessions-head-avgPower")).toContainText("⌀ PowerkW");
    await expect(page.getByTestId("sessions-foot-avgPower")).toBeVisible("1:30");

    await expect(page.getByTestId("sessions-entry")).toHaveCount(4);
  });
});

test.describe("mobile basics", async () => {
  test("column select", async ({ page }) => {
    await page.setViewportSize(mobile);
    await page.goto("/#/sessions?year=2023&month=5");

    await expect(page.getByTestId("sessions-head").locator("th")).toHaveCount(5);

    await expect(page.getByTestId("sessions-head-energy")).toContainText("ChargedkWh");
    await expect(page.getByTestId("sessions-foot-energy")).toBeVisible("20.0");

    await page.getByTestId("sessions-head-energy").getByRole("combobox").selectOption("Solar");
    await expect(page.getByTestId("sessions-head-solar")).toContainText("Solar%");
    await expect(page.getByTestId("sessions-foot-solar")).toBeVisible("67.3");

    await page.getByTestId("sessions-head-solar").getByRole("combobox").selectOption("Cost");
    await expect(page.getByTestId("sessions-head-price")).toContainText("Cost€");
    await expect(page.getByTestId("sessions-foot-price")).toBeVisible("5.50");

    await page.getByTestId("sessions-head-price").getByRole("combobox").selectOption("⌀ Price");
    await expect(page.getByTestId("sessions-head-avgPrice")).toContainText("⌀ Pricect/kWh");
    await expect(page.getByTestId("sessions-foot-avgPrice")).toBeVisible("27.5");
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

      await page
        .getByTestId("filter-vehicle")
        .locator("visible=true")
        .selectOption("blauer e-Golf (2)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(2);
      await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/blauer e-Golf/);
      await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/blauer e-Golf/);

      await page
        .getByTestId("filter-vehicle")
        .locator("visible=true")
        .selectOption("weißes Model 3 (2)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(2);
      await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/weißes Model 3/);
      await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/weißes Model 3/);

      await page
        .getByTestId("filter-vehicle")
        .locator("visible=true")
        .selectOption("all vehicles (4)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(4);
    });

    test("by loadpoint", async ({ page }) => {
      await page.setViewportSize(viewport);
      await page.goto("/#/sessions?year=2023&month=5");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(4);

      await page
        .getByTestId("filter-loadpoint")
        .locator("visible=true")
        .selectOption("Carport (3)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(3);
      await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/Carport/);
      await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/Carport/);
      await expect(page.getByTestId("sessions-entry").nth(2)).toHaveText(/Carport/);

      await page.getByTestId("filter-loadpoint").locator("visible=true").selectOption("Garage (1)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(1);
      await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/Garage/);

      await page
        .getByTestId("filter-loadpoint")
        .locator("visible=true")
        .selectOption("all charging points (4)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(4);
    });

    test("by vehicle and loadpoint", async ({ page }) => {
      await page.setViewportSize(viewport);
      await page.goto("/#/sessions?year=2023&month=5");

      await page
        .getByTestId("filter-loadpoint")
        .locator("visible=true")
        .selectOption("Carport (3)");
      await page
        .getByTestId("filter-vehicle")
        .locator("visible=true")
        .selectOption("weißes Model 3 (1)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(1);
      await expect(page.getByTestId("sessions-entry")).toHaveText(/Carport/);
      await expect(page.getByTestId("sessions-entry")).toHaveText(/weißes Model 3/);

      await page
        .getByTestId("filter-vehicle")
        .locator("visible=true")
        .selectOption("blauer e-Golf (2)");
      await expect(page.getByTestId("sessions-entry")).toHaveCount(2);
      await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/Carport/);
      await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/blauer e-Golf/);
      await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/Carport/);
      await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/blauer e-Golf/);
    });

    test("by vehicle and loadpoint disabled options", async ({ page }) => {
      await page.setViewportSize(viewport);
      await page.goto("/#/sessions?year=2023&month=5");

      await page
        .getByTestId("filter-vehicle")
        .locator("visible=true")
        .selectOption("blauer e-Golf (2)");
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

      await page
        .getByTestId("filter-loadpoint")
        .locator("visible=true")
        .selectOption("Carport (3)");
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

test.describe("csv export", async () => {
  test("total export", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByRole("link", { name: "Download total CSV" })).toHaveAttribute(
      "href",
      "./api/sessions?format=csv&lang=en"
    );

    await page.goto("/#/sessions?year=2023&month=6");
    await expect(page.getByRole("link", { name: "Download total CSV" })).toHaveAttribute(
      "href",
      "./api/sessions?format=csv&lang=en"
    );
  });
  test("monthly export", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByRole("link", { name: "Download May 2023 CSV" })).toHaveAttribute(
      "href",
      "./api/sessions?format=csv&lang=en&year=2023&month=5"
    );

    await page.goto("/#/sessions?year=2023&month=6");
    await expect(page.getByRole("link", { name: "Download June 2023 CSV" })).toHaveCount(0);
  });
});
