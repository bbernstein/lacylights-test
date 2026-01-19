import { test, expect } from "@playwright/test";
import { DashboardPage } from "../pages/dashboard.page";
import { FixturesPage } from "../pages/fixtures.page";
import { LooksPage } from "../pages/looks.page";
import { EffectsPage } from "../pages/effects.page";

/**
 * LacyLights Dashboard E2E Tests
 *
 * This test suite validates the dashboard page functionality:
 * - All dashboard cards are displayed (Fixtures, Looks, Effects, Look Boards, Cue Lists, Settings)
 * - Cards show correct counts and item lists
 * - Navigation links work correctly
 * - Effects card shows effects with type indicators (tests WAVEFORM/purple type)
 *
 * NOTE: Only WAVEFORM effect type is tested. Additional effect types (CROSSFADE/blue,
 * MASTER/yellow, STATIC/gray) could be added for more comprehensive coverage but would
 * increase test execution time significantly.
 *
 * Tests run in serial mode to maintain state between tests.
 */
test.describe("Dashboard", () => {
  test.describe.configure({ mode: "serial" });

  // In CI, set up a fallback proxy in case any code still references port 4000.
  // TODO: Extract this proxy logic to a shared test fixture to reduce duplication
  // across test files (also used in happy-path.spec.ts).
  test.beforeEach(async ({ page }) => {
    if (process.env.CI) {
      await page.route("**/localhost:4000/**", async (route) => {
        const request = route.request();
        const originalUrl = request.url();
        const proxiedUrl = originalUrl.replace(":4000", ":4001");

        if (request.method() === "OPTIONS") {
          await route.fulfill({
            status: 204,
            headers: {
              "Access-Control-Allow-Origin": "*",
              "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
              "Access-Control-Allow-Headers": "Content-Type, Authorization, Accept",
              "Access-Control-Max-Age": "86400",
            },
          });
          return;
        }

        const response = await route.fetch({ url: proxiedUrl });
        await route.fulfill({
          response,
          headers: {
            ...response.headers(),
            "Access-Control-Allow-Origin": "*",
          },
        });
      });
    }
  });

  // Test data for creating entities
  const testData = {
    fixture: {
      name: "Dashboard Test Fixture",
      manufacturer: "Generic",
      model: "RGB Fader",
      universe: 1,
      startChannel: 100,
    },
    look: {
      name: "Dashboard Test Look",
      description: "A look created for dashboard testing",
    },
    effect: {
      name: "Dashboard Test Effect",
      type: "Waveform (LFO)" as const,
      waveform: "Sine" as const,
    },
  };

  test("1. Dashboard displays all cards", async ({ page }) => {
    const dashboardPage = new DashboardPage(page);
    await dashboardPage.goto();

    // Verify all cards are visible
    expect(await dashboardPage.hasAllCards()).toBe(true);

    // Verify individual cards
    await expect(dashboardPage.fixturesCard).toBeVisible();
    await expect(dashboardPage.looksCard).toBeVisible();
    await expect(dashboardPage.effectsCard).toBeVisible();
    await expect(dashboardPage.lookBoardsCard).toBeVisible();
    await expect(dashboardPage.cueListsCard).toBeVisible();
    await expect(dashboardPage.settingsCard).toBeVisible();
  });

  test("2. Dashboard has View all links for each card", async ({ page }) => {
    const dashboardPage = new DashboardPage(page);
    await dashboardPage.goto();

    // Should have 6 "View all" links (one for each card)
    const viewAllCount = await dashboardPage.getViewAllLinksCount();
    expect(viewAllCount).toBe(6);
  });

  test("3. Create test fixture and verify it appears on dashboard", async ({ page }) => {
    // First create a fixture
    const fixturesPage = new FixturesPage(page);
    await fixturesPage.goto();
    await fixturesPage.addFixture(testData.fixture);
    expect(await fixturesPage.hasFixture(testData.fixture.name)).toBe(true);

    // Now check the dashboard
    const dashboardPage = new DashboardPage(page);
    await dashboardPage.goto();

    // Verify fixture count is at least 1
    const fixtureCount = await dashboardPage.getCardCount("fixtures");
    expect(fixtureCount).toBeGreaterThanOrEqual(1);

    // Verify fixture name appears in the card
    expect(await dashboardPage.hasFixtureInCard(testData.fixture.name)).toBe(true);
  });

  test("4. Create test look and verify it appears on dashboard", async ({ page }) => {
    // First create a look
    const looksPage = new LooksPage(page);
    await looksPage.goto();
    await looksPage.createLook(testData.look.name, testData.look.description);
    expect(await looksPage.hasLook(testData.look.name)).toBe(true);

    // Now check the dashboard
    const dashboardPage = new DashboardPage(page);
    await dashboardPage.goto();

    // Verify look count is at least 1
    const lookCount = await dashboardPage.getCardCount("looks");
    expect(lookCount).toBeGreaterThanOrEqual(1);

    // Verify look name appears in the card
    expect(await dashboardPage.hasLookInCard(testData.look.name)).toBe(true);
  });

  test("5. Create test effect and verify it appears on dashboard", async ({ page }) => {
    // First create an effect
    const effectsPage = new EffectsPage(page);
    await effectsPage.goto();
    await effectsPage.createEffect({
      name: testData.effect.name,
      type: testData.effect.type,
      waveform: testData.effect.waveform,
    });
    expect(await effectsPage.hasEffect(testData.effect.name)).toBe(true);

    // Now check the dashboard
    const dashboardPage = new DashboardPage(page);
    await dashboardPage.goto();

    // Verify effect count is at least 1
    const effectCount = await dashboardPage.getCardCount("effects");
    expect(effectCount).toBeGreaterThanOrEqual(1);

    // Verify effect name appears in the card
    expect(await dashboardPage.hasEffectInCard(testData.effect.name)).toBe(true);

    // Verify effect type indicator (WAVEFORM = purple)
    expect(
      await dashboardPage.hasEffectTypeIndicator(testData.effect.name, "purple")
    ).toBe(true);
  });

  test("6. Effects card link navigates to effects page", async ({ page }) => {
    const dashboardPage = new DashboardPage(page);
    await dashboardPage.goto();

    // Click on the Effects card title link
    await dashboardPage.clickCardLink("effects");

    // Should navigate to effects page
    await expect(page).toHaveURL(/\/effects/);
  });

  test("7. Effects card View all link navigates to effects page", async ({ page }) => {
    const dashboardPage = new DashboardPage(page);
    await dashboardPage.goto();

    // Click on the View all link in the Effects card
    await dashboardPage.clickViewAll("effects");

    // Should navigate to effects page
    await expect(page).toHaveURL(/\/effects/);
  });

  test("8. Dashboard shows Settings card with system info", async ({ page }) => {
    const dashboardPage = new DashboardPage(page);
    await dashboardPage.goto();

    // Settings card should show Art-Net status
    expect(
      await dashboardPage.cardContainsText("settings", "Art-Net Output")
    ).toBe(true);

    // Settings card should show Fade Update Rate
    expect(
      await dashboardPage.cardContainsText("settings", "Fade Update Rate")
    ).toBe(true);
  });

  test("9. All dashboard card title links work", async ({ page }) => {
    const dashboardPage = new DashboardPage(page);

    // Test Fixtures link
    await dashboardPage.goto();
    await dashboardPage.clickCardLink("fixtures");
    await expect(page).toHaveURL(/\/fixtures/);

    // Test Looks link
    await dashboardPage.goto();
    await dashboardPage.clickCardLink("looks");
    await expect(page).toHaveURL(/\/looks/);

    // Test Effects link
    await dashboardPage.goto();
    await dashboardPage.clickCardLink("effects");
    await expect(page).toHaveURL(/\/effects/);

    // Test Look Boards link
    await dashboardPage.goto();
    await dashboardPage.clickCardLink("look-boards");
    await expect(page).toHaveURL(/\/look-board/);

    // Test Cue Lists link
    await dashboardPage.goto();
    await dashboardPage.clickCardLink("cue-lists");
    await expect(page).toHaveURL(/\/cue-lists/);

    // Test Settings link
    await dashboardPage.goto();
    await dashboardPage.clickCardLink("settings");
    await expect(page).toHaveURL(/\/settings/);
  });
});

/**
 * Cleanup test - runs last to clean up test data.
 */
test.describe("Dashboard Cleanup", () => {
  test.skip("Delete test data", async ({ page }) => {
    // This test is skipped by default
    // Enable it if you want to clean up after running the dashboard tests

    // Delete test fixture
    const fixturesPage = new FixturesPage(page);
    await fixturesPage.goto();
    await fixturesPage.deleteFixture("Dashboard Test Fixture");

    // Delete test look
    const looksPage = new LooksPage(page);
    await looksPage.goto();
    await looksPage.deleteLook("Dashboard Test Look");

    // Delete test effect
    const effectsPage = new EffectsPage(page);
    await effectsPage.goto();
    await effectsPage.deleteEffect("Dashboard Test Effect");
  });
});
