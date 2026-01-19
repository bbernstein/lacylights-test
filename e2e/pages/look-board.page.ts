import { Page, expect } from "@playwright/test";
import { BasePage } from "./base.page";

/**
 * Page object for the Look Board list page (/look-board).
 */
export class LookBoardListPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async goto(): Promise<void> {
    await super.goto("/look-board");
    await this.waitForHeading("Look Boards");
  }

  /**
   * Create a new look board.
   */
  async createBoard(name: string, description?: string): Promise<void> {
    // Use exact match to avoid matching Undo button that may contain "Create" in its title
    await this.page
      .getByRole("button", { name: "New Look Board", exact: true })
      .click();

    // Wait for modal - it shows "Create Look Board" heading
    const modalHeading = this.page.getByText("Create Look Board");
    await expect(modalHeading).toBeVisible();

    // Fill in board name - use placeholder to find the input
    const nameInput = this.page.getByPlaceholder(/main stage|act 1|house lights/i);
    await nameInput.fill(name);

    if (description) {
      await this.page.getByPlaceholder(/optional description/i).fill(description);
    }

    // Click "Create Board" button
    await this.page.getByRole("button", { name: /create board/i }).click();

    // Wait for modal to close
    await expect(modalHeading).toBeHidden({ timeout: 10000 });
    await this.waitForLoading();
  }

  /**
   * Open a look board by name.
   */
  async openBoard(name: string): Promise<void> {
    // Make sure we're on the look board list page
    await this.goto();

    // Click on the board card/row - look for the board name in the list
    const boardCard = this.page.locator(`text=${name}`).first();
    await boardCard.click();

    // Wait for navigation to the specific board
    // URL format is /look-board/?board=[id] or /look-board/[id]
    await this.page.waitForURL(/\/look-board\/(\?board=)?[a-z0-9-]+/, { timeout: 10000 });
  }

  /**
   * Get the count of look boards.
   */
  async getBoardCount(): Promise<number> {
    // Count board cards or list items
    const cards = await this.page.locator("[class*='shadow'][class*='rounded']").count();
    return cards;
  }

  /**
   * Check if a board with the given name exists.
   */
  async hasBoard(name: string): Promise<boolean> {
    return await this.hasText(name);
  }
}

/**
 * Page object for a specific Look Board (/look-board/[id]).
 */
export class LookBoardPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  /**
   * Wait for the board canvas to be ready.
   */
  async waitForCanvas(): Promise<void> {
    await this.waitForLoading();
    // Canvas or board container should be visible
    await this.page.waitForTimeout(1000);
  }

  /**
   * Exit focus mode if we're in it (focus mode hides the toolbar).
   */
  async exitFocusMode(): Promise<void> {
    const exitFocusButton = this.page.getByRole("button", { name: /exit focus mode/i });
    if (await exitFocusButton.isVisible()) {
      await exitFocusButton.click();
      await this.page.waitForTimeout(500);
    }
  }

  /**
   * Switch to Layout Mode (required to add/edit buttons).
   */
  async switchToLayoutMode(): Promise<void> {
    // First exit focus mode if we're in it - focus mode hides the toolbar
    await this.exitFocusMode();

    // Check if we're already in layout mode by looking for the mode buttons
    const layoutButton = this.page.getByRole("button", { name: /layout mode/i });

    // If the layout button exists and is visible, click it to switch modes
    if (await layoutButton.isVisible()) {
      await layoutButton.click();
      await this.page.waitForTimeout(500);
    }
  }

  /**
   * Add a look to the board.
   */
  async addLookToBoard(lookName: string, position?: { x: number; y: number }): Promise<void> {
    // First ensure we're in Layout Mode (required to add looks)
    await this.switchToLayoutMode();

    // Look for the add button - could be "Add Your First Look" (empty state) or "+ Add Look" (toolbar)
    // Use the first button that matches, which should be the one to open the modal
    const addButton = this.page.getByRole("button", { name: /add.*look|add your first look/i }).first();
    await addButton.click();

    // Wait for modal with look list to appear
    await this.page.waitForTimeout(500);

    // Select the look by clicking its checkbox or row
    // The modal shows a list of available looks with checkboxes
    const lookCheckbox = this.page.locator(`label:has-text("${lookName}")`).first();
    if (await lookCheckbox.isVisible()) {
      await lookCheckbox.click();
    } else {
      // Fallback - click directly on the look name
      const lookOption = this.page.getByText(lookName, { exact: true }).first();
      await lookOption.click();
    }

    // Wait for selection to register
    await this.page.waitForTimeout(300);

    // Click the confirm button - should match "Add N Look(s)" pattern
    // Use a more specific pattern that won't match "Add Your First Look"
    const confirmButton = this.page.getByRole("button", { name: /^add \d+ look/i });
    await confirmButton.click();

    await this.waitForLoading();
  }

  /**
   * Click a look button on the board.
   * Note: Should be in Play mode for this to activate the look.
   */
  async clickLookButton(lookName: string): Promise<void> {
    // Use getByRole with accessible name pattern matching
    // Button names are "Drag look X" in Layout mode and "Activate look X" in Play mode
    const button = this.page.getByRole("button", {
      name: new RegExp(`(Drag|Activate) look ${lookName}`, "i"),
    });
    await button.click();
  }

  /**
   * Get the count of buttons on the board.
   */
  async getButtonCount(): Promise<number> {
    // Count look buttons (exclude toolbar buttons)
    const buttons = await this.page
      .locator("[class*='absolute'][class*='rounded']")
      .count();
    return buttons;
  }

  /**
   * Check if a button for a look exists.
   * Note: Button names vary by mode - "Activate look X" in Play mode, "Drag look X" in Layout mode
   */
  async hasLookButton(lookName: string): Promise<boolean> {
    // Use getByRole with accessible name pattern matching
    // Button names are "Drag look X" in Layout mode and "Activate look X" in Play mode
    const button = this.page.getByRole("button", {
      name: new RegExp(`(Drag|Activate) look ${lookName}`, "i"),
    });
    // Wait for DOM to settle then check count
    await this.page.waitForTimeout(500);
    const count = await button.count();
    return count > 0;
  }
}
