import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  expectModalVisible,
  expectModalHidden,
  editorClear,
  editorPaste,
  addDemoCharger,
  newLoadpoint,
} from "./utils";
import { startSimulator, stopSimulator, simulatorUrl, simulatorApply } from "./simulator";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

const CONFIG = "fast.evcc.yaml";

test.describe("HEMS", () => {
  test("grid sessions", async ({ page }) => {
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

    await expect(hemsModal.getByTestId("yaml-editor")).toBeVisible();
    await expect(hemsModal.getByRole("button", { name: "Save" })).toBeVisible();
    await expect(hemsModal).not.toContainText("Configured via evcc.yaml");

    await hemsModal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(hemsModal);
  });

  test("modal yaml-configured", async ({ page }) => {
    await start("hems-yaml.evcc.yaml");
    await page.goto("/#/config");

    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    const hemsModal = page.getByTestId("hems-modal");
    await expectModalVisible(hemsModal);

    await expect(hemsModal.getByTestId("grid-sessions")).not.toBeVisible();
    await expect(hemsModal).toContainText("Configured via evcc.yaml");
    await expect(hemsModal.getByTestId("yaml-editor")).not.toBeVisible();
    await expect(hemsModal.getByRole("button", { name: "Save" })).not.toBeVisible();
    await expect(hemsModal.getByRole("button", { name: "Cancel" })).toBeVisible();

    await hemsModal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(hemsModal);
  });

  test("configure loadmanagement and hems via UI, verify logic", async ({ page }) => {
    await startSimulator();
    await start(CONFIG);

    await page.goto("/#/config");

    // configure circuits
    await page.getByTestId("circuits").getByRole("button", { name: "edit" }).click();
    const circuitsModal = page.getByTestId("circuits-modal");
    await expectModalVisible(circuitsModal);
    const circuitsEditor = circuitsModal.getByTestId("yaml-editor");
    await editorClear(circuitsEditor);
    await editorPaste(circuitsEditor, page, `- name: gridcontrol`);
    await circuitsModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(circuitsModal);

    // configure hems
    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    const hemsModal = page.getByTestId("hems-modal");
    await expectModalVisible(hemsModal);
    const hemsEditor = hemsModal.getByTestId("yaml-editor");
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
    await hemsModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(hemsModal);

    await restart(CONFIG);
    await page.goto("/#/config");

    // configure loadpoint
    const lpModal = page.getByTestId("loadpoint-modal");
    await newLoadpoint(page, "Carport");
    await addDemoCharger(page);
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);
    await expect(page.getByTestId("loadpoint")).toContainText("Carport");

    await restart(CONFIG);
    await page.goto("/#/config");

    // assign loadpoint to circuit
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByLabel("Circuit").selectOption("gridcontrol");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    await restart(CONFIG);
    await page.goto("/#/config");

    // verify circuits
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
      ["External Limit", "Consumption limited", "yes", "Power", "0.0 kW / 4.2 kW"].join(""),
      { timeout: 10000 }
    );

    // verify main ui
    await page.getByTestId("home-link").click();
    const hemsWarning = page.getByTestId("hems-warning");
    await expect(hemsWarning).toContainText(
      "External limit: Reduced charging to not exceed 4.2 kW."
    );

    // disable in simulator
    await page.goto(simulatorUrl());
    await hemsRelayCheckbox.uncheck();
    await simulatorApply(page);

    // verify main ui
    await page.goto("/");
    await expect(hemsWarning).not.toBeVisible();

    await stopSimulator();
  });
});
