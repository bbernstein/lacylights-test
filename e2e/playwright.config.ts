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
  timeout: 10000, // 10 second timeout per test

  use: {
    baseURL: "http://localhost:3001",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "on-first-retry",
  },

  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        // In CI, disable web security to allow cross-origin requests between
        // the static frontend (localhost:3001) and backend API (localhost:4001).
        // This is safe for testing because both services are local and trusted.
        // Note: --disable-web-security requires --user-data-dir to work properly.
        launchOptions: process.env.CI
          ? {
              args: [
                "--disable-web-security",
                "--disable-site-isolation-trials",
                "--allow-running-insecure-content",
                // Combine all disabled features into one flag to avoid conflicts
                "--disable-features=IsolateOrigins,site-per-process,BlockInsecurePrivateNetworkRequests",
                // Required for --disable-web-security to work
                "--user-data-dir=/tmp/chrome-test-profile",
              ],
            }
          : undefined,
      },
    },
  ],

  globalSetup: "./global-setup.ts",
  globalTeardown: "./global-teardown.ts",
});
