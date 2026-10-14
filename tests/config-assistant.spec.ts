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
    // clicking fails if the tab bar overlaps the button, they share a z-index
    await open.click();
    const panel = page.getByTestId("assistant-panel");
    await expect(panel).toBeVisible();

    // panel must clear the bottom tab bar, both sit at the same z-index
    const clearsTabBar = await page.evaluate(() => {
      const panel = document.querySelector('[data-testid="assistant-panel"]');
      const nav = document.querySelector('[data-testid="bottom-tab-bar"]');
      if (!panel || !nav) return false;
      return panel.getBoundingClientRect().bottom <= nav.getBoundingClientRect().top;
    });
    expect(clearsTabBar).toBe(true);

    // question reaches the backend, which reports the unreachable model
    await page.getByTestId("assistant-input").fill("what is wrong?");
    await page.getByRole("button", { name: "Send" }).click();
    await expect(page.getByTestId("assistant-panel")).toContainText("what is wrong?");
    await expect(page.getByTestId("assistant-panel")).toContainText(/connection refused/i);

    // cursor up/down browses the asked questions
    const input = page.getByTestId("assistant-input");
    await input.fill("and now?");
    await input.press("Enter");
    await expect(page.getByTestId("assistant-panel")).toContainText("and now?");

    await input.fill("draft");
    await input.press("ArrowUp");
    await expect(input).toHaveValue("and now?");
    await input.press("ArrowUp");
    await expect(input).toHaveValue("what is wrong?");
    // oldest entry is the end of the line
    await input.press("ArrowUp");
    await expect(input).toHaveValue("what is wrong?");
    await input.press("ArrowDown");
    await expect(input).toHaveValue("and now?");
    // back past the newest entry restores the draft
    await input.press("ArrowDown");
    await expect(input).toHaveValue("draft");
    await input.fill("");

    // escape aborts the running question and hands it back for editing
    const messages = page.getByTestId("assistant-panel").locator(".message");
    await expect(messages).toHaveCount(2);
    await page.route("**/api/assistant/chat", () => {
      /* never answers */
    });
    await input.fill("takes forever");
    await input.press("Enter");
    await expect(page.getByTestId("assistant-stop")).toBeVisible();
    await input.press("Escape");
    await expect(page.getByTestId("assistant-stop")).toHaveCount(0);
    await expect(input).toHaveValue("takes forever");
    await expect(messages).toHaveCount(2);
    await page.unroute("**/api/assistant/chat");
    await input.fill("");

    // available on every page, the conversation survives navigation
    await page.goto("/#/log");
    await expect(page.getByTestId("assistant-panel")).toContainText("what is wrong?");
    await page.goto("/#/");
    await expect(page.getByTestId("assistant-panel")).toContainText("what is wrong?");
  });
});
