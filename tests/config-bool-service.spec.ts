import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible, getDatalistOptions } from "./utils";

test.use({ baseURL: baseUrl() });

const templateFlags = [
  "--disable-auth",
  "--template-type",
  "meter",
  "--template",
  "tests/config-bool-service-demo.tpl.yaml",
];

test.beforeAll(async () => {
  await start(undefined, undefined, templateFlags);
});

test.afterAll(async () => {
  await stop();
});

test.describe("config bool service param", async () => {
  test("unset bool defaults to false and fills service placeholder", async ({ page }) => {
    // service call fires with insecure=false even though the bool was never touched
    const servicePromise = page.waitForRequest(
      /\/config\/service\/demo\/insecure\?insecure=false$/
    );

    await page.goto("/#/config");
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Bool Service Meter");

    await servicePromise;

    // entity datalist is populated from the echoed flag
    const entity = meterModal.getByLabel("Entity");
    expect(await getDatalistOptions(entity)).toEqual(["insecure=false"]);
  });
});
