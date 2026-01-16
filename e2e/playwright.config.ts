import { defineConfig, devices } from "@playwright/test";

/**
 * Playwright configuration for LacyLights E2E tests.
 *
 * Tests run against:
 * - Backend: http://localhost:4001 (GraphQL API)
 * - Frontend: http://localhost:3001 (Next.js)
 *
 * Global setup starts both services, global teardown stops them.
 */
export default defineConfig({
  testDir: "./tests",
  fullyParallel: false, // Sequential for database state consistency
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1, // Single worker for state consistency
  reporter: [["html"], ["list"]],
  timeout: 60000, // 60 second timeout per test

  use: {
    baseURL: "http://localhost:3001",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "on-first-retry",
  },

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],

  globalSetup: "./global-setup.ts",
  globalTeardown: "./global-teardown.ts",
});
