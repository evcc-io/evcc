import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { enableExperimental } from "./utils";

const CONFIG_YAML = "config-circuit.yaml";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

test.describe("circuit", async () => {
  test("from yaml", async ({ page }) => {
    await start(CONFIG_YAML);

    await page.goto("/#/config");
    await enableExperimental(page);

    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    await expect(page.getByTestId("loadpoint")).toContainText(["Power", "1.0 kW"].join(""));

    await expect(page.getByTestId("grid")).toHaveCount(1);
    await expect(page.getByTestId("grid")).toContainText(["Power", "2.1 kW"].join(""));
    await expect(page.getByTestId("grid")).toContainText(
      ["Current L1, L2, L3", "3.0 · 3.0 · 3.0 A"].join("")
    );

    await expect(page.getByTestId("circuits")).toHaveCount(1);
    await expect(page.getByTestId("circuits")).toContainText(["Power", "2.1 kW"].join(""));
    await expect(page.getByTestId("circuits")).toContainText(
      ["Current", "3.0 A / 16.0 A"].join("")
    );
  });
});
