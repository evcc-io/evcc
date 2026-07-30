import { test, expect, type Page } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  expectModalVisible,
  expectModalHidden,
  newLoadpoint,
  addDemoCharger,
  finishLoadpoint,
} from "./utils";

const CONFIG = "priority-order.evcc.yaml";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

async function enableExperimental(page: Page) {
  await page
    .getByTestId("generalconfig-experimental")
    .getByRole("button", { name: "edit" })
    .click();
  const modal = page.getByTestId("experimental-modal");
  await expectModalVisible(modal);
  await modal.getByLabel("Enable experimental features.").click();
  await modal.getByRole("button", { name: "Close" }).click();
  await expectModalHidden(modal);
}

async function openPriorityModal(page: Page) {
  await page.getByTestId("loadpoint-priority-button").click();
  const modal = page.getByTestId("priority-modal");
  await expectModalVisible(modal);
  return modal;
}

test.describe("loadpoint priority order", async () => {
  test("move loadpoint to a new tier and persist", async ({ page }) => {
    await start(CONFIG);
    await page.goto("/#/config");
    await enableExperimental(page);

    const modal = await openPriorityModal(page);
    const laneZero = modal.getByRole("list", { name: "Tier 0", exact: true });
    const laneOne = modal.getByRole("list", { name: "Tier 1", exact: true });

    // zero state: lane 0 with both chips, empty lane 1 above as drop target
    await expect(laneZero.getByRole("listitem")).toHaveCount(2);
    await expect(laneOne).toBeVisible();
    await expect(laneOne.getByRole("listitem")).toHaveCount(0);

    // keyboard: move Carport one tier up
    await modal.getByRole("listitem", { name: "Draggable: Carport" }).press("ArrowUp");
    await expect(laneOne.getByRole("listitem")).toHaveCount(1);
    await expect(laneZero.getByRole("listitem")).toHaveCount(1);

    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    // persisted after restart
    await restart(CONFIG);
    await page.goto("/#/config");
    await openPriorityModal(page);
    await expect(laneOne.getByRole("listitem", { name: "Draggable: Carport" })).toBeVisible();
    await expect(laneZero.getByRole("listitem", { name: "Draggable: Garage" })).toBeVisible();
  });

  test("add tier button shows another empty lane", async ({ page }) => {
    await start(CONFIG);
    await page.goto("/#/config");
    await enableExperimental(page);

    const modal = await openPriorityModal(page);

    await expect(modal.getByRole("list", { name: "Tier 2", exact: true })).toHaveCount(0);
    await modal.getByRole("button", { name: "Add tier" }).click();
    await expect(modal.getByRole("list", { name: "Tier 2", exact: true })).toBeVisible();
  });

  test("arrow down keeps values at zero", async ({ page }) => {
    await start(CONFIG);
    await page.goto("/#/config");
    await enableExperimental(page);

    const modal = await openPriorityModal(page);

    await modal.getByRole("listitem", { name: "Draggable: Garage" }).press("ArrowDown");
    await expect(
      modal.getByRole("list", { name: "Tier 0", exact: true }).getByRole("listitem")
    ).toHaveCount(2);
    await expect(modal.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  test("mixed yaml and ui loadpoints", async ({ page }) => {
    await start(CONFIG);
    await page.goto("/#/config");
    await enableExperimental(page);

    // third loadpoint via ui
    await newLoadpoint(page, "Solar Carport");
    await addDemoCharger(page);
    await finishLoadpoint(page);
    await expect(page.getByTestId("loadpoint")).toHaveCount(3);

    await restart(CONFIG);
    await page.reload();

    // reshuffle: Solar Carport -> 2, Garage -> 1
    const modal = await openPriorityModal(page);
    const solarCarport = modal.getByRole("listitem", { name: "Draggable: Solar Carport" });
    await expect(
      modal.getByRole("list", { name: "Tier 0", exact: true }).getByRole("listitem")
    ).toHaveCount(3);
    await modal.getByRole("button", { name: "Add tier" }).click();
    await solarCarport.press("ArrowUp");
    await solarCarport.press("ArrowUp");
    await modal.getByRole("listitem", { name: "Draggable: Garage" }).press("ArrowUp");
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    // loadpoint config modal reflects the new priority
    const lpModal = page.getByTestId("loadpoint-modal");
    await page.getByTestId("loadpoint").nth(2).getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await expect(lpModal.getByLabel("Priority")).toHaveValue("2");
    await lpModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(lpModal);

    // persisted after restart, yaml and db loadpoints alike
    await restart(CONFIG);
    await page.reload();
    await openPriorityModal(page);
    await expect(
      modal
        .getByRole("list", { name: "Tier 2", exact: true })
        .getByRole("listitem", { name: "Draggable: Solar Carport" })
    ).toBeVisible();
    await expect(
      modal
        .getByRole("list", { name: "Tier 1", exact: true })
        .getByRole("listitem", { name: "Draggable: Garage" })
    ).toBeVisible();
    await expect(
      modal
        .getByRole("list", { name: "Tier 0", exact: true })
        .getByRole("listitem", { name: "Draggable: Carport" })
    ).toBeVisible();
  });
});
