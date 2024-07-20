import { defineConfig, devices } from "@playwright/test";

/**
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: "./tests",
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 3 : 0,
  timeout: 30000, // default 30s
  workers: process.env.CI ? 3 : 4,
  reporter: "html",
  use: {
    baseURL: "http://127.0.0.1:7070",
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
});
