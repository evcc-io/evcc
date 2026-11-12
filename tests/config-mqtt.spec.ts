import { createServer, type Server } from "node:net";
import { test, expect } from "@playwright/test";
import { Aedes } from "aedes";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

let broker: Aedes;
let server: Server;

test.beforeEach(async ({ page }) => {
  // local broker, public test brokers drop connections and evcc treats that as a startup error
  broker = await Aedes.createBroker({
    authenticate: (_, username, password, done) =>
      done(null, username === VALID_USERNAME && password?.toString() === VALID_PASSWORD),
  });
  server = createServer(broker.handle);
  await new Promise<void>((resolve) => server.listen(BROKER_PORT, "127.0.0.1", resolve));
  await start();
  await page.goto("/#/config");
});

test.afterEach(async () => {
  await stop();
  await new Promise<void>((resolve) => broker.close(() => server.close(() => resolve())));
});

const BROKER_PORT = 14000 + Number(process.env["TEST_WORKER_INDEX"] ?? 0);
const VALID_BROKER = `127.0.0.1:${BROKER_PORT}`;
const INVALID_BROKER = "unknown.example.org";
const VALID_TOPIC = "my-topic";
const VALID_CLIENT_ID = "my-client-id";
const VALID_USERNAME = "rw";
const VALID_PASSWORD = "readwrite";

test.describe("mqtt", async () => {
  test("mqtt not configured", async ({ page }) => {
    await expect(page.getByTestId("mqtt")).toBeVisible();
    await expect(page.getByTestId("mqtt")).toContainText(["Configured", "no"].join(""));
  });

  test("mqtt via ui", async ({ page }) => {
    await page.getByTestId("mqtt").getByRole("button", { name: "edit" }).click();
    const modal = await page.getByTestId("mqtt-modal");
    await expectModalVisible(modal);

    // setup with invalid broker
    await modal.getByLabel("Broker").fill(INVALID_BROKER);
    await modal.getByLabel("Topic").fill("  " + VALID_TOPIC + " "); // whitespace should be trimmed
    await modal.getByLabel("Client ID").fill(VALID_CLIENT_ID);
    await modal.getByLabel("Username").fill(VALID_USERNAME);
    await modal.getByLabel("Password").fill(VALID_PASSWORD);
    await page.getByRole("button", { name: "Save" }).click();
    await expect(modal.getByTestId("error")).not.toBeVisible();
    await expectModalHidden(modal);

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart();

    // config error
    await expect(page.getByTestId("mqtt")).toHaveClass(/round-box--error/);
    await expect(page.getByTestId("mqtt")).toContainText(
      ["Broker", INVALID_BROKER, "Topic", VALID_TOPIC].join("")
    );
    await expect(page.getByTestId("fatal-error")).toContainText("failed configuring mqtt");

    await page.getByTestId("mqtt").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);
    await expect(modal.getByLabel("Broker")).toHaveValue(INVALID_BROKER);
    await expect(modal.getByLabel("Topic")).toHaveValue(VALID_TOPIC); // whitespace has been trimmed
    await expect(modal.getByLabel("Client ID")).toHaveValue(VALID_CLIENT_ID);
    await expect(modal.getByLabel("Username")).toHaveValue(VALID_USERNAME);
    await expect(modal.getByLabel("Password")).toHaveValue("***");

    // use valid broker
    await modal.getByLabel("Broker").fill(VALID_BROKER);
    await modal.getByRole("button", { name: "Save" }).click();
    await expect(page.getByTestId("mqtt")).toContainText(
      ["Broker", VALID_BROKER, "Topic", VALID_TOPIC].join("")
    );
    await restart();

    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
    await expect(page.getByTestId("mqtt")).not.toHaveClass(/round-box--error/);
    await expect(page.getByTestId("mqtt")).toContainText(
      ["Broker", VALID_BROKER, "Topic", VALID_TOPIC].join("")
    );
  });
});
