import { defineConfig, devices } from "@playwright/test";

/**
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: "./tests",
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 4 : 0,
  timeout: 30000, // default 30s
  expect: {
    // Increase timeout for assertions to handle slow modal animations
    timeout: 10000,
  },
  reporter: [[process.env.CI ? "github" : "list"], ["html", { open: "never" }]],
  use: {
    baseURL: "http://127.0.0.1:7070",
    trace: "on-first-retry",
    video: "on-first-retry",
    screenshot: "only-on-failure",
    permissions: ["clipboard-write"],
  },
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        viewport: { width: 1400, height: 1400 },
      },
    },
  ],
});
