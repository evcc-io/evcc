import { test, expect, type Page } from "@playwright/test";
import { execFileSync } from "node:child_process";
import os from "node:os";
import path from "node:path";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

const open = async (page: Page) => {
  await page.goto("/#/config");
  const modal = page.getByTestId("circuits-modal");
  if (!(await modal.isVisible()))
    await page.getByTestId("circuits").getByRole("button", { name: "edit" }).click();
  await expectModalVisible(modal);
  return modal;
};

const stored = () => {
  const rows = JSON.parse(
    execFileSync(
      "sqlite3",
      [
        "-json",
        path.join(os.tmpdir(), `evcc-${new URL(baseUrl()).port}.db`),
        "SELECT value FROM configs WHERE id = 13",
      ],
      { encoding: "utf8" }
    )
  );
  return JSON.parse(rows[0].value);
};

test("device-first assignment, rollback and restart persistence", async ({ page }) => {
  await start("fast.evcc.yaml", "config-circuit-assignment.sql");
  const modal = await open(page);
  const pool = modal.getByRole("list", { name: "Not assigned", exact: true });
  const row = modal.getByRole("listitem", { name: "Wallbox left" });

  await row.click();
  await expect(pool.getByRole("listitem", { name: "Wallbox left" })).toBeVisible();

  const assignmentResponse = page.waitForResponse(
    (response) =>
      response.url().endsWith("/api/config/loadpoints/13") &&
      response.request().method() === "PATCH"
  );
  await modal.getByRole("button", { name: "Circuit Garage", exact: true }).click();
  expect((await assignmentResponse).status()).toBe(200);
  await expect(pool.getByRole("listitem", { name: "Wallbox left" })).toHaveCount(0);
  expect(stored()).toMatchObject({
    circuit: "db:9",
    minSoc: 35,
    priority: 4,
    solarShare: 0.6,
  });

  await row.press("Enter");
  await expect(row).toHaveAttribute("aria-selected", "true");
  await page.route("**/api/config/loadpoints/13", (route) =>
    route.fulfill({ status: 400, contentType: "application/json", body: '{"error":"rejected"}' })
  );
  await modal.getByRole("button", { name: "Circuit Home", exact: true }).click();
  await expect(modal.getByRole("alert")).toContainText("Could not save assignment");
  expect(stored().circuit).toBe("db:9");
  await expect(row).not.toHaveClass(/assignment-flash/);

  await page.unroute("**/api/config/loadpoints/13");
  await pool.press("Space");
  await expect(pool.getByRole("listitem", { name: "Wallbox left" })).toBeVisible();
  await row.click();
  await modal.getByRole("button", { name: "Circuit Home", exact: true }).click();
  expect(stored().circuit).toBe("db:8");

  await row.dragTo(modal.getByRole("button", { name: "Circuit Garage", exact: true }));
  await expect.poll(() => stored().circuit).toBe("db:9");

  await row.dragTo(pool);
  await expect(pool.getByRole("listitem", { name: "Wallbox left" })).toBeVisible();
  expect(stored().circuit).toBe("");

  await row.dragTo(modal.getByRole("button", { name: "Circuit Home", exact: true }));
  await expect.poll(() => stored().circuit).toBe("db:8");

  await modal.getByRole("button", { name: "Close", exact: true }).last().click();
  await expect(modal).not.toBeVisible();
  await page.goto("about:blank");
  await restart("fast.evcc.yaml");
  await page.goto("/#/config");
  await page.reload();
  await open(page);

  await expect(
    page.getByTestId("assignment-pool").getByRole("listitem", { name: "Wallbox left" })
  ).toHaveCount(0);
  const response = await page.request.get("/api/config/loadpoints/13");
  expect((await response.json()).circuit).toBe("db:8");
});
