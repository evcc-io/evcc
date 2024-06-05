import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";

const CONFIG_EMPTY = "config-empty.evcc.yaml";
const CONFIG_WITH_TARIFFS = "config-with-tariffs.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

async function login(page) {
  await page.locator("#loginPassword").fill("secret");
  await page.getByRole("button", { name: "Login" }).click();
}

async function enableExperimental(page) {
  await page
    .getByTestId("generalconfig-experimental")
    .getByRole("button", { name: "edit" })
    .click();
  await page.getByLabel("Experimental 🧪").click();
  await page.getByRole("button", { name: "Close" }).click();
}

async function goToConfig(page) {
  await page.goto("/#/config");
  await login(page);
  await enableExperimental(page);
}

test.describe("tariffs", async () => {
  test("tariffs not configured", async ({ page }) => {
    await start(CONFIG_EMPTY, "password.sql");
    await goToConfig(page);

    await expect(page.getByTestId("tariffs")).toBeVisible();
    await expect(page.getByTestId("tariffs")).toContainText(
      ["Tariffs", "Currency", "EUR"].join("")
    );
  });

  test("configure tariff", async ({ page }) => {
    await start(CONFIG_EMPTY, "password.sql");
    await goToConfig(page);

    await page.getByTestId("tariffs").getByRole("button", { name: "edit" }).click();
    const modal = await page.getByTestId("tariffs-modal");
    await expect(modal).toBeVisible();

    // default content
    await expect(modal).toContainText("# currency: EUR");

    await modal.locator(".monaco-editor").click();
    // remove existing content
    await page.keyboard.press("Meta+KeyA");
    await page.keyboard.press("Backspace");
    await page.keyboard.press("Meta+KeyA");
    await page.keyboard.press("Backspace");
    // enter new content
    await page.keyboard.type("currency: CHF\n");
    await page.keyboard.type("grid:\n");
    await page.keyboard.type("  type: fixed\n");
    await page.keyboard.type("price: 0.123\n");

    await page.getByRole("button", { name: "Save" }).click();

    // modal closes
    await expect(modal).not.toBeVisible();

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart(CONFIG_EMPTY);

    // restart done
    await expect(restartButton).not.toBeVisible();

    await expect(page.getByTestId("tariffs")).toContainText(
      ["Currency", "CHF", "Grid price", "12.3 rp"].join("")
    );
  });

  test("show from evcc.yaml", async ({ page }) => {
    await start(CONFIG_WITH_TARIFFS, "password.sql");
    await goToConfig(page);

    await expect(page.getByTestId("tariffs")).toBeVisible();
    await expect(page.getByTestId("tariffs")).toContainText(
      [
        "Tariffs",
        "Currency",
        "SEK",
        "Grid price",
        "0.3 SEK",
        "Feed-in price",
        "-0.1 SEK",
        "Grid CO₂",
        "300 g",
      ].join("")
    );
  });
});
