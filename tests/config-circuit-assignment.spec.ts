import { test, expect, type Page } from "@playwright/test";
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

test("device-first assignment, rollback and restart persistence", async ({ page }) => {
  await start("fast.evcc.yaml", "config-circuit-assignment.sql");
  const modal = await open(page);
  const pool = modal.getByRole("list", { name: "Not assigned", exact: true });
  const row = modal.getByRole("listitem", { name: "Wallbox left" });
  const rowIn = (list: string) =>
    modal
      .getByRole("list", { name: list, exact: true })
      .getByRole("listitem", { name: "Wallbox left" });
  const circuit = (title: string) =>
    modal.getByRole("button", { name: `Circuit ${title}`, exact: true });

  await expect(rowIn("Not assigned")).toBeVisible();
  await row.click();
  await circuit("Garage").click();
  await expect(rowIn("Garage")).toBeVisible();

  await row.press("Enter");
  await expect(row).toHaveAttribute("aria-selected", "true");
  await page.route("**/api/config/loadpoints/13", (route) =>
    route.fulfill({ status: 400, contentType: "application/json", body: '{"error":"rejected"}' })
  );
  await circuit("Home").click();
  await expect(modal.getByRole("alert")).toContainText("Could not save assignment");
  await expect(rowIn("Garage")).toBeVisible();

  await page.unroute("**/api/config/loadpoints/13");
  await pool.press("Space");
  await expect(rowIn("Not assigned")).toBeVisible();
  await row.click();
  await circuit("Home").click();
  await expect(rowIn("Home")).toBeVisible();

  await row.dragTo(circuit("Garage"));
  await expect(rowIn("Garage")).toBeVisible();

  await row.dragTo(pool);
  await expect(rowIn("Not assigned")).toBeVisible();

  await row.dragTo(circuit("Home"));
  await expect(rowIn("Home")).toBeVisible();

  await modal.getByRole("button", { name: "Close", exact: true }).last().click();
  await expect(modal).not.toBeVisible();
  await page.goto("about:blank");
  await restart("fast.evcc.yaml");
  await open(page);
  await expect(rowIn("Home")).toBeVisible();
});
