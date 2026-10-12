import { test, expect, type Locator, type Page } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible } from "./utils";
import { simulatorUrl, startSimulator, stopSimulator } from "./simulator";

test.use({ baseURL: baseUrl() });

const templateFlags = [
  "--disable-auth",
  "--template-type",
  "vehicle",
  "--template",
  "tests/config-provider-auth-challenge-demo.tpl.yaml",
];

test.beforeEach(async () => {
  await startSimulator();
  await start(undefined, undefined, templateFlags);
});
test.afterEach(async () => {
  await stop();
  await stopSimulator();
});

// open the vehicle modal at step 1 with server, method and email set, password left to the caller
async function openStepOne(page: Page, method: string): Promise<Locator> {
  await page.goto("/#/config");
  await page.getByTestId("add-vehicle").click();
  const vehicleModal = page.getByTestId("vehicle-modal");
  await expectModalVisible(vehicleModal);
  await vehicleModal.getByLabel("Manufacturer").selectOption("Auth Demo Vehicle");

  // step 1: auth fields only
  await expect(vehicleModal.getByLabel("Email")).toBeVisible();
  await expect(vehicleModal.getByLabel("Title")).not.toBeVisible();
  await expect(vehicleModal.getByRole("button", { name: "Validate & save" })).not.toBeVisible();

  await vehicleModal.getByLabel("Authentication Method").selectOption(method);
  await vehicleModal.getByLabel("Email").fill("driver@example.com");
  await vehicleModal.getByLabel("Server").fill(simulatorUrl());
  return vehicleModal;
}

async function saveVehicle(page: Page, vehicleModal: Locator) {
  await vehicleModal.getByLabel("Title").fill("Demo Car");
  await vehicleModal.getByRole("button", { name: "Validate & save" }).click();
  await expectModalHidden(vehicleModal);
  await expect(page.getByTestId("vehicle")).toContainText("Demo Car");

  const providers = page.getByTestId("auth-providers");
  await expect(providers).toContainText("Demo Auth");
  await expect(providers.getByRole("button", { name: "disconnect" })).toBeVisible();
}

test.describe("provider auth challenge", async () => {
  test("captcha", async ({ page }) => {
    const vehicleModal = await openStepOne(page, "challenge");

    // wrong password
    await vehicleModal.getByLabel("Password").fill("wrong");
    await vehicleModal.getByRole("button", { name: "Prepare connection" }).click();
    await expect(vehicleModal).toContainText("invalid credentials");
    await expect(vehicleModal.getByLabel("Email")).toBeVisible();

    // correct password leads to captcha, connect button gone
    await vehicleModal.getByLabel("Password").fill("topsecret");
    await vehicleModal.getByRole("button", { name: "Prepare connection" }).click();
    const captcha = vehicleModal.getByLabel("Enter the characters shown in the image.");
    await expect(captcha).toBeVisible();
    await expect(vehicleModal.getByRole("presentation")).toBeVisible();
    await expect(
      vehicleModal.getByRole("button", { name: "Prepare connection" })
    ).not.toBeVisible();

    // wrong captcha shows a fresh one
    await captcha.fill("0000");
    await vehicleModal.getByRole("button", { name: "Continue" }).click();
    await expect(captcha).toHaveValue("");
    await expect(vehicleModal.getByRole("presentation")).toBeVisible();

    // correct captcha completes login, step 2 shows the vehicle form
    await captcha.fill("1234");
    await vehicleModal.getByRole("button", { name: "Continue" }).click();
    await expect(vehicleModal.getByLabel("Title")).toBeVisible();
    await expect(captcha).not.toBeVisible();
    // login alone needs no restart, only the device save below does
    await expect(page.getByTestId("restart-needed")).not.toBeVisible();

    await saveVehicle(page, vehicleModal);

    // disconnect, reconnect from the card: stored credentials go straight to the captcha
    const providers = page.getByTestId("auth-providers");
    await providers.getByRole("button", { name: "disconnect" }).click();
    const authModal = page.getByTestId("auth-provider-modal");
    await expectModalVisible(authModal);
    await authModal.getByRole("button", { name: "Disconnect" }).click();
    await expectModalHidden(authModal);
    await expect(providers.getByRole("button", { name: "connect" })).toBeVisible();

    await providers.getByRole("button", { name: "connect" }).click();
    await expectModalVisible(authModal);
    const modalCaptcha = authModal.getByLabel("Enter the characters shown in the image.");
    await expect(modalCaptcha).toBeVisible();
    await modalCaptcha.fill("1234");
    await authModal.getByRole("button", { name: "Continue" }).click();
    await expect(authModal).toContainText("Demo Auth is now connected and ready to use.");
    await authModal.getByText("Close", { exact: true }).click();
    await expectModalHidden(authModal);
    await expect(providers.getByRole("button", { name: "disconnect" })).toBeVisible();
  });

  test("link and pasted code", async ({ page }) => {
    const vehicleModal = await openStepOne(page, "code");
    // credentials are not used by the code flow but the template requires them
    await vehicleModal.getByLabel("Password").fill("unused");
    await vehicleModal.getByRole("button", { name: "Prepare connection" }).click();

    // link to the vendor login and a code field, no connect button
    const loginLink = vehicleModal.getByRole("link", { name: "Open login page" });
    await expect(loginLink).toBeVisible();
    const code = vehicleModal.getByLabel(
      "Paste the code or the address of the page you were redirected to."
    );
    await expect(code).toBeVisible();
    await expect(
      vehicleModal.getByRole("button", { name: "Prepare connection" })
    ).not.toBeVisible();

    // vendor login opens in a new tab and ends on a redirect page carrying the code
    const popupPromise = page.waitForEvent("popup");
    await loginLink.click();
    const popup = await popupPromise;
    await popup.getByRole("button", { name: "Login Successfully" }).click();
    await popup.waitForURL(/\/void\?.*code=/);
    const redirectUrl = popup.url();
    await popup.close();

    // wrong code is rejected, the field stays
    await code.fill("wrong");
    await vehicleModal.getByRole("button", { name: "Continue" }).click();
    await expect(vehicleModal).toContainText("invalid code");
    await expect(code).toBeVisible();

    // pasting the whole redirect url completes the login
    await code.fill(redirectUrl);
    await vehicleModal.getByRole("button", { name: "Continue" }).click();
    await expect(vehicleModal.getByLabel("Title")).toBeVisible();
    await expect(code).not.toBeVisible();

    await saveVehicle(page, vehicleModal);
  });
});
