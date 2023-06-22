const { test, expect } = require("@playwright/test");
const { start, stop } = require("./evcc");

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
    await expect(page.getByText("Charged 20.0 kWh")).toBeVisible();
    await expect(page.getByText("Solar 67.3%")).toBeVisible();
    await expect(page.getByText("Σ Price 5.50 €")).toBeVisible();
    await expect(page.getByText("Ø Price 27.5 ct/kWh")).toBeVisible();
    await expect(page.getByTestId("sessions-entry")).toHaveCount(4);
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

test.describe("filter", async () => {
  test("by vehicle", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(4);

    await page.getByLabel("all vehicles").selectOption("blauer e-Golf (2)");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(2);
    await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/blauer e-Golf/);
    await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/blauer e-Golf/);

    await page.getByLabel("all vehicles").selectOption("weißes Model 3 (2)");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(2);
    await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/weißes Model 3/);
    await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/weißes Model 3/);

    await page.getByLabel("all vehicles").selectOption("all vehicles (4)");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(4);
  });

  test("by loadpoint", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(4);

    await page.getByLabel("all charging points").selectOption("Carport (3)");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(3);
    await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/Carport/);
    await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/Carport/);
    await expect(page.getByTestId("sessions-entry").nth(2)).toHaveText(/Carport/);

    await page.getByLabel("all charging points").selectOption("Garage (1)");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(1);
    await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/Garage/);

    await page.getByLabel("all charging points").selectOption("all charging points (4)");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(4);
  });

  test("by vehicle and loadpoint", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");

    await page.getByLabel("all charging points").selectOption("Carport (3)");
    await page.getByLabel("all vehicles").selectOption("weißes Model 3 (1)");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(1);
    await expect(page.getByTestId("sessions-entry")).toHaveText(/Carport/);
    await expect(page.getByTestId("sessions-entry")).toHaveText(/weißes Model 3/);

    await page.getByLabel("all vehicles").selectOption("blauer e-Golf (2)");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(2);
    await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/Carport/);
    await expect(page.getByTestId("sessions-entry").nth(0)).toHaveText(/blauer e-Golf/);
    await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/Carport/);
    await expect(page.getByTestId("sessions-entry").nth(1)).toHaveText(/blauer e-Golf/);
  });

  test("by vehicle and loadpoint disabled options", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");

    await page.getByLabel("all vehicles").selectOption("blauer e-Golf (2)");
    const option = page.locator("option[value=Garage]");
    await expect(option).toHaveAttribute("disabled", "");
    await expect(option).toHaveText("Garage (0)");
  });

  test("keep filter when paging", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");

    await page.getByLabel("all charging points").selectOption("Carport (3)");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(3);
    await page.getByRole("link", { name: "June" }).click();
    await page.getByRole("link", { name: "May" }).click();
    await expect(page.getByTestId("sessions-entry")).toHaveCount(3);
  });
});

test.describe("columns", async () => {
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
    await expect(page.getByTestId("co2")).toBeVisible();
  });
  test("hide co2 column if it doesnt have values", async ({ page }) => {
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("co2")).toHaveCount(0);
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
