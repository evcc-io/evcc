import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  expectModalVisible,
  expectModalHidden,
  editorClear,
  editorPaste,
  enableAppContext,
  expectAppEvent,
} from "./utils";
import { startSimulator, stopSimulator, simulatorUrl, simulatorApply } from "./simulator";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

const CONFIG = "fast.evcc.yaml";

test.describe("HEMS", () => {
  test("grid sessions in template modal", async ({ page }) => {
    await start(CONFIG, "hems.sql");
    await page.goto("/#/config");

    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    const hemsModal = page.getByTestId("hems-modal");
    await expectModalVisible(hemsModal);

    await expect(hemsModal.getByTestId("grid-sessions")).toBeVisible();
    await expect(hemsModal).toContainText("Recorded 3 grid limitation events");
    await expect(hemsModal).toContainText("Most recent");

    const csvLink = hemsModal.getByRole("link", { name: "Download CSV" });
    await expect(csvLink).toHaveAttribute("href", /\.\/api\/gridsessions\?format=csv&lang=en/);

    // template selector is shown in create mode, yaml editor hidden until "User-defined" is chosen
    await expect(hemsModal.getByLabel("Provider")).toBeVisible();
    await expect(hemsModal.getByTestId("yaml-editor")).not.toBeVisible();
    await expect(hemsModal).not.toContainText("Configured via evcc.yaml");

    await hemsModal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(hemsModal);
  });

  test("modal yaml-configured is locked", async ({ page }) => {
    await start("hems-yaml.evcc.yaml");
    await page.goto("/#/config");

    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    const hemsModal = page.getByTestId("hems-modal");
    await expectModalVisible(hemsModal);

    await expect(hemsModal.getByTestId("grid-sessions")).not.toBeVisible();
    await expect(hemsModal).toContainText("Configured via evcc.yaml");
    await expect(hemsModal.getByLabel("Provider")).not.toBeVisible();
    await expect(hemsModal.getByTestId("yaml-editor")).not.toBeVisible();
    await expect(hemsModal.getByRole("button", { name: "Save" })).not.toBeVisible();
    await expect(hemsModal.getByRole("button", { name: "Cancel" })).toBeVisible();

    await hemsModal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(hemsModal);
  });

  test("user-defined relay drives external limit without circuits", async ({ page }) => {
    const GRID_CONFIG = "hems-grid.evcc.yaml";
    await startSimulator();
    await start(GRID_CONFIG);

    await page.goto("/#/config");

    // configure hems via user-defined provider
    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    const hemsModal = page.getByTestId("hems-modal");
    await expectModalVisible(hemsModal);
    await hemsModal.getByLabel("Provider").selectOption({ label: "User-defined device" });
    const hemsEditor = hemsModal.getByTestId("yaml-editor");
    await expect(hemsEditor).toBeVisible();
    await editorClear(hemsEditor);
    await editorPaste(
      hemsEditor,
      page,
      `type: relay
maxPower: 4200
interval: 0.1s
limit:
  source: http
  uri: ${simulatorUrl()}/api/state
  jq: .hems.relay`
    );
    await hemsModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(hemsModal);

    await restart(GRID_CONFIG);
    await page.goto("/#/config");

    // verify external control is the only circuit visible
    await expect(page.getByTestId("circuits")).toContainText(
      ["External Limit", "Power", "0.0 kW"].join("")
    );

    // enable hems in simulator
    await page.goto(simulatorUrl());
    const hemsRelayCheckbox = page.getByLabel("Relay Limit");
    await hemsRelayCheckbox.check();
    await simulatorApply(page);

    // verify config ui
    await page.goto("/#/config");
    await expect(page.getByTestId("circuits")).toContainText(
      ["External Limit", "Consumption limited", "yes", "Power", "0.0 kW / 4.2 kW"].join("")
    );

    // disable in simulator
    await page.goto(simulatorUrl());
    await hemsRelayCheckbox.uncheck();
    await simulatorApply(page);

    await stopSimulator();
  });

  test("change button resets modal and unconfigures the card", async ({ page }) => {
    await start("hems-grid.evcc.yaml");
    await page.goto("/#/config");

    // configure hems via user-defined
    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    const hemsModal = page.getByTestId("hems-modal");
    await expectModalVisible(hemsModal);
    await hemsModal.getByLabel("Provider").selectOption({ label: "User-defined device" });
    const hemsEditor = hemsModal.getByTestId("yaml-editor");
    await editorClear(hemsEditor);
    await editorPaste(
      hemsEditor,
      page,
      `type: relay
maxPower: 4200
limit:
  source: const
  value: false`
    );
    await hemsModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(hemsModal);

    // card now configured
    await expect(page.getByTestId("hems")).toContainText("Communication");

    // reopen modal in edit mode
    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(hemsModal);
    await expect(hemsEditor).toBeVisible();
    await expect(hemsModal.getByLabel("Provider")).toBeDisabled();

    // click pen (change) and confirm
    page.once("dialog", (d) => d.accept());
    await hemsModal.getByRole("button", { name: "Change" }).click();

    // provider selector reappears, yaml editor gone
    await expect(hemsModal.getByLabel("Provider")).toBeEnabled();
    await expect(hemsEditor).not.toBeVisible();
    await expect(hemsModal.getByRole("button", { name: "Change" })).not.toBeVisible();

    // close modal, card shows unconfigured
    await hemsModal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(hemsModal);
    await expect(page.getByTestId("hems")).toContainText(["Configured", "no"].join(""));
  });

  test.describe("grid sessions CSV in app context", () => {
    test("dispatches download event", async ({ page }) => {
      await enableAppContext(page);
      await start(CONFIG, "hems.sql");
      await page.goto("/#/config");

      await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
      const hemsModal = page.getByTestId("hems-modal");
      await expectModalVisible(hemsModal);

      const csvLink = hemsModal.getByRole("link", { name: "Download CSV" });
      await csvLink.click();
      expect(await expectAppEvent(page)).toMatchObject({
        type: "download",
        url: expect.stringContaining("/api/gridsessions?format=csv&lang=en"),
      });
    });
  });

  test("user-defined relay drives external limit with circuits", async ({ page }) => {
    const GRID_CONFIG = "hems-grid.evcc.yaml";
    await start(GRID_CONFIG);

    await page.goto("/#/config");

    // configure circuits
    await page.getByTestId("circuits").getByRole("button", { name: "edit" }).click();
    const circuitsModal = page.getByTestId("circuits-modal");
    await expectModalVisible(circuitsModal);
    const circuitsEditor = circuitsModal.getByTestId("yaml-editor");
    await editorClear(circuitsEditor);
    await editorPaste(circuitsEditor, page, `- name: main\n  title: House`);
    await circuitsModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(circuitsModal);

    // configure hems via user-defined provider
    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    const hemsModal = page.getByTestId("hems-modal");
    await expectModalVisible(hemsModal);
    await hemsModal.getByLabel("Provider").selectOption({ label: "User-defined device" });
    const hemsEditor = hemsModal.getByTestId("yaml-editor");
    await expect(hemsEditor).toBeVisible();
    await editorClear(hemsEditor);
    await editorPaste(
      hemsEditor,
      page,
      `type: relay
maxPower: 4200
interval: 0.1s
limit:
  source: const
  value: true`
    );
    await hemsModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(hemsModal);

    await restart(GRID_CONFIG);
    await page.goto("/#/config");

    // verify external control is the top-most circuit, with main beneath it
    await expect(page.getByTestId("circuits")).toContainText(
      [
        "External Limit",
        "Consumption limited",
        "yes",
        "Power",
        "0.0 kW / 4.2 kW",
        "House",
        "Consumption limited",
        "yes",
        "Power",
        "0.0 kW",
      ].join("")
    );
  });
});
