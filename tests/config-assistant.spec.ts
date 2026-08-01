import { test, expect, type Page } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden } from "./utils";
import axios from "axios";

const CONFIG = "config-grid-only.evcc.yaml";
const CHAT_URL = `${baseUrl()}/api/assistant/chat`;

test.use({ baseURL: baseUrl() });

test.beforeEach(async () => {
  await start(CONFIG);
});

test.afterEach(async () => {
  await stop();
});

async function enableExperimental(page: Page) {
  await page.goto("/#/config");
  const entry = page.getByTestId("generalconfig-experimental");
  await entry.getByRole("button", { name: "edit" }).click();
  const modal = page.getByTestId("experimental-modal");
  await expectModalVisible(modal);
  await modal.getByLabel("Enable experimental features.").click();
  await modal.getByRole("button", { name: "Close" }).click();
  await expectModalHidden(modal);
}

test.describe("assistant", () => {
  test("chat rejected while unconfigured", async () => {
    const response = await axios.post(
      CHAT_URL,
      { messages: [{ role: "user", content: "hi" }] },
      { validateStatus: () => true }
    );
    expect(response.status).toBe(412);
  });

  test("configure and use the helper", async ({ page }) => {
    await enableExperimental(page);

    // no helper without configuration
    await expect(page.getByTestId("assistant-open")).toHaveCount(0);

    const card = page.getByTestId("assistant");
    await expect(card).toBeVisible();
    await card.getByRole("button", { name: "edit" }).click();

    const modal = page.getByTestId("assistant-modal");
    await expectModalVisible(modal);

    // model follows the provider, unless it was typed by hand
    const model = modal.getByLabel("Model");
    await expect(model).toHaveValue("gpt-5");
    await modal.getByLabel("Provider").selectOption("Anthropic");
    await expect(model).toHaveValue("claude-opus-5");
    await model.fill("my-own-model");
    await modal.getByLabel("Provider").selectOption("OpenAI");
    await expect(model).toHaveValue("my-own-model");

    // ollama needs no api key
    await modal.getByLabel("Provider").selectOption("Ollama");
    await model.fill("qwen3");
    await modal.getByLabel("API endpoint").fill("http://127.0.0.1:1");
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    // floating helper appears on the config page
    const open = page.getByTestId("assistant-open");
    await expect(open).toBeVisible();
    await open.click();
    await expect(page.getByTestId("assistant-panel")).toBeVisible();

    // question reaches the backend, which reports the unreachable model
    await page.getByTestId("assistant-input").fill("what is wrong?");
    await page.getByRole("button", { name: "Send" }).click();
    await expect(page.getByTestId("assistant-panel")).toContainText("what is wrong?");
    await expect(page.getByTestId("assistant-panel")).toContainText(/connection refused/i);

    // and on the log page
    await page.goto("/#/log");
    await expect(page.getByTestId("assistant-open")).toBeVisible();
  });
});
