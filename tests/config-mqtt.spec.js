import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { enableExperimental } from "./utils";

const CONFIG = "config-grid-only.evcc.yaml";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.beforeEach(async ({ page }) => {
  await start(CONFIG);
  await page.goto("/#/config");
  await enableExperimental(page);
});

test.afterEach(async () => {
  await stop();
});
test.describe("mqtt", async () => {
  test("mqtt not configured", async ({ page }) => {
    await expect(page.getByTestId("mqtt")).toBeVisible();
    await expect(page.getByTestId("mqtt")).toContainText(["Configured", "no"].join(""));
  });

  test("mqtt via ui", async ({ page }) => {
    await page.getByTestId("mqtt").getByRole("button", { name: "edit" }).click();
    const modal = await page.getByTestId("mqtt-modal");

    await modal.getByLabel("Broker").fill("unknown.example.org");
    await modal.getByLabel("Topic").fill("my-topic");
    await modal.getByLabel("Client ID").fill("my-client-id");

    await page.getByRole("button", { name: "Save" }).click();
    await expect(modal.getByTestId("error")).not.toBeVisible();
    await expect(modal).not.toBeVisible();

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart(CONFIG);

    // config error
    await expect(restartButton).not.toBeVisible();
    await expect(page.getByTestId("mqtt")).toHaveClass(/round-box--error/);
    await expect(page.getByTestId("mqtt")).toContainText(
      ["Broker", "unknown.example.org", "Topic", "my-topic"].join("")
    );
    await expect(page.getByTestId("bottom-banner")).toContainText("failed configuring mqtt");

    await page.getByTestId("mqtt").getByRole("button", { name: "edit" }).click();
    await page.getByRole("button", { name: "Remove" }).click();
    await expect(page.getByTestId("mqtt")).toContainText(["Configured", "no"].join(""));
    await expect(restartButton).toBeVisible();
    await restart(CONFIG);
    await expect(restartButton).not.toBeVisible();
    await expect(page.getByTestId("mqtt")).not.toHaveClass(/round-box--error/);
    await expect(page.getByTestId("mqtt")).toContainText(["Configured", "no"].join(""));
  });
});
