import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.beforeEach(async () => {
  await start("basics.evcc.yaml");
});
test.afterEach(async () => {
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

test("force navigation after websocket open timeout", async ({ page }) => {
  // Replace WebSocket with a mock that never fires onopen,
  // simulating Safari's bug where the upgrade request is silently dropped.
  await page.addInitScript(() => {
    (window as any).WebSocket = class {
      readyState = 0;
      onopen: (() => void) | null = null;
      onclose: ((ev: CloseEvent) => void) | null = null;
      onerror: (() => void) | null = null;
      onmessage: (() => void) | null = null;
      close() {
        this.readyState = 3;
        this.onclose?.(new CloseEvent("close"));
      }
      send() {}
    };
  });
  await page.goto("/");
  // after the 5s open timeout the app navigates to strip the hash fragment
  await page.waitForURL(/wsRetry/, { timeout: 10000 });
});
