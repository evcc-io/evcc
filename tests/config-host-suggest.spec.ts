import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible, enableExperimental } from "./utils";

test.use({ baseURL: baseUrl() });

const HOSTS = [
  { ip: "192.0.2.20", mac: "00:15:BB:12:34:56", hostname: "sma3009876543" },
  { ip: "192.0.2.42", mac: "A8:03:2A:B1:23:45", hostname: "shellyplusplug-a8032ab" },
  { ip: "192.0.2.50", mac: "E8:0A:B9:12:34:56" },
];

test.beforeAll(async () => {
  process.env["EVCC_DISCOVERY_HOSTS"] = JSON.stringify(HOSTS);
  await start();
});

test.afterAll(async () => {
  delete process.env["EVCC_DISCOVERY_HOSTS"];
  await stop();
});

test.describe("host suggestions", async () => {
  test("template match", async ({ page }) => {
    await page.goto("/#/config");
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const modal = page.getByTestId("meter-modal");
    await expectModalVisible(modal);
    await modal.getByLabel("Manufacturer").selectOption("Shelly Pro 3EM");

    // single match is applied
    const host = modal.getByRole("combobox", { name: "IP address or hostname" });
    await expect(host).toHaveValue("192.0.2.42");

    await host.click();
    const matching = modal.getByRole("group", { name: "Matching" }).getByRole("option");
    await expect(matching).toHaveCount(1);
    await expect(matching).toContainText("192.0.2.42");
    await expect(matching).toContainText("shellyplusplug-a8032ab");
    await expect(matching).toContainText("Shelly");

    const other = modal.getByRole("group", { name: "Other" }).getByRole("option");
    await expect(other).toContainText(["192.0.2.20", "192.0.2.50"]);
    await expect(other.first()).toContainText("SMA");
    await expect(other.last()).toContainText("Cisco Systems, Inc");
  });

  test("modbus", async ({ page }) => {
    await page.goto("/#/config");
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const modal = page.getByTestId("meter-modal");
    await expectModalVisible(modal);
    await modal.getByLabel("Manufacturer").selectOption("SMA Data Manager");

    const host = modal.getByRole("combobox", { name: "IP address or hostname" });
    await host.click();
    const matching = modal.getByRole("group", { name: "Matching" }).getByRole("option");
    await expect(matching).toContainText(["192.0.2.20"]);

    await matching.click();
    await expect(host).toHaveValue("192.0.2.20");
  });

  test("discovery data", async ({ page }) => {
    await page.goto("/#/config");
    await expect(page.getByRole("button", { name: "Network discovery" })).toHaveCount(0);

    await enableExperimental(page);

    await page.getByRole("button", { name: "Network discovery" }).click();
    const modal = page.getByTestId("discovery-modal");
    await expectModalVisible(modal);
    const configured = modal.getByRole("textbox", { name: "Configured devices" });
    expect(await configured.inputValue()).toContain('"devices": []');

    await modal.getByRole("tab", { name: "All devices" }).click();
    const all = modal.getByRole("textbox", { name: "All devices" });
    const value = await all.inputValue();
    expect(value).toContain('"ip":"192.0.2.20"');
    expect(value).toContain('"mac":"00:15:BB:12:34:56"');
    expect(value).toContain('"vendor":"SMA Solar Technology AG"');
    expect(value).toContain('"hostname":"shellyplusplug-a8032ab"');
  });
});
