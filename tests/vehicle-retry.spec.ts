import { createServer, type Server } from "node:net";
import { test, expect } from "@playwright/test";
import { Aedes } from "aedes";
import { start, stop, baseUrl } from "./evcc";
import { simulatorConfig, simulatorUrl } from "./simulator";

test.use({ baseURL: baseUrl() });

let broker: Aedes;
let server: Server;

test.beforeEach(async () => {
  broker = await Aedes.createBroker();
  server = createServer(broker.handle);
  process.env["EVCC_VEHICLE_RETRY_INTERVAL"] = "1s";
  await start(simulatorConfig("./tests/vehicle-retry.evcc.yaml"));
});

test.afterEach(async () => {
  delete process.env["EVCC_VEHICLE_RETRY_INTERVAL"];
  await stop();
  await new Promise<void>((resolve) => broker.close(() => server.close(() => resolve())));
});

test("vehicle retry", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByTestId("vehicle-not-reachable-icon")).toBeVisible();

  await new Promise<void>((resolve) => server.listen(new URL(simulatorUrl()).port, resolve));
  broker.publish(
    { cmd: "publish", topic: "car/soc", payload: "42", qos: 0, retain: true, dup: false },
    () => {}
  );

  await expect(page.getByTestId("vehicle-not-reachable-icon")).not.toBeVisible();
  await expect(page.getByTestId("current-soc")).toContainText("42%");
});
