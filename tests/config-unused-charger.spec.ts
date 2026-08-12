import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden } from "./utils";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

test.describe("unused charger", async () => {
  test("is listed and can be deleted", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    await expect(page.getByTestId("unused-charger")).toHaveCount(0);

    // a charger no loadpoint references, as left behind when a loadpoint setup is
    // interrupted after the charger was saved
    const res = await page.request.post("./api/config/devices/charger", {
      data: { type: "template", template: "demo-charger", status: "C", power: 11000 },
    });
    expect(res.ok()).toBeTruthy();

    await page.reload();

    const card = page.getByTestId("unused-charger");
    await expect(card).toHaveCount(1);

    // deleting it is only possible from here, no loadpoint leads to it
    await card.getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("charger-modal");
    await expectModalVisible(modal);
    await modal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(modal);

    await expect(page.getByTestId("unused-charger")).toHaveCount(0);
  });
});
