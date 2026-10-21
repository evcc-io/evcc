import { test, expect, type Page } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden } from "./utils";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

// test runs start with the command line level debug,httpd:trace
const overrides = (page: Page, count: number) =>
  page.getByRole("button", { name: `${count} log level overrides active` });

const addOverride = async (page: Page, area: string, level: string) => {
  await page.getByRole("button", { name: "Log levels" }).click();
  const modal = page.getByTestId("loglevels-modal");
  await expectModalVisible(modal);
  await modal.getByRole("button", { name: "Add area override" }).click();
  await modal.getByLabel("Area", { exact: true }).last().selectOption(area);
  await modal.getByLabel("Level", { exact: true }).last().selectOption(level);
  await modal.getByRole("button", { name: "Save" }).click();
  await expectModalHidden(modal);
};

test.describe("log levels", async () => {
  test("override", async ({ page }) => {
    await start("basics.evcc.yaml");
    await page.goto("/#/log");
    await expect(overrides(page, 1)).toBeVisible();

    await addOverride(page, "lp-1", "TRACE");
    await expect(overrides(page, 2)).toBeVisible();

    await restart("basics.evcc.yaml");
    await page.goto("/#/log");
    await expect(overrides(page, 2)).toBeVisible();
  });

  test("reset", async ({ page }) => {
    await start("basics.evcc.yaml");
    await page.goto("/#/log");

    await addOverride(page, "lp-1", "TRACE");
    await expect(overrides(page, 2)).toBeVisible();

    await page.getByRole("button", { name: "Log levels" }).click();
    const modal = page.getByTestId("loglevels-modal");
    await expectModalVisible(modal);
    await modal.getByRole("button", { name: "Reset" }).click();
    await expectModalHidden(modal);
    await expect(overrides(page, 1)).toBeVisible();
  });
});

test.describe("log levels dialog", async () => {
  test("edit", async ({ page }) => {
    await start("basics.evcc.yaml");
    await page.goto("/#/log");

    await page.getByRole("button", { name: "Log levels" }).click();
    const modal = page.getByTestId("loglevels-modal");
    await expectModalVisible(modal);

    const save = modal.getByRole("button", { name: "Save" });
    const areas = modal.getByLabel("Area", { exact: true });
    await expect(save).toBeDisabled();
    await expect(areas).toHaveCount(1);

    // areas used by other rows are not offered again
    await modal.getByRole("button", { name: "Add area override" }).click();
    await expect(areas).toHaveCount(2);
    await expect(areas.last().getByRole("option", { name: "httpd" })).toHaveCount(0);
    await expect(areas.first().getByRole("option", { name: "httpd" })).toHaveCount(1);

    // a row needs area and level before saving
    await areas.last().selectOption("lp-1");
    await expect(save).toBeDisabled();
    await modal.getByLabel("Level", { exact: true }).last().selectOption("TRACE");
    await expect(save).toBeEnabled();

    // removing rows discards the override once saved
    await modal.getByRole("button", { name: "Remove" }).last().click();
    await modal.getByRole("button", { name: "Remove" }).first().click();
    await expect(areas).toHaveCount(0);
    await expect(save).toBeEnabled();
    await save.click();
    await expectModalHidden(modal);
    await expect(page.getByRole("button", { name: "log level overrides active" })).toHaveCount(0);
  });

  test("cancel", async ({ page }) => {
    await start("basics.evcc.yaml");
    await page.goto("/#/log");

    await page.getByRole("button", { name: "Log levels" }).click();
    const modal = page.getByTestId("loglevels-modal");
    await expectModalVisible(modal);
    await modal.getByLabel("Default level").selectOption("TRACE");
    await expect(modal.getByRole("button", { name: "Save" })).toBeEnabled();
    await modal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(modal);

    await page.getByRole("button", { name: "Log levels" }).click();
    await expectModalVisible(modal);
    await expect(modal.getByLabel("Default level")).toHaveValue("DEBUG");
    await expect(modal.getByRole("button", { name: "Save" })).toBeDisabled();
  });
});

test.describe("log levels stale", async () => {
  test("unavailable area", async ({ page }) => {
    await start("basics.evcc.yaml");
    await page.goto("/#/log");
    await page.request.post("/api/config/logLevels", {
      data: { default: "debug", levels: { "lp-9": "trace" } },
    });

    await page.getByRole("button", { name: "Log levels" }).click();
    const modal = page.getByTestId("loglevels-modal");
    await expectModalVisible(modal);
    await expect(
      modal.getByRole("option", { name: "lp-9 (not available)", selected: true })
    ).toBeAttached();
  });
});
