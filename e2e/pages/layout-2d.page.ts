import { Page, expect } from "@playwright/test";
import { BasePage } from "./base.page";

/**
 * Timeout constants for canvas operations.
 * Documented reasoning for each timeout value.
 */
/**
 * Timeout constants for canvas operations.
 *
 * Note: These fixed timeouts are intentional and appropriate for canvas-based E2E tests.
 * Unlike DOM elements that can be polled for state changes, the canvas rendering pipeline
 * and WebSocket pubsub don't expose pollable state. These values are tuned for reliability
 * across local and CI environments.
 */
const TIMEOUTS = {
  /** Time for canvas rendering to stabilize after navigation */
  CANVAS_STABILIZATION: 500,
  /** Time for drag operations to register in the canvas */
  DRAG_REGISTRATION: 300,
  /** Time for WebSocket pubsub delivery (typically <500ms, allow 1.5s for CI) */
  PUBSUB_DELIVERY: 1500,
  /** Small delay between rapid sequential actions */
  RAPID_ACTION: 50,
  /** Delay between drag movement steps */
  DRAG_STEP: 20,
} as const;

/**
 * Page object for the 2D Layout view within the Look Editor.
 * Extends BasePage to inherit common navigation and utility methods.
 */
export class Layout2DPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  /**
   * Navigate directly to a look's edit page.
   * @param lookId - The ID of the look to edit
   */
  async goto(lookId: string): Promise<void> {
    await super.goto(`/looks/${lookId}/edit`);
  }

  /**
   * Switch to the 2D Layout view.
   */
  async switchTo2DLayout(): Promise<void> {
    const layoutButton = this.page.getByRole("button", { name: /2D Layout/i });
    await expect(layoutButton).toBeVisible({ timeout: 10000 });
    await layoutButton.click();
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
   * @returns true if there are pending changes to save
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
    const modifier = process.platform === "darwin" ? "Meta" : "Control";
    await this.page.keyboard.press(`${modifier}+z`);
    await this.page.waitForTimeout(TIMEOUTS.CANVAS_STABILIZATION);
  }

  /**
   * Perform redo via keyboard shortcut.
   */
  async redo(): Promise<void> {
    const modifier = process.platform === "darwin" ? "Meta" : "Control";
    await this.page.keyboard.press(`${modifier}+Shift+z`);
    await this.page.waitForTimeout(TIMEOUTS.CANVAS_STABILIZATION);
  }

  /**
   * Wait for canvas to stabilize after navigation or state changes.
   */
  async waitForCanvasStabilization(): Promise<void> {
    await this.page.waitForTimeout(TIMEOUTS.CANVAS_STABILIZATION);
  }

  /**
   * Wait for pubsub updates to be delivered via WebSocket.
   */
  async waitForPubsubDelivery(): Promise<void> {
    await this.page.waitForTimeout(TIMEOUTS.PUBSUB_DELIVERY);
  }

  /**
   * Drag a fixture on the canvas by simulating mouse events.
   * Since fixtures are rendered on canvas, we need to use coordinates.
   *
   * @param startX - Starting X coordinate relative to canvas
   * @param startY - Starting Y coordinate relative to canvas
   * @param endX - Ending X coordinate relative to canvas
   * @param endY - Ending Y coordinate relative to canvas
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
      throw new Error(
        `Canvas bounding box not found during drag operation from (${startX},${startY}) to (${endX},${endY}). ` +
          "Ensure the canvas is visible and rendered before calling dragOnCanvas()."
      );
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
    // Using 5 steps provides smooth movement while being efficient
    const steps = 5;
    for (let i = 1; i <= steps; i++) {
      const x = absStartX + ((absEndX - absStartX) * i) / steps;
      const y = absStartY + ((absEndY - absStartY) * i) / steps;
      await this.page.mouse.move(x, y);
      await this.page.waitForTimeout(TIMEOUTS.DRAG_STEP);
    }

    await this.page.mouse.up();
    await this.page.waitForTimeout(TIMEOUTS.DRAG_REGISTRATION);
  }

  /**
   * Click on a specific position on the canvas.
   * @param x - X coordinate relative to canvas
   * @param y - Y coordinate relative to canvas
   */
  async clickOnCanvas(x: number, y: number): Promise<void> {
    const canvas = this.getCanvas();
    const box = await canvas.boundingBox();

    if (!box) {
      throw new Error(
        `Canvas bounding box not found during click at (${x},${y}). ` +
          "Ensure the canvas is visible and rendered before calling clickOnCanvas()."
      );
    }

    await this.page.mouse.click(box.x + x, box.y + y);
  }
}
