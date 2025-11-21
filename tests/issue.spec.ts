import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorHost } from "./simulator";
import { enableExperimental, expectModalVisible, expectModalHidden } from "./utils";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await startSimulator();
});

test.afterAll(async () => {
  await stopSimulator();
});

test.afterEach(async () => {
  await stop();
});

const REDACT_CONFIG = "sponsor.evcc.yaml";
const CONFIG = "issue.evcc.yaml";

test.describe("issue creation", () => {
  test("verify evcc.yaml redaction", async ({ page }) => {
    await start(REDACT_CONFIG);
    await page.goto("/#/issue");

    // get configuration (yaml)
    await page
      .getByTestId("issueYamlConfig-additional-item")
      .getByRole("button", { name: "show details" })
      .click();
    const modal = page.getByTestId("issueYamlConfig-modal");
    await expectModalVisible(modal);
    const configContent = await modal.getByRole("textbox").inputValue();

    // check for redation
    expect(configContent).toContain("sponsortoken: *****");
    expect(configContent).toContain("user: *****");
    expect(configContent).toContain("password: *****");

    // ensure redacted values are not present
    expect(configContent).not.toContain("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9");
    expect(configContent).not.toContain("test@example.org");
    expect(configContent).not.toContain("none");

    // verify other poarts
    expect(configContent).toContain("site:");
    expect(configContent).toContain("loadpoints:");
  });

  test("create issue via ui", async ({ page }) => {
    await start(CONFIG);
    await page.goto("/#/config");

    // Enable experimental features
    await enableExperimental(page, false);

    // Create a Shelly meter with username (to test private data redaction)
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Shelly 1PM");
    await meterModal.getByLabel("IP address or hostname").fill(simulatorHost());
    await meterModal.getByLabel("Username").fill("testuser@example.com");
    await meterModal.getByLabel("Password").fill("secretpass");

    await meterModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("grid")).toBeVisible();

    // Restart to apply changes
    await restart(CONFIG);
    await page.reload();

    // Navigate to issue creation from config page
    await page.getByRole("link", { name: "Report a problem" }).click();
    await expect(page.getByRole("heading", { name: "Report a problem" })).toBeVisible();

    await expect(page.getByRole("button", { name: "Start GitHub Discussion..." })).toBeVisible();
    await page.getByRole("radio", { name: "Found a bug" }).click();
    await expect(page.getByRole("button", { name: "Create GitHub Issue..." })).toBeVisible();

    // Fill out the issue form
    await page.getByLabel("Title").fill("Kaboom");
    await page
      .getByLabel("Description")
      .fill("This is a test issue created from the config page workflow");
    await page
      .getByLabel("Steps to reproduce")
      .fill("1. Go to config\n2. Enable experimental\n3. Add meter\n4. Report issue");

    // check yaml data
    const yamlItem = page.getByTestId("issueYamlConfig-additional-item");
    await yamlItem.getByRole("button", { name: "show details" }).click();
    const yamlModal = page.getByTestId("issueYamlConfig-modal");
    await expectModalVisible(yamlModal);
    await expect(yamlModal.getByRole("textbox")).toHaveValue(/carport_pv/);
    await yamlModal.getByRole("button", { name: "Close" }).first().click();
    await expectModalHidden(yamlModal);

    // check ui data and verify private data redaction
    const uiItem = page.getByTestId("issueUiConfig-additional-item");
    await uiItem.getByRole("button", { name: "show details" }).click();
    const uiModal = page.getByTestId("issueUiConfig-modal");
    await expectModalVisible(uiModal);
    const uiContent = await uiModal.getByRole("textbox").inputValue();

    // Verify meter is present but private data is redacted
    expect(uiContent).toContain("shelly"); // meter type should be visible
    expect(uiContent).not.toContain("testuser@example.com"); // user should be redacted
    expect(uiContent).not.toContain("secretpass"); // password should be redacted
    expect(uiContent).toContain("***"); // redaction marker should be present

    await uiModal.getByRole("button", { name: "Close" }).first().click();
    await expectModalHidden(uiModal);

    // check log data
    const logsItem = page.getByTestId("issueLogs-additional-item");
    await logsItem.getByRole("button", { name: "show details" }).click();
    const logsModal = page.getByTestId("issueLogs-modal");
    await expectModalVisible(logsModal);
    await expect(logsModal.getByRole("textbox")).toHaveValue(/DEBUG/);
    await logsModal.getByRole("button", { name: "Close" }).first().click();
    await expectModalHidden(logsModal);

    // check state
    const stateItem = page.getByTestId("issueState-additional-item");
    await stateItem.getByRole("button", { name: "show details" }).click();
    const stateModal = page.getByTestId("issueState-modal");
    await expectModalVisible(stateModal);
    await expect(stateModal.getByRole("textbox")).toHaveValue(/"telemetry":/);
    await stateModal.getByRole("button", { name: "Close" }).first().click();
    await expectModalHidden(stateModal);

    const stateSwitch = stateItem.getByRole("switch", { name: "include" });
    await stateSwitch.check();
    await expect(stateSwitch).toBeChecked();

    await page.getByRole("button", { name: "Create GitHub Issue..." }).click();

    // 2-step process
    const summaryModal = page.getByTestId("issue-summary-modal");
    await expectModalVisible(summaryModal);
    await expect(summaryModal.getByRole("heading", { name: /GitHub Problem/ })).toBeVisible();
    await expect(summaryModal.getByRole("heading", { name: /Step 1:/ })).toBeVisible();
    await expect(summaryModal.getByRole("heading", { name: /Step 2:/ })).toBeVisible();

    // check info in textarea
    const textarea = summaryModal.getByTestId("issue-summary-modal-textarea");
    await expect(textarea).toBeVisible();
    const textareaContent = await textarea.inputValue();
    expect(textareaContent).toContain("carport_pv"); // from evcc.yaml
    expect(textareaContent).toContain("shelly"); // from ui config
    expect(textareaContent).toContain("DEBUG"); // from logs
    expect(textareaContent).toContain('"telemetry":'); // from state

    // check only basics in github link
    let href = await summaryModal
      .getByRole("link", { name: "Create GitHub Issue" })
      .getAttribute("href");
    expect(href).toContain("https://github.com/evcc-io/evcc/issues/new?title=Kaboom&body=");
    expect(href).not.toContain("TestShelly"); // from ui config
    expect(href).not.toContain("carport_pv"); // from evcc.yaml

    // close modal
    await summaryModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(summaryModal);

    // replace long state with short custom message
    await stateItem.getByRole("button", { name: "show details" }).click();
    await expectModalVisible(stateModal);
    await stateModal.getByRole("textbox").fill("MyFancyState");
    await stateModal.getByRole("button", { name: "Apply & close" }).first().click();
    await expectModalHidden(stateModal);

    // single-step process
    await page.getByRole("button", { name: "Create GitHub Issue..." }).click();
    await expectModalVisible(summaryModal);
    await expect(summaryModal.getByRole("heading", { name: /Step 1:/ })).not.toBeVisible();
    await expect(summaryModal.getByRole("heading", { name: /Step 2:/ })).not.toBeVisible();

    // verify contents in github link
    href = await summaryModal
      .getByRole("link", { name: "Create GitHub Issue" })
      .getAttribute("href");
    expect(href).toContain("https://github.com/evcc-io/evcc/issues/new?title=Kaboom&body=");
    expect(href).toContain("shelly"); // from ui config
    expect(href).toContain("carport_pv"); // from evcc.yaml
    expect(href).toContain("DEBUG"); // from logs
    expect(href).toContain("MyFancyState"); // from state
  });
});
