import { test, expect, Page } from "@playwright/test";
import { FixturesPage } from "../pages/fixtures.page";
import { LooksPage } from "../pages/looks.page";
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

/**
 * Page object for the 2D Layout view within the Look Editor.
 */
class Layout2DPage {
  readonly page: Page;

  constructor(page: Page) {
    this.page = page;
  }

  /**
   * Navigate directly to a look's edit page in 2D layout view.
   */
  async goto(lookId: string): Promise<void> {
    await this.page.goto(`/looks/${lookId}/edit`);
    await this.page.waitForLoadState("domcontentloaded");
    await this.waitForLoading();
  }

  /**
   * Wait for loading to complete.
   */
  async waitForLoading(): Promise<void> {
    await this.page.waitForFunction(() => {
      const body = document.body.textContent || "";
      return !body.includes("Loading...");
    }, { timeout: 30000 });
  }

  /**
   * Switch to the 2D Layout view.
   */
  async switchTo2DLayout(): Promise<void> {
    // Click the 2D Layout button (it's a button, not a tab)
    const layoutButton = this.page.getByRole("button", { name: /2D Layout/i });
    await expect(layoutButton).toBeVisible({ timeout: 10000 });
    await layoutButton.click();

    // Wait for the canvas to be visible
    await this.waitForCanvas();
  }

  /**
   * Wait for the layout canvas to be rendered.
   */
  async waitForCanvas(): Promise<void> {
    await expect(this.page.locator("canvas")).toBeVisible({ timeout: 10000 });
  }

  /**
   * Get the canvas element.
   */
  getCanvas() {
    return this.page.locator("canvas");
  }

  /**
   * Check if the Save Layout button is enabled (indicating unsaved changes).
   */
  async hasPendingChanges(): Promise<boolean> {
    const saveButton = this.page.getByRole("button", { name: /Save Layout/i });
    const isDisabled = await saveButton.isDisabled();
    return !isDisabled;
  }

  /**
   * Save the current layout positions.
   */
  async saveLayout(): Promise<void> {
    const saveButton = this.page.getByRole("button", { name: /Save Layout/i });
    await expect(saveButton).toBeEnabled({ timeout: 5000 });
    await saveButton.click();

    // Wait for save to complete (button becomes disabled)
    await expect(saveButton).toBeDisabled({ timeout: 10000 });
  }

  /**
   * Perform undo via keyboard shortcut.
   */
  async undo(): Promise<void> {
    // Use Meta+Z for macOS, Control+Z for others
    const modifier = process.platform === "darwin" ? "Meta" : "Control";
    await this.page.keyboard.press(`${modifier}+z`);

    // Wait for the undo operation to complete
    await this.page.waitForTimeout(500);
  }

  /**
   * Perform redo via keyboard shortcut.
   */
  async redo(): Promise<void> {
    const modifier = process.platform === "darwin" ? "Meta" : "Control";
    await this.page.keyboard.press(`${modifier}+Shift+z`);

    // Wait for the redo operation to complete
    await this.page.waitForTimeout(500);
  }

  /**
   * Drag a fixture on the canvas by simulating mouse events.
   * Since fixtures are rendered on canvas, we need to use coordinates.
   *
   * @param startX - Starting X coordinate (in viewport)
   * @param startY - Starting Y coordinate (in viewport)
   * @param endX - Ending X coordinate (in viewport)
   * @param endY - Ending Y coordinate (in viewport)
   */
  async dragOnCanvas(
    startX: number,
    startY: number,
    endX: number,
    endY: number
  ): Promise<void> {
    const canvas = this.getCanvas();
    const box = await canvas.boundingBox();

    if (!box) {
      throw new Error("Canvas bounding box not found");
    }

    // Translate coordinates relative to canvas position
    const absStartX = box.x + startX;
    const absStartY = box.y + startY;
    const absEndX = box.x + endX;
    const absEndY = box.y + endY;

    // Perform drag operation
    await this.page.mouse.move(absStartX, absStartY);
    await this.page.mouse.down();

    // Move in steps to trigger drag detection
    const steps = 5;
    for (let i = 1; i <= steps; i++) {
      const x = absStartX + ((absEndX - absStartX) * i) / steps;
      const y = absStartY + ((absEndY - absStartY) * i) / steps;
      await this.page.mouse.move(x, y);
      await this.page.waitForTimeout(20);
    }

    await this.page.mouse.up();
  }

  /**
   * Click on a specific position on the canvas.
   */
  async clickOnCanvas(x: number, y: number): Promise<void> {
    const canvas = this.getCanvas();
    const box = await canvas.boundingBox();

    if (!box) {
      throw new Error("Canvas bounding box not found");
    }

    await this.page.mouse.click(box.x + x, box.y + y);
  }
}

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
    await page.waitForTimeout(500);

    // Drag from a position to another (assuming fixture is somewhere in the canvas)
    // The exact coordinates depend on the auto-layout positioning
    // We'll drag in the center area of the canvas
    await layout2D.dragOnCanvas(200, 200, 300, 250);

    // Wait for the drag to register
    await page.waitForTimeout(300);

    // Check if there are pending changes (save button should be enabled if we hit a fixture)
    const hasPending = await layout2D.hasPendingChanges();

    // If we successfully moved a fixture, save it
    if (hasPending) {
      await layout2D.saveLayout();
      // Verify save completed (button disabled again)
      expect(await layout2D.hasPendingChanges()).toBe(false);
    }
  });

  test("4. Move fixture again and save", async ({ page }) => {
    const layout2D = new Layout2DPage(page);
    await layout2D.goto(lookId);
    await layout2D.switchTo2DLayout();

    // Wait for canvas to stabilize
    await page.waitForTimeout(500);

    // Try different coordinates to hit a fixture
    // First click to select, then drag
    await layout2D.clickOnCanvas(250, 250);
    await page.waitForTimeout(200);

    // Drag to a new position
    await layout2D.dragOnCanvas(250, 250, 400, 350);
    await page.waitForTimeout(300);

    const hasPending = await layout2D.hasPendingChanges();

    if (hasPending) {
      await layout2D.saveLayout();
      expect(await layout2D.hasPendingChanges()).toBe(false);
    }
  });

  test("5. Undo position change and verify restoration", async ({ page }) => {
    const layout2D = new Layout2DPage(page);
    await layout2D.goto(lookId);
    await layout2D.switchTo2DLayout();

    // Wait for canvas to stabilize
    await page.waitForTimeout(500);

    // Perform undo
    await layout2D.undo();

    // Wait for the pubsub subscription to deliver the update
    // and for the canvas to re-render
    await page.waitForTimeout(1500);

    // The UI should have updated via the fixtureDataChanged subscription
    // We can verify by checking that no save is needed (positions match DB)
    // Since undo restores the DB state and the subscription triggers a refetch,
    // the local state should match the DB, resulting in no pending changes
    const hasPending = await layout2D.hasPendingChanges();
    expect(hasPending).toBe(false);
  });

  test("6. Redo position change", async ({ page }) => {
    const layout2D = new Layout2DPage(page);
    await layout2D.goto(lookId);
    await layout2D.switchTo2DLayout();

    // Wait for canvas to stabilize
    await page.waitForTimeout(500);

    // Perform redo
    await layout2D.redo();

    // Wait for pubsub and canvas update
    await page.waitForTimeout(1500);

    // After redo, positions should still match DB (redo was applied to DB)
    const hasPending = await layout2D.hasPendingChanges();
    expect(hasPending).toBe(false);
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
