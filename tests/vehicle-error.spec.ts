import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

test.beforeEach(async () => {
  await start("vehicle-error.evcc.yaml");
});

test.afterEach(async () => {
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test.describe("vehicle startup error (using failing Tesla API)", async () => {
  test("broken vehicle: normal title and 'not reachable' icon", async ({ page }) => {
    await expect(page.getByTestId("vehicle-name")).toHaveText("Broken Tesla");
    await expect(page.getByTestId("vehicle-not-reachable-icon")).toBeVisible();

    const moreTab = page.getByTestId("tab-more");
    await moreTab.click();
    await moreTab.getByRole("button", { name: "Vehicles" }).click();
    const modal = page.getByTestId("vehicle-settings-modal");
    await expectModalVisible(modal);
    await expect(modal.getByRole("combobox", { name: "Minimum charge" })).toBeEnabled();
    await expect(modal.getByRole("combobox", { name: "Default limit" })).toBeEnabled();
  });

  test("guest vehicle: normal title and no icon", async ({ page }) => {
    // switch to offline vehicle
    await page.getByRole("combobox", { name: "Change vehicle" }).selectOption("Guest vehicle");

    await expect(page.getByTestId("vehicle-name")).toHaveText("Guest vehicle");
    await expect(page.getByTestId("vehicle-not-reachable-icon")).not.toBeVisible();
  });
});
