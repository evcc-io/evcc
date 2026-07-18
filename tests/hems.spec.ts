import { test, expect } from "@playwright/test";
import axios from "axios";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  expectModalVisible,
  expectModalHidden,
  editorClear,
  editorPaste,
  enableAppContext,
  expectAppEvent,
  newLoadpoint,
  addDemoCharger,
} from "./utils";
import { startSimulator, stopSimulator, simulatorUrl, simulatorApply } from "./simulator";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

const CONFIG = "fast.evcc.yaml";

test.describe("HEMS", () => {
  test("no recorded events section in create mode", async ({ page }) => {
    await start(CONFIG, "hems.sql");
    await page.goto("/#/config");

    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    const hemsModal = page.getByTestId("hems-modal");
    await expectModalVisible(hemsModal);

    await expect(hemsModal.getByTestId("grid-sessions")).not.toBeVisible();

    // template selector is shown in create mode, yaml editor hidden until "User-defined" is chosen
    await expect(hemsModal.getByLabel("Integration")).toBeVisible();
    await expect(hemsModal.getByTestId("yaml-editor")).not.toBeVisible();
    await expect(hemsModal).not.toContainText("Configured via evcc.yaml");

    await hemsModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(hemsModal);
  });

  test("recorded events in configured modal", async ({ page }) => {
    await start("hems-yaml.evcc.yaml", "hems.sql");
    await page.goto("/#/config");

    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    const hemsModal = page.getByTestId("hems-modal");
    await expectModalVisible(hemsModal);

    await expect(hemsModal.getByTestId("grid-sessions")).toBeVisible();
    await expect(hemsModal).toContainText("Recorded events");
    await expect(hemsModal).toContainText("Recorded 3 grid limitation events");
    await expect(hemsModal).toContainText("Most recent");

    await hemsModal.getByRole("button", { name: "Download" }).click();
    await expect(hemsModal.getByRole("link", { name: "CSV" })).toHaveAttribute(
      "href",
      "./api/gridsessions?lang=en&format=csv"
    );
    await expect(hemsModal.getByRole("link", { name: "XLSX" })).toHaveAttribute(
      "href",
      "./api/gridsessions?lang=en&format=xlsx"
    );

    await hemsModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(hemsModal);
  });

  test("modal yaml-configured is locked", async ({ page }) => {
    await start("hems-yaml.evcc.yaml");
    await page.goto("/#/config");

    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    const hemsModal = page.getByTestId("hems-modal");
    await expectModalVisible(hemsModal);

    await expect(hemsModal.getByTestId("grid-sessions")).toBeVisible();
    await expect(hemsModal).toContainText("No events recorded yet");
    await expect(hemsModal).toContainText("Configured via evcc.yaml");
    await expect(hemsModal.getByLabel("Integration")).not.toBeVisible();
    await expect(hemsModal.getByTestId("yaml-editor")).not.toBeVisible();
    await expect(hemsModal.getByRole("button", { name: "Save" })).not.toBeVisible();

    await hemsModal.getByRole("button", { name: "Close" }).click();
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
    await hemsModal.getByLabel("Integration").selectOption({ label: "User-defined integration" });
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

    // no circuits configured, no synthesized external circuit
    await expect(page.getByTestId("circuits")).toContainText(["Configured", "no"].join(""));
    await expect(page.getByTestId("circuits").getByTestId("device-banner")).not.toBeVisible();

    // enable hems in simulator
    await page.goto(simulatorUrl());
    const hemsRelayCheckbox = page.getByLabel("Relay (dim)");
    await hemsRelayCheckbox.check();
    await simulatorApply(page);

    // verify config ui: limit shown on hems card, no hint without circuits
    await page.goto("/#/config");
    await expect(page.getByTestId("hems")).toContainText(["Consumption limit", "4.2 kW"].join(""));
    await expect(page.getByTestId("circuits").getByTestId("device-banner")).not.toBeVisible();

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
    await hemsModal.getByLabel("Integration").selectOption({ label: "User-defined integration" });
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

    // card shows user-defined label
    await expect(page.getByTestId("hems")).toContainText("User-defined integration");

    // reopen modal in edit mode
    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(hemsModal);
    await expect(hemsEditor).toBeVisible();
    await expect(hemsModal.getByLabel("Integration")).toBeDisabled();

    // click pen (change) and confirm
    page.once("dialog", (d) => d.accept());
    await hemsModal.getByRole("button", { name: "Change" }).click();

    // provider selector reappears, yaml editor gone
    await expect(hemsModal.getByLabel("Integration")).toBeEnabled();
    await expect(hemsEditor).not.toBeVisible();
    await expect(hemsModal.getByRole("button", { name: "Change" })).not.toBeVisible();

    // close modal, card shows unconfigured
    await hemsModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(hemsModal);
    await expect(page.getByTestId("hems")).toContainText(["Configured", "no"].join(""));
  });

  test("fnn signals", async ({ page }) => {
    const GRID_CONFIG = "hems-grid.evcc.yaml";
    await startSimulator();
    await start(GRID_CONFIG);

    await page.goto("/#/config");

    // configure circuits, hint on the circuits card requires them
    await page.getByTestId("circuits").getByRole("button", { name: "edit" }).click();
    const circuitsModal = page.getByTestId("circuits-modal");
    await expectModalVisible(circuitsModal);
    const circuitsEditor = circuitsModal.getByTestId("yaml-editor");
    await editorClear(circuitsEditor);
    await editorPaste(circuitsEditor, page, `- name: main\n  title: House`);
    await circuitsModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(circuitsModal);

    // configure fnn hems with all signals wired to the simulator
    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    const hemsModal = page.getByTestId("hems-modal");
    await expectModalVisible(hemsModal);
    await hemsModal.getByLabel("Integration").selectOption({ label: "User-defined integration" });
    const hemsEditor = hemsModal.getByTestId("yaml-editor");
    await expect(hemsEditor).toBeVisible();
    await editorClear(hemsEditor);
    const signalSource = (signal: string) => `
  source: http
  uri: ${simulatorUrl()}/api/state
  jq: .hems.${signal}`;
    await editorPaste(
      hemsEditor,
      page,
      `type: fnn
maxDimPower: 4200
maxCurtailPower: 10000
interval: 0.1s
w3:${signalSource("w3")}
s1:${signalSource("s1")}
s2:${signalSource("s2")}
w4:${signalSource("w4")}`
    );
    await hemsModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(hemsModal);

    await restart(GRID_CONFIG);

    // drive signals via simulator api, switching between both uis is too slow
    const setSignal = async (signal: string, active: boolean) => {
      const { data: state } = await axios.get(`${simulatorUrl()}/api/state`);
      state.hems[signal] = active;
      await axios.post(`${simulatorUrl()}/api/state`, state);
    };
    const hems = page.getByTestId("hems");
    const hint = page.getByTestId("circuits").getByTestId("device-banner");
    const pvBanner = page.getByTestId("pv").getByTestId("device-banner");

    // no signals: no limits
    await page.goto("/#/config");
    await expect(hems).toContainText(
      ["Consumption limited", "no", "Feed-in limited", "no"].join("")
    );
    await expect(hint).not.toBeVisible();

    // dim (W4): consumption limited
    await setSignal("w4", true);
    await expect(hems).toContainText(["Consumption limit", "4.2 kW"].join(""));
    await expect(hint).toHaveText("Consumption limited");
    await expect(pvBanner).not.toBeVisible();

    // main screen shows the warning
    await page.goto("/#/");
    await expect(page.getByTestId("hems-warning")).toContainText("4.2 kW");
    await page.goto("/#/config");

    // full curtailment (W3): feed-in limited to 0
    await setSignal("w4", false);
    await setSignal("w3", true);
    await expect(hems).toContainText(["Feed-in limit", "0.0 kW"].join(""));
    await expect(hint).not.toBeVisible();
    // device status is polled, reload for a fresh value
    await page.reload();
    await expect(pvBanner).toHaveText("Production limited");

    // partial curtailment (S1): 60% of 10 kW
    await setSignal("w3", false);
    await setSignal("s1", true);
    await expect(hems).toContainText(["Feed-in limit", "6.0 kW"].join(""));

    // main screen shows the production warning
    await page.goto("/#/");
    await expect(page.getByTestId("hems-warning")).toContainText(["Feed-in", "≤ 6.0 kW"].join(""));
    await expect(page.getByTestId("hems-warning")).not.toContainText("Consumption");
    await page.goto("/#/config");

    // partial curtailment (S2): 30% of 10 kW
    await setSignal("s1", false);
    await setSignal("s2", true);
    await expect(hems).toContainText(["Feed-in limit", "3.0 kW"].join(""));

    // all clear: banners disappear
    await setSignal("s2", false);
    await expect(hint).not.toBeVisible();
    await page.reload();
    await expect(pvBanner).not.toBeVisible();
    await expect(hems).toContainText(
      ["Consumption limited", "no", "Feed-in limited", "no"].join("")
    );

    await stopSimulator();
  });

  test("curtailment without circuits", async ({ page }) => {
    const GRID_CONFIG = "hems-grid.evcc.yaml";
    await start(GRID_CONFIG);

    await page.goto("/#/config");

    // configure hems with constant curtailment, no circuits
    await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
    const hemsModal = page.getByTestId("hems-modal");
    await expectModalVisible(hemsModal);
    await hemsModal.getByLabel("Integration").selectOption({ label: "User-defined integration" });
    const hemsEditor = hemsModal.getByTestId("yaml-editor");
    await expect(hemsEditor).toBeVisible();
    await editorClear(hemsEditor);
    await editorPaste(
      hemsEditor,
      page,
      `type: fnn
maxCurtailPower: 10000
interval: 0.1s
w3:
  source: const
  value: true`
    );
    await hemsModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(hemsModal);

    await restart(GRID_CONFIG);
    await page.goto("/#/config");

    await expect(page.getByTestId("hems")).toContainText(["Feed-in limit", "0.0 kW"].join(""));
    await expect(page.getByTestId("pv").getByTestId("device-banner")).toHaveText(
      "Production limited"
    );
  });

  test.describe("grid sessions CSV in app context", () => {
    test("dispatches download event", async ({ page }) => {
      await enableAppContext(page);
      await start("hems-yaml.evcc.yaml", "hems.sql");
      await page.goto("/#/config");

      await page.getByTestId("hems").getByRole("button", { name: "edit" }).click();
      const hemsModal = page.getByTestId("hems-modal");
      await expectModalVisible(hemsModal);

      await hemsModal.getByRole("button", { name: "Download" }).click();
      await hemsModal.getByRole("link", { name: "CSV" }).click();
      expect(await expectAppEvent(page)).toMatchObject({
        type: "download",
        url: expect.stringContaining("/api/gridsessions?lang=en&format=csv"),
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
    await hemsModal.getByLabel("Integration").selectOption({ label: "User-defined integration" });
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

    // circuit tree shows only the user circuit, consumption limit as hint above it
    await expect(page.getByTestId("circuits").getByTestId("device-banner")).toHaveText(
      "Consumption limited"
    );
    await expect(page.getByTestId("circuits")).toContainText(["House", "Power", "0.0 kW"].join(""));
    await expect(page.getByTestId("circuits")).not.toContainText("External Limit");

    // a new loadpoint can only be assigned to the dedicated circuit
    await newLoadpoint(page, "Carport");
    await addDemoCharger(page);
    const lpModal = page.getByTestId("loadpoint-modal");
    await expectModalVisible(lpModal);
    await lpModal.getByRole("link", { name: "Advanced configuration" }).click();
    const circuitOptions = lpModal.getByLabel("Circuit").getByRole("option");
    await expect(circuitOptions).toHaveText(["---", "House [main]"]);
  });
});
