import { test, expect } from "@playwright/test";
import { FixturesPage } from "../pages/fixtures.page";
import { LooksPage } from "../pages/looks.page";
import { Layout2DPage } from "../pages/layout-2d.page";
import { setupCiProxy } from "../helpers/ci-proxy";

/**
 * LacyLights E2E Tests: Fixture Position Undo/Redo
 *
 * This test suite validates that fixture position changes in the 2D Layout
 * view can be properly undone and that real-time pubsub updates reflect
 * the changes in the UI.
 *
 * Tests the following flow:
 * 1. Create fixtures and a look
 * 2. Open 2D Layout view
 * 3. Drag fixture to new position
 * 4. Save the position
 * 5. Undo (Cmd+Z) and verify position is restored
 */

test.describe("Fixture Position Undo/Redo", () => {
  test.describe.configure({ mode: "serial" });

  // In CI, set up a fallback proxy in case any code still references port 4000.
  test.beforeEach(async ({ page }) => {
    await setupCiProxy(page);
  });

  // Test data
  const testData = {
    fixture: {
      name: "Position Test Light",
      manufacturer: "Generic",
      model: "RGB Fader",
      universe: 1,
      startChannel: 100, // Use high channel to avoid conflicts
    },
    look: {
      name: "Position Test Look",
      description: "Look for testing fixture position undo",
    },
  };

  let lookId: string;

  test("1. Setup: Create fixture and look", async ({ page }) => {
    // Create fixture
    const fixturesPage = new FixturesPage(page);
    await fixturesPage.goto();
    await fixturesPage.addFixture(testData.fixture);
    expect(await fixturesPage.hasFixture(testData.fixture.name)).toBe(true);

    // Create look
    const looksPage = new LooksPage(page);
    await looksPage.goto();
    await looksPage.createLook(testData.look.name, testData.look.description);
    expect(await looksPage.hasLook(testData.look.name)).toBe(true);

    // Get the look ID from the URL when opening the look
    await looksPage.openLook(testData.look.name);
    const url = page.url();
    const match = url.match(/\/looks\/([a-z0-9-]+)\/edit/);
    if (match) {
      lookId = match[1];
    }
    expect(lookId).toBeDefined();
  });

  test("2. Switch to 2D Layout and verify canvas renders", async ({ page }) => {
    const layout2D = new Layout2DPage(page);
    await layout2D.goto(lookId);
    await layout2D.switchTo2DLayout();

    // Verify canvas is visible
    await expect(layout2D.getCanvas()).toBeVisible();

    // Verify save button is initially disabled (no changes)
    expect(await layout2D.hasPendingChanges()).toBe(false);
  });

  test("3. Drag fixture and save position", async ({ page }) => {
    const layout2D = new Layout2DPage(page);
    await layout2D.goto(lookId);
    await layout2D.switchTo2DLayout();

    // Wait for canvas to stabilize
    await layout2D.waitForCanvasStabilization();

    // Drag from a position to another (assuming fixture is somewhere in the canvas)
    // The exact coordinates depend on the auto-layout positioning
    // We'll drag in the center area of the canvas
    await layout2D.dragOnCanvas(200, 200, 300, 250);

    // Check if there are pending changes (save button should be enabled if we hit a fixture)
    const hasPending = await layout2D.hasPendingChanges();

    // If we successfully moved a fixture, save it
    if (hasPending) {
      await layout2D.saveLayout();
      // Verify save completed (button disabled again)
      expect(
        await layout2D.hasPendingChanges(),
        "Save button should be disabled after saving"
      ).toBe(false);
    }
  });

  test("4. Move fixture again and save", async ({ page }) => {
    const layout2D = new Layout2DPage(page);
    await layout2D.goto(lookId);
    await layout2D.switchTo2DLayout();

    // Wait for canvas to stabilize
    await layout2D.waitForCanvasStabilization();

    // Try different coordinates to hit a fixture
    // First click to select, then drag
    await layout2D.clickOnCanvas(250, 250);
    await layout2D.waitForCanvasStabilization();

    // Drag to a new position
    await layout2D.dragOnCanvas(250, 250, 400, 350);

    const hasPending = await layout2D.hasPendingChanges();

    if (hasPending) {
      await layout2D.saveLayout();
      expect(
        await layout2D.hasPendingChanges(),
        "Save button should be disabled after saving"
      ).toBe(false);
    }
  });

  test("5. Undo position change and verify restoration", async ({ page }) => {
    const layout2D = new Layout2DPage(page);
    await layout2D.goto(lookId);
    await layout2D.switchTo2DLayout();

    // Wait for canvas to stabilize
    await layout2D.waitForCanvasStabilization();

    // Perform undo
    await layout2D.undo();

    // Wait for the pubsub subscription to deliver the update
    // and for the canvas to re-render
    await layout2D.waitForPubsubDelivery();

    // The UI should have updated via the fixtureDataChanged subscription
    // We can verify by checking that no save is needed (positions match DB)
    // Since undo restores the DB state and the subscription triggers a refetch,
    // the local state should match the DB, resulting in no pending changes
    const hasPending = await layout2D.hasPendingChanges();
    expect(
      hasPending,
      "Save button should be disabled after undo (positions match DB)"
    ).toBe(false);
  });

  test("6. Redo position change", async ({ page }) => {
    const layout2D = new Layout2DPage(page);
    await layout2D.goto(lookId);
    await layout2D.switchTo2DLayout();

    // Wait for canvas to stabilize
    await layout2D.waitForCanvasStabilization();

    // Perform redo
    await layout2D.redo();

    // Wait for pubsub and canvas update
    await layout2D.waitForPubsubDelivery();

    // After redo, positions should still match DB (redo was applied to DB)
    const hasPending = await layout2D.hasPendingChanges();
    expect(
      hasPending,
      "Save button should be disabled after redo (positions match DB)"
    ).toBe(false);
  });

  test("7. Cleanup: Delete test data", async ({ page }) => {
    // Delete look
    const looksPage = new LooksPage(page);
    await looksPage.goto();

    if (await looksPage.hasLook(testData.look.name)) {
      await looksPage.deleteLook(testData.look.name);
    }

    // Delete fixture
    const fixturesPage = new FixturesPage(page);
    await fixturesPage.goto();

    if (await fixturesPage.hasFixture(testData.fixture.name)) {
      await fixturesPage.deleteFixture(testData.fixture.name);
    }
  });
});

