import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { enableExperimental } from "./utils";

test.use({ baseURL: baseUrl() });

const shortToken = (t: string) => t.substring(0, 6) + "......." + t.substring(t.length - 6);

const EXPIRED_TOKEN =
  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJldmNjLmlvIiwic3ViIjoidHJpYWwiLCJleHAiOjE3NTQ5OTI4MDAsImlhdCI6MTc1MzY5NjgwMCwic3BlIjp0cnVlLCJzcmMiOiJtYSJ9.XKa5DHT-icCM9awcX4eS8feW0J_KIjsx2IxjcRRQOcQ";
const SHORT_TOKEN = shortToken(EXPIRED_TOKEN);

test.afterEach(async () => {
  await stop();
});

test.describe("sponsor token", () => {
  test("token from YAML config", async ({ page }) => {
    await start("sponsor.evcc.yaml");
    await page.goto("/#/config");
    await enableExperimental(page);

    // Check fatal error
    await expect(page.getByTestId("fatal-error")).toContainText("sponsorship");

    // Open sponsor modal
    const sponsorEntry = page.getByTestId("generalconfig-sponsoring");
    await expect(sponsorEntry).toContainText("invalid");
    await sponsorEntry.getByRole("button", { name: "Edit" }).click();

    const modal = page.getByTestId("sponsor-modal");
    const tokenInput = modal.getByRole("textbox", { name: "Your token" });

    await expect(tokenInput).toHaveValue(SHORT_TOKEN);
    await expect(tokenInput).toHaveClass(/is-invalid/);
    await expect(modal.getByText("via evcc.yaml")).toBeVisible();
    await expect(modal.getByRole("button", { name: "Remove" })).not.toBeVisible();
  });

  test("token from database config", async ({ page }) => {
    await start(undefined, "sponsor.sql");
    await page.goto("/#/config");
    await enableExperimental(page);

    // Open sponsor modal
    await page
      .getByTestId("generalconfig-sponsoring")
      .getByRole("button", { name: "Edit" })
      .click();

    const modal = page.getByTestId("sponsor-modal");
    await expect(modal.getByText("via evcc.yaml")).not.toBeVisible();

    // Click change button to reveal textarea
    await modal.getByRole("button", { name: "Change token" }).click();

    await expect(modal.getByRole("textbox", { name: "Enter your token" })).toBeVisible();
    await expect(modal.getByRole("button", { name: "Remove" })).toBeVisible();
  });

  test("insert token in new installation", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page);

    // Open sponsor modal and enter token
    await page
      .getByTestId("generalconfig-sponsoring")
      .getByRole("button", { name: "Edit" })
      .click();

    const modal = page.getByTestId("sponsor-modal");
    const textarea = modal.getByRole("textbox", { name: "Enter your token" });

    await textarea.fill(EXPIRED_TOKEN);
    // Try to save to trigger validation
    await modal.getByRole("button", { name: "Save" }).click();
    await expect(modal).toContainText("token is expired");
  });
});
