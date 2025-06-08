import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { enableExperimental, expectModalHidden, expectModalVisible } from "./utils";
import { isMqttReachable } from "./mqtt";

const CONFIG = "config-grid-only.evcc.yaml";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.beforeEach(async ({ page }) => {
  await start(CONFIG);
  await page.goto("/#/config");
  await enableExperimental(page);
});

test.afterEach(async () => {
  await stop();
});

const VALID_BROKER = "test.mosquitto.org:1884";
const INVALID_BROKER = "unknown.example.org";
const VALID_TOPIC = "my-topic";
const VALID_CLIENT_ID = "my-client-id";
const VALID_USERNAME = "rw";
const VALID_PASSWORD = "readwrite";

test.describe("mqtt", async () => {
  test.skip(
    async () => !(await isMqttReachable(VALID_BROKER, VALID_USERNAME, VALID_PASSWORD)),
    `MQTT broker ${VALID_BROKER} is not reachable, skipping tests`
  );

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

    await restart(CONFIG);

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
    await restart(CONFIG);

    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
    await expect(page.getByTestId("mqtt")).not.toHaveClass(/round-box--error/);
    await expect(page.getByTestId("mqtt")).toContainText(
      ["Broker", VALID_BROKER, "Topic", VALID_TOPIC].join("")
    );
  });
});