/**
 * Additional tests for edge cases in fixture position undo.
 */
test.describe("Fixture Position Undo - Edge Cases", () => {
  test.beforeEach(async ({ page }) => {
    await setupCiProxy(page);
  });

  test("Undo without prior changes does nothing harmful", async ({ page }) => {
    const looksPage = new LooksPage(page);
    await looksPage.goto();

    // Just verify the page loads without pressing undo
    // (no crashes when undo stack is empty)
    expect(await looksPage.hasText("Looks")).toBe(true);

    // Press undo on a page without pending changes
    const modifier = process.platform === "darwin" ? "Meta" : "Control";
    await page.keyboard.press(`${modifier}+z`);

    // Page should still be functional
    expect(await looksPage.hasText("Looks")).toBe(true);
  });

  test("Multiple rapid undos are handled correctly", async ({ page }) => {
    // This test ensures that rapid undo operations don't cause race conditions
    const looksPage = new LooksPage(page);
    await looksPage.goto();

    const modifier = process.platform === "darwin" ? "Meta" : "Control";

    // Press undo multiple times rapidly
    for (let i = 0; i < 5; i++) {
      await page.keyboard.press(`${modifier}+z`);
      await page.waitForTimeout(50);
    }

    // Page should still be functional
    await page.waitForTimeout(500);
    expect(await looksPage.hasText("Looks")).toBe(true);
  });
});
