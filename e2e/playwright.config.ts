import { defineConfig, devices } from "@playwright/test";

/**
 * Playwright configuration for LacyLights E2E tests.
 *
 * Tests run against:
 * - Backend: http://localhost:4000 (GraphQL API)
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
  timeout: 30000, // 30 second timeout per test (first test needs app startup time)

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
        // In CI, server-side CORS_ALLOW_ALL=true handles cross-origin requests.
        // These browser flags provide additional CORS relaxation as a fallback.
        launchOptions: process.env.CI
          ? {
              args: [
                "--disable-web-security",
                "--disable-site-isolation-trials",
                "--allow-running-insecure-content",
                "--disable-features=IsolateOrigins,site-per-process,BlockInsecurePrivateNetworkRequests",
              ],
            }
          : undefined,
      },
    },
  ],

  globalSetup: "./global-setup.ts",
  globalTeardown: "./global-teardown.ts",
});
