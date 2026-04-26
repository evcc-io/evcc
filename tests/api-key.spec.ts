import { test, expect, type Page, type Locator } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

const BASIC = "basics.evcc.yaml";
const PASSWORD = "secret";

async function loginAndOpenApiKey(page: Page): Promise<Locator> {
  await page.goto("/#/config");

  const loginModal = page.getByTestId("login-modal");
  await expectModalVisible(loginModal);
  await loginModal.getByLabel("Administrator Password").fill(PASSWORD);
  await loginModal.getByRole("button", { name: "Login" }).click();
  await expectModalHidden(loginModal);

  return openApiKeyModal(page);
}

async function openApiKeyModal(page: Page): Promise<Locator> {
  await page.getByTestId("generalconfig-security").getByRole("button", { name: "edit" }).click();
  const securityModal = page.getByTestId("security-modal");
  await expectModalVisible(securityModal);
  await securityModal.getByRole("button", { name: /^(?:Generate|Regenerate) API Key$/ }).click();

  const apiKeyModal = page.getByTestId("api-key-modal");
  await expectModalVisible(apiKeyModal);
  return apiKeyModal;
}

async function generateKey(
  page: Page,
  modal: Locator,
  action: "Generate API Key" | "Regenerate API Key"
): Promise<string> {
  if (action === "Regenerate API Key") {
    page.once("dialog", (dialog) => dialog.accept());
  }
  await modal.getByRole("button", { name: action, exact: true }).click();
  await modal.getByLabel("Administrator Password").fill(PASSWORD);
  await modal.getByRole("button", { name: action, exact: true }).click();

  const keyInput = modal.getByLabel("API Key", { exact: true });
  await expect(keyInput).toBeVisible();
  const key = await keyInput.inputValue();
  expect(key).toMatch(/^evcc_/);
  expect(key.length).toBeGreaterThan(10);
  return key;
}

test("generate first key", async ({ page }) => {
  await start(BASIC, "password.sql", "");
  const modal = await loginAndOpenApiKey(page);

  await expect(modal.getByRole("button", { name: "Generate API Key" })).toBeVisible();

  const key = await generateKey(page, modal, "Generate API Key");
  expect(key).toMatch(/^evcc_/);

  await modal.getByRole("button", { name: "Done" }).click();
  await expectModalHidden(modal);

  // reopen and verify the regenerate path is now offered
  const reopened = await openApiKeyModal(page);
  await expect(reopened.getByRole("button", { name: "Regenerate API Key" })).toBeVisible();

  await stop();
});

test("regenerate replaces old key", async ({ page, request }) => {
  await start(BASIC, "password.sql", "");
  const modal = await loginAndOpenApiKey(page);

  const first = await generateKey(page, modal, "Generate API Key");
  await modal.getByRole("button", { name: "Done" }).click();
  await expectModalHidden(modal);

  const reopened = await openApiKeyModal(page);
  const second = await generateKey(page, reopened, "Regenerate API Key");
  await reopened.getByRole("button", { name: "Done" }).click();
  expect(second).not.toBe(first);

  const oldRes = await request.get("/api/config/site", {
    headers: { Authorization: `Bearer ${first}` },
  });
  expect(oldRes.status()).toBe(401);

  const newRes = await request.get("/api/config/site", {
    headers: { Authorization: `Bearer ${second}` },
  });
  expect(newRes.status()).toBe(200);

  await stop();
});

test("api key authenticates protected endpoints and bypasses backup pw", async ({
  page,
  request,
}) => {
  await start(BASIC, "password.sql", "");
  const modal = await loginAndOpenApiKey(page);
  const key = await generateKey(page, modal, "Generate API Key");

  const ok = await request.get("/api/config/site", {
    headers: { Authorization: `Bearer ${key}` },
  });
  expect(ok.status()).toBe(200);

  // backup with empty body, bypass via API key
  const backup = await request.post("/api/system/backup", {
    headers: { Authorization: `Bearer ${key}`, "Content-Type": "application/json" },
    data: {},
  });
  expect(backup.status()).toBe(200);
  expect(backup.headers()["content-disposition"] || "").toContain("evcc-backup-");

  const unauth = await request.get("/api/config/site");
  expect(unauth.status()).toBe(401);

  await stop();
});

test("api key cannot rotate itself without admin password", async ({ page, request }) => {
  await start(BASIC, "password.sql", "");
  const modal = await loginAndOpenApiKey(page);
  const key = await generateKey(page, modal, "Generate API Key");

  const bad = await request.post("/api/auth/apikey", {
    headers: { Authorization: `Bearer ${key}`, "Content-Type": "application/json" },
    data: { password: "" },
  });
  expect(bad.status()).toBe(401);

  const good = await request.post("/api/auth/apikey", {
    headers: { Authorization: `Bearer ${key}`, "Content-Type": "application/json" },
    data: { password: PASSWORD },
  });
  expect(good.status()).toBe(200);
  const body = await good.json();
  expect(body.key).toMatch(/^evcc_/);
  expect(body.key).not.toBe(key);

  await stop();
});

test("api key cannot change admin password without correct current", async ({ page, request }) => {
  await start(BASIC, "password.sql", "");
  const modal = await loginAndOpenApiKey(page);
  const key = await generateKey(page, modal, "Generate API Key");

  const bad = await request.put("/api/auth/password", {
    headers: { Authorization: `Bearer ${key}`, "Content-Type": "application/json" },
    data: { current: "", new: "anything" },
  });
  expect(bad.status()).toBe(400);

  await stop();
});

test("disable-auth shows banner and disables actions", async ({ page }) => {
  await start(BASIC, null, "--disable-auth");
  await page.goto("/#/config");

  await page.getByTestId("generalconfig-security").getByRole("button", { name: "edit" }).click();
  const security = page.getByTestId("security-modal");
  await expectModalVisible(security);
  await expect(security.getByText(/Authentication is disabled/i)).toBeVisible();
  await expect(security.getByRole("button", { name: "Update password" })).toBeDisabled();
  await expect(security.getByRole("button", { name: "Generate API Key" })).toBeDisabled();

  await stop();
});
