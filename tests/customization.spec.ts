import { readFileSync } from "fs";
import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { openMoreMenu } from "./utils";

test.use({ baseURL: baseUrl() });

test.describe("default", async () => {
  test.beforeAll(async () => {
    await start("basics.evcc.yaml", null, ["--disable-auth"]);
  });

  test.afterAll(async () => {
    await stop();
  });

  test("manifest and page title use evcc", async ({ page }) => {
    const manifest = await (await page.request.get("/meta/site.webmanifest")).json();
    expect(manifest.name).toBe("evcc");
    expect(manifest.short_name).toBe("evcc");
    expect(manifest.icons).toHaveLength(7);

    const html = await (await page.request.get("/")).text();
    expect(html).toContain('<meta name="apple-mobile-web-app-title" content="evcc" />');
    expect(html).toContain("<title>evcc</title>");

    await page.goto("/");
    await expect(page).toHaveTitle("Hello World | evcc");
  });
});

test.describe("customization", async () => {
  test.beforeAll(async () => {
    await start("basics.evcc.yaml", null, [
      "--disable-auth",
      "--custom-brand",
      "G1GA HEMS",
      "--custom-logo-light",
      "tests/custom-logo-light.svg",
      "--custom-logo-dark",
      "tests/custom-logo-dark.svg",
      "--custom-icon",
      "tests/custom-icon.png",
      "--custom-website",
      "https://example.com/hems",
      "--custom-email",
      "support@example.com",
      "--custom-phone",
      "+49 123 456789",
    ]);
  });

  test.afterAll(async () => {
    await stop();
  });

  test("manifest, icon and page title use custom brand", async ({ page }) => {
    const manifest = await (await page.request.get("/meta/site.webmanifest")).json();
    expect(manifest.name).toBe("G1GA HEMS");
    expect(manifest.short_name).toBe("G1GA HEMS");
    expect(manifest.icons).toHaveLength(2);
    expect(manifest.icons[0].src).toBe("android-chrome-512x512.png");

    const html = await (await page.request.get("/")).text();
    expect(html).toContain('<meta name="apple-mobile-web-app-title" content="G1GA HEMS" />');
    expect(html).toContain("<title>G1GA HEMS</title>");

    const icon = await page.request.get("/meta/apple-touch-icon.png");
    expect(icon.headers()["content-type"]).toBe("image/png");
    expect(await icon.body()).toEqual(readFileSync("tests/custom-icon.png"));

    await page.goto("/");
    await expect(page).toHaveTitle("Hello World | G1GA HEMS");
  });

  test("more menu shows custom brand", async ({ page }) => {
    await page.goto("/");
    const menu = await openMoreMenu(page);
    await expect(menu.getByRole("button", { name: /G1GA HEMS/ })).toBeVisible();
  });

  test("about modal shows custom logo and contact info", async ({ page }) => {
    await page.goto("/");
    const menu = await openMoreMenu(page);
    await menu.getByRole("button", { name: /G1GA HEMS/ }).click();

    // only the theme-matching logo variant is visible
    const logo = page.getByRole("img", { name: "G1GA HEMS" });
    await expect(logo).toBeVisible();
    await expect(logo).toHaveAttribute("src", "./custom-logo-light");

    const website = page.getByRole("link", { name: "example.com", exact: true });
    await expect(website).toBeVisible();
    await expect(website).toHaveAttribute("href", "https://example.com/hems");

    const phone = page.getByRole("link", { name: "+49 123 456789" });
    await expect(phone).toBeVisible();
    await expect(phone).toHaveAttribute("href", "tel:+49123456789");

    const email = page.getByRole("link", { name: "support@example.com" });
    await expect(email).toBeVisible();
    await expect(email).toHaveAttribute("href", "mailto:support@example.com");

    // footer: powered by evcc instead of community credit
    await expect(page.getByText("powered by")).toBeVisible();
    await expect(page.getByRole("link", { name: "evcc", exact: true })).toHaveAttribute(
      "href",
      "https://evcc.io/"
    );
    await expect(page.getByRole("link", { name: "open source" })).toBeVisible();
    await expect(page.getByText("evcc community")).not.toBeVisible();
  });

  test("issue page uses email flow", async ({ page }) => {
    await page.goto("/#/issue");

    // github-only elements gone
    await expect(page.getByRole("radio")).toHaveCount(0);
    await expect(page.getByLabel("Steps to Reproduce")).toHaveCount(0);
    await expect(page.getByText("Please write your issue in English")).toHaveCount(0);

    await expect(page.getByRole("button", { name: "Send request by email" })).toBeVisible();
    await expect(
      page.getByText("Attach this file to your email if more information is needed.")
    ).toBeVisible();

    const downloadButton = page.getByRole("button", { name: "Download debug information" });
    await expect(downloadButton).toBeVisible();
    const downloadPromise = page.waitForEvent("download");
    await downloadButton.click();
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/^evcc-debug-v.+\.txt$/);
  });

  test("help modal shows contact button instead of github discussions", async ({ page }) => {
    await page.goto("/");
    const menu = await openMoreMenu(page);
    await menu.getByRole("button", { name: "Need Help?" }).click();

    const contact = page.getByRole("link", { name: "Contact us" });
    await expect(contact).toBeVisible();
    await expect(contact).toHaveAttribute("href", "mailto:support@example.com");
    await expect(page.getByRole("link", { name: "GitHub discussions" })).not.toBeVisible();
  });
});
