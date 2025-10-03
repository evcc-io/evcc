import { test, expect, type Page, type Locator } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden, openTopNavigation, dragElement } from "./utils";

const CONFIG_LOADPOINT_SORT = "loadpoint-sort.evcc.yaml";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.beforeAll(async () => {
  await start(CONFIG_LOADPOINT_SORT);
});

test.afterAll(async () => {
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

async function openModal(page: Page) {
  await openTopNavigation(page);
  await page.getByTestId("topnavigation-settings").click();
  const modal = page.getByTestId("global-settings-modal");
  await expectModalVisible(modal);
  return modal;
}

async function closeModal(modal: Locator) {
  await modal.getByRole("button", { name: "Close" }).click();
  await expectModalHidden(modal);
}

test.describe("loadpoint ordering and hiding", async () => {
  test("initial loadpoint order", async ({ page }) => {
    await expect(page.getByTestId("loadpoint")).toHaveCount(3);

    const loadpoints = page.getByTestId("loadpoint");
    await expect(loadpoints.nth(0)).toContainText("First Loadpoint");
    await expect(loadpoints.nth(1)).toContainText("Second Loadpoint");
    await expect(loadpoints.nth(2)).toContainText("Third Loadpoint");
  });

  test("hide a loadpoint", async ({ page }) => {
    const modal = await openModal(page);

    await modal.getByRole("switch", { name: "Hide Second Loadpoint" }).click();

    await closeModal(modal);

    await expect(page.getByTestId("loadpoint")).toHaveCount(2);
    await expect(page.getByRole("heading", { name: "Second Loadpoint" })).not.toBeVisible();
  });

  test("reorder loadpoints", async ({ page }) => {
    const modal = await openModal(page);

    const firstLoadpointDragItem = modal.getByRole("listitem", {
      name: "Draggable: First Loadpoint",
    });
    const thirdLoadpointDragItem = modal.getByRole("listitem", {
      name: "Draggable: Third Loadpoint",
    });

    await dragElement(page, firstLoadpointDragItem, thirdLoadpointDragItem);

    await closeModal(modal);

    await expect(page.getByTestId("loadpoint")).toHaveCount(3);
    const reorderedLoadpoints = page.getByTestId("loadpoint");
    await expect(reorderedLoadpoints.nth(0)).toContainText("Second Loadpoint");
    await expect(reorderedLoadpoints.nth(1)).toContainText("Third Loadpoint");
    await expect(reorderedLoadpoints.nth(2)).toContainText("First Loadpoint");
  });

  test("reset to initial state", async ({ page }) => {
    const modal = await openModal(page);

    await modal.getByRole("switch", { name: "Hide Second Loadpoint" }).click();

    const firstLoadpointDragItem = modal.getByRole("listitem", {
      name: "Draggable: First Loadpoint",
    });
    const thirdLoadpointDragItem = modal.getByRole("listitem", {
      name: "Draggable: Third Loadpoint",
    });

    await dragElement(page, firstLoadpointDragItem, thirdLoadpointDragItem);

    await modal.getByRole("button", { name: "Reset" }).click();

    await closeModal(modal);

    await expect(page.getByTestId("loadpoint")).toHaveCount(3);
    const loadpoints = page.getByTestId("loadpoint");
    await expect(loadpoints.nth(0)).toContainText("First Loadpoint");
    await expect(loadpoints.nth(1)).toContainText("Second Loadpoint");
    await expect(loadpoints.nth(2)).toContainText("Third Loadpoint");
  });

  test("persist changes after page reload", async ({ page }) => {
    const modal = await openModal(page);

    await modal.getByRole("switch", { name: "Hide Second Loadpoint" }).click();
    await closeModal(modal);

    await page.reload();
    await expect(page.getByTestId("loadpoint")).toHaveCount(2);
    await expect(page.getByRole("heading", { name: "Second Loadpoint" })).not.toBeVisible();
  });
});
