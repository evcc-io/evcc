const { test, expect, devices } = require("@playwright/test");
const { start, stop } = require("./evcc");

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
    await expect(page.getByRole("heading", { name: "MAY 2023" })).toBeVisible();
    await expect(page.getByTestId("sessions-nodata")).toHaveCount(0);
    await expect(page.getByRole("table")).toBeVisible();
    await expect(page.getByTestId("sessions-head")).toHaveCount(1);

    await expect(page.getByTestId("sessions-head-energy")).toContainText("ChargedkWh");
    await expect(page.getByTestId("sessions-foot-energy")).toBeVisible("20.0");

    await expect(page.getByTestId("sessions-head-solar")).toContainText("Solar%");
    await expect(page.getByTestId("sessions-foot-solar")).toBeVisible("67.3");

    await expect(page.getByTestId("sessions-head-price")).toContainText("Σ Price€");
    await expect(page.getByTestId("sessions-foot-price")).toBeVisible("5.50");

    await expect(page.getByTestId("sessions-head-avgPrice")).toContainText("Ø Pricect/kWh");
    await expect(page.getByTestId("sessions-foot-avgPrice")).toBeVisible("27.5");

    await expect(page.getByTestId("sessions-entry")).toHaveCount(4);
  });
});

test.describe("mobile basics", async () => {
  test("column select", async ({ page }) => {
    await page.setViewportSize(mobile);
    await page.goto("/#/sessions?year=2023&month=5");

    // hidden columns
    await expect(page.getByTestId("sessions-head-energy")).not.toBeVisible();
    await expect(page.getByTestId("sessions-foot-energy")).not.toBeVisible();
    await expect(page.getByTestId("sessions-head-solar")).not.toBeVisible();
    await expect(page.getByTestId("sessions-foot-solar")).not.toBeVisible();
    await expect(page.getByTestId("sessions-head-price")).not.toBeVisible();
    await expect(page.getByTestId("sessions-foot-price")).not.toBeVisible();
    await expect(page.getByTestId("sessions-head-avgPrice")).not.toBeVisible();
    await expect(page.getByTestId("sessions-foot-avgPrice")).not.toBeVisible();

    await expect(page.getByTestId("sessions-head-mobile")).toContainText("ChargedkWh");
    await expect(page.getByTestId("sessions-foot-mobile")).toBeVisible("20.0");

    await page.getByTestId("mobile-column").selectOption("Solar");
    await expect(page.getByTestId("sessions-head-mobile")).toContainText("Solar%");
    await expect(page.getByTestId("sessions-foot-mobile")).toBeVisible("67.3");

    await page.getByTestId("mobile-column").selectOption("Σ Price");
    await expect(page.getByTestId("sessions-head-mobile")).toContainText("Σ Price€");
    await expect(page.getByTestId("sessions-foot-mobile")).toBeVisible("5.50");

    await page.getByTestId("mobile-column").selectOption("Ø Price");
    await expect(page.getByTestId("sessions-head-mobile")).toContainText("Ø Pricect/kWh");
    await expect(page.getByTestId("sessions-foot-mobile")).toBeVisible("27.5");
  });

  test("keep selection when paging", async ({ page }) => {
    await page.setViewportSize(mobile);
    await page.goto("/#/sessions?year=2023&month=5");

    await page.getByTestId("mobile-column").selectOption("Solar");
    await page.getByRole("link", { name: "Apr" }).click();
    await page.getByRole("link", { name: "May" }).click();
    await expect(page.getByTestId("sessions-head-mobile")).toContainText("Solar%");
  });
});

test.describe("paging", async () => {
  test("prev/next links", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByRole("link", { name: "April" })).toBeVisible();
    await expect(page.getByRole("link", { name: "June" })).toBeVisible();
  });
  test("next month", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await page.getByRole("link", { name: "June" }).click();
    await expect(page.getByRole("heading", { name: "JUNE 2023" })).toBeVisible();
    await expect(page.getByRole("link", { name: "May" })).toBeVisible();
    await expect(page.getByRole("link", { name: "July" })).toBeVisible();
  });
  test("prev month", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=6");
    await page.getByRole("link", { name: "May" }).click();
    await expect(page.getByRole("heading", { name: "MAY 2023" })).toBeVisible();
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
      await page.getByRole("link", { name: "Jun" }).click();
      await page.getByRole("link", { name: "May" }).click();
      await expect(page.getByTestId("sessions-entry")).toHaveCount(3);
    });
  });
}

test.describe("columns desktop", async () => {
  test("show vehicle column if multiple different exist", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("vehicle")).toBeVisible();
  });
  test("hide vehicle column only one exists", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=3");
    await expect(page.getByTestId("vehicle")).toHaveCount(0);
  });
  test("show loadpoint column if multiple different exist", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("loadpoint")).toBeVisible();
  });
  test("hide loadpoint column only one exists", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=3");
    await expect(page.getByTestId("loadpoint")).toHaveCount(0);
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
