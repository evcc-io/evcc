import { test, expect, type WebSocketRoute } from "@playwright/test";
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

test("stay offline until api is ready", async ({ page }) => {
  let client: WebSocketRoute | undefined;
  await page.routeWebSocket("/ws", (ws) => {
    client = ws;
    const server = ws.connectToServer();
    ws.onMessage((message) => server.send(message));
    server.onMessage((message) => ws.send(message));
  });
  const send = (msg: object) => client?.send(JSON.stringify(msg));
  const offline = page.getByTestId("offline-indicator");

  await page.goto("/#/config");
  // initial state received
  await expect(page.getByTestId("generalconfig-network")).toContainText(new URL(baseUrl()).port);
  await expect(offline).toHaveAttribute("aria-hidden", "true");

  // server restarted, devices initializing
  send({ startupCompleted: false, apiReady: false });
  await expect(offline).toHaveAttribute("aria-hidden", "false");

  // devices initialized, http handlers not registered yet
  send({ startupCompleted: true });
  // later message as marker that the previous one has been processed
  send({ network: { port: 4242 } });
  await expect(page.getByTestId("generalconfig-network")).toContainText("4242");
  await expect(offline).toHaveAttribute("aria-hidden", "false");

  send({ apiReady: true });
  await expect(offline).toHaveAttribute("aria-hidden", "true");
});
