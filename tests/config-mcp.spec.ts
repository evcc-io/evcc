import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden } from "./utils";
import axios from "axios";

const CONFIG = "config-grid-only.evcc.yaml";
const MCP_URL = `${baseUrl()}/mcp`;

test.use({ baseURL: baseUrl() });

test.beforeEach(async () => {
  await start(CONFIG);
});

test.afterEach(async () => {
  await stop();
});

const initRequest = {
  jsonrpc: "2.0",
  id: 1,
  method: "initialize",
  params: {
    protocolVersion: "2024-11-05",
    capabilities: {},
    clientInfo: { name: "playwright", version: "1" },
  },
};

const initHeaders = {
  "Content-Type": "application/json",
  Accept: "application/json, text/event-stream",
};

test.describe("mcp", () => {
  test("/mcp not available without experimental", async () => {
    const response = await axios.post(MCP_URL, initRequest, {
      headers: initHeaders,
      validateStatus: () => true,
    });
    expect(response.status).toBe(404);
  });

  test("enable experimental, restart, verify endpoint", async ({ page }) => {
    await page.goto("/#/config");

    // MCP card not shown yet
    await expect(page.getByTestId("mcp")).toHaveCount(0);

    // enable experimental
    const experimentalEntry = page.getByTestId("generalconfig-experimental");
    await experimentalEntry.getByRole("button", { name: "edit" }).click();
    const experimentalModal = page.getByTestId("experimental-modal");
    await expectModalVisible(experimentalModal);
    await experimentalModal.getByLabel("Enable experimental features.").click();
    await experimentalModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(experimentalModal);

    // MCP card now visible (Services section, gated on experimental)
    const mcpCard = page.getByTestId("mcp");
    await expect(mcpCard).toBeVisible();

    // open MCP modal — pre-restart, warning is shown and endpoint is announced
    await mcpCard.getByRole("button", { name: "edit" }).click();
    const mcpModal = page.getByTestId("mcp-modal");
    await expectModalVisible(mcpModal);
    await expect(mcpModal).toContainText("Will be available after restart.");
    const mcpUrl = await mcpModal.getByLabel("MCP endpoint").inputValue();
    expect(mcpUrl).toBe(MCP_URL);
    await mcpModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(mcpModal);

    // restart and re-open modal — warning should be gone
    await restart(CONFIG);
    await page.reload();
    await page.getByTestId("mcp").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(mcpModal);
    await expect(mcpModal).not.toContainText("Will be available after restart.");

    // hit the endpoint at the URL the modal advertises
    const response = await axios.post(mcpUrl, initRequest, {
      headers: initHeaders,
      validateStatus: () => true,
    });
    expect(response.status).toBe(200);
  });
});
