const { defineConfig, devices } = require("@playwright/test");

/**
 * @see https://playwright.dev/docs/test-configuration
 */
module.exports = defineConfig({
  testDir: "./tests",
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  timeout: 15000, // 15s (default 30s)
  expect: { timeout: 2500 }, // 2.5s (default 5s)
  workers: 1, // run testfiles serially to avoid port and database conflicts
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
