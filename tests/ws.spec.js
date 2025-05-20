import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("basics.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

test("show loadpoint with connect websocket", async ({ page }) => {
  await page.routeWebSocket("/ws", (ws) => {
    const server = ws.connectToServer();
    ws.onMessage((message) => {
      server.send(message);
    });
    server.onMessage((message) => {
      ws.send(message);
    });
  });
  await page.goto("/");
  await expect(page.getByRole("link", { name: "Let's start configuration" })).toBeHidden();
  await expect(page.getByText("Not connected to a server.")).toBeHidden();
  await expect(page.getByTestId("loadpoint")).toBeVisible();
});

test("show no config screen while startup", async ({ page }) => {
  await page.routeWebSocket("/ws", () => {
    // connect, but don't send any messages
  });
  await page.goto("/");
  await expect(page.getByRole("link", { name: "Let's start configuration" })).toBeHidden();
});

test("show offline when websocket is closed", async ({ page }) => {
  await page.routeWebSocket("/ws", (ws) => {
    ws.close();
  });
  await page.goto("/");
  await expect(page.getByText("Not connected to a server.")).toBeVisible();
  await expect(page.getByRole("link", { name: "Let's start configuration" })).toBeHidden();
});
