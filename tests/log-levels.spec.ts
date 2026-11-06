import { test, expect, type Page } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

const checkArea = async (page: Page, area: string) => {
  await page.getByRole("checkbox", { name: area }).check();
  await expect(page.getByRole("button", { name: area }).first()).toBeVisible();
};

const saveAreas = async (page: Page, level: string) => {
  await page.getByLabel("Filter by log level").selectOption(level);
  const save = page.getByRole("button", {
    name: "Save this level for the selected areas in the system log",
  });
  await save.click();
  await expect(save).toBeDisabled();
};

test.describe("log levels", async () => {
  test("area override", async ({ page }) => {
    await start("basics.evcc.yaml");
    await page.goto("/#/log");
    await page.getByRole("button", { name: "All areas" }).click();
    await checkArea(page, "lp-1");
    await saveAreas(page, "trace");

    await restart("basics.evcc.yaml");
    await page.goto("about:blank");
    await page.goto("/#/log");
    await page.getByRole("button", { name: "All areas" }).click();
    await expect(page.getByText("Stored level: TRACE")).toBeAttached();

    // default level removes the override
    await checkArea(page, "lp-1");
    await saveAreas(page, "info");
    await expect(page.getByText("Stored level: TRACE")).not.toBeAttached();
  });

  test("default level", async ({ page }) => {
    await start("basics.evcc.yaml");
    await page.goto("/#/log");
    const save = page.getByRole("button", {
      name: "Save this level as default for the system log",
    });
    await page.getByLabel("Filter by log level").selectOption("trace");
    await save.click();
    await expect(save).toBeDisabled();
    await expect(page.getByRole("option", { name: "TRACE ●" })).toBeAttached();
  });

  test("unavailable area", async ({ page }) => {
    await start("basics.evcc.yaml");
    await page.request.post("/api/config/logLevels", {
      data: { default: "info", levels: { "lp-9": "trace" } },
    });
    await page.goto("/#/log");
    await page.getByRole("button", { name: "All areas" }).click();
    await checkArea(page, "lp-1");
    await saveAreas(page, "debug");

    const state = await (await page.request.get("/api/state")).json();
    expect(state.logLevels.levels).toEqual({ "lp-1": "DEBUG" });
  });
});
