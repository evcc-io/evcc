import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  enableExperimental,
  expectModalHidden,
  expectModalVisible,
  openTopNavigation,
  expectTopNavigationClosed,
} from "./utils";

const CONFIG_GRID_ONLY = "config-grid-only.evcc.yaml";
const NETWORK_HOST = "somehostname.local";
const NETWORK_EXTERNAL_URL = "https://ext.evcc.example";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(CONFIG_GRID_ONLY);
});
test.afterAll(async () => {
  await stop();
});

test.describe("basics", async () => {
  test("navigation to config", async ({ page }) => {
    await page.goto("/");
    await openTopNavigation(page);
    await page.getByRole("link", { name: "Configuration" }).click();
    await expectTopNavigationClosed(page);
    await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();
  });
});

test.describe("general", async () => {
  test("change site title", async ({ page }) => {
    // initial value on main ui
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Hello World" })).toBeVisible();

    // change value in config
    await page.goto("/#/config");
    await enableExperimental(page, false);

    await expect(page.getByTestId("generalconfig-title")).toContainText("Hello World");
    await page.getByTestId("generalconfig-title").getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("title-modal");
    await expectModalVisible(modal);
    await modal.getByLabel("Title").fill("Whoops World");

    // close modal and ignore entry on cancel
    await modal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(modal);
    await expect(page.getByTestId("generalconfig-title")).toContainText("Hello World");

    // change and save value
    await page.getByTestId("generalconfig-title").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);
    await modal.getByLabel("Title").fill("Ahoy World");
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);
    await expect(page.getByTestId("generalconfig-title")).toContainText("Ahoy World");

    // check changed value on main ui
    await page.getByTestId("home-link").click();
    await expect(page.getByRole("heading", { name: "Ahoy World" })).toBeVisible();
  });
});

test.describe("network modal", async () => {
  test("persists host and external url across restart", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page, false);

    const networkEntry = page.getByTestId("generalconfig-network");
    await expect(networkEntry).toBeVisible();
    await networkEntry.getByRole("button", { name: "edit" }).click();

    const modal = page.getByTestId("network-modal");
    await expectModalVisible(modal);

    const portValue = await modal.getByLabel("Port").inputValue();
    await modal.getByLabel("External URL").fill(NETWORK_EXTERNAL_URL);
    await modal.getByLabel("mDNS Hostname").fill(NETWORK_HOST);

    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    // values immediatelly visible
    await expect(networkEntry).toContainText(portValue);

    // check persistance
    await restart(CONFIG_GRID_ONLY);
    await page.reload();

    await expect(networkEntry).toBeVisible();
    await networkEntry.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);

    await expect(modal.getByLabel("mDNS Hostname")).toHaveValue(NETWORK_HOST);
    await expect(modal.getByLabel("Port")).toHaveValue(portValue);
    await expect(modal.getByLabel("External URL")).toHaveValue(NETWORK_EXTERNAL_URL);
    const internalUrl = await modal.getByLabel("Internal URL").inputValue();
    expect(internalUrl).toContain(`:${portValue}`);

    await modal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(modal);
  });
});
