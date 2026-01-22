import { Page, expect } from "@playwright/test";
import { BasePage } from "./base.page";

/**
 * Page object for the Looks page (/looks).
 */
export class LooksPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async goto(): Promise<void> {
    await super.goto("/looks");
    await this.waitForHeading("Looks");
  }

  /**
   * Get the look row/card element by name.
   * Delegates to the base class getItemRow method.
   */
  private getLookRow(name: string) {
    return this.getItemRow(name);
  }

  /**
   * Click "Create Look" button to open the modal.
   */
  async openCreateLookModal(): Promise<void> {
    // Use exact match to avoid matching Undo button that may contain "Create look" in its title
    await this.page.getByRole("button", { name: "Create Look", exact: true }).click();
    await expect(this.page.getByRole("dialog")).toBeVisible();
  }

  /**
   * Create a new look via the modal.
   */
  async createLook(name: string, description?: string): Promise<void> {
    await this.openCreateLookModal();

    await this.page.getByLabel(/name/i).fill(name);

    if (description) {
      await this.page.getByLabel(/description/i).fill(description);
    }

    // Submit - scope to the dialog to avoid matching Undo button
    const dialog = this.page.getByRole("dialog");
    await dialog.getByRole("button", { name: /create/i }).click();

    // Wait for modal to close
    await expect(this.page.getByRole("dialog")).toBeHidden({ timeout: 10000 });

    // Wait for look to appear in the table
    await this.waitForLoading();
    await expect(this.page.getByRole("cell", { name })).toBeVisible();
  }

  /**
   * Open a look for editing.
   */
  async openLook(name: string): Promise<void> {
    // Wait for page to finish loading before interacting
    await this.waitForLoading();

    const row = this.getLookRow(name);
    const editButton = row.locator('button[title="Edit look"]');

    // Wait for the button to be visible before clicking
    await expect(editButton).toBeVisible({ timeout: 10000 });
    await editButton.click();

    // Wait for navigation to look editor
    await this.page.waitForURL(/\/looks\/[a-z0-9-]+\/edit/);
  }

  /**
   * Activate a look directly from the list.
   */
  async activateLook(name: string): Promise<void> {
    // Wait for page to finish loading before interacting
    await this.waitForLoading();

    const row = this.getLookRow(name);
    const activateButton = row.locator('button[title="Activate look"]');

    // Wait for the button to be visible before clicking
    await expect(activateButton).toBeVisible({ timeout: 10000 });
    await activateButton.click();
  }

  /**
   * Delete a look by name.
   */
  async deleteLook(name: string): Promise<void> {
    this.setupDialogHandler(true);

    // Wait for page to finish loading before interacting
    await this.waitForLoading();

    const row = this.getLookRow(name);
    const deleteButton = row.locator('button[title="Delete look"]');

    // Wait for the button to be visible before clicking
    await expect(deleteButton).toBeVisible({ timeout: 10000 });
    await deleteButton.click();
  }

  /**
   * Duplicate a look by name.
   */
  async duplicateLook(name: string): Promise<void> {
    this.setupDialogHandler(true);

    // Wait for page to finish loading before interacting
    await this.waitForLoading();

    const row = this.getLookRow(name);
    const duplicateButton = row.locator('button[title="Duplicate look"]');

    // Wait for the button to be visible before clicking
    await expect(duplicateButton).toBeVisible({ timeout: 10000 });
    await duplicateButton.click();
  }

  /**
   * Get the count of looks displayed.
   */
  async getLookCount(): Promise<number> {
    const tableRows = await this.page.locator("tbody tr").count();
    if (tableRows > 0) {
      return tableRows;
    }
    const cards = await this.page.locator(".space-y-4 > div").count();
    return cards;
  }

  /**
   * Check if a look with the given name exists.
   */
  async hasLook(name: string): Promise<boolean> {
    return await this.hasText(name);
  }
}

/**
 * Page object for the Look Editor (/looks/[id]/edit).
 */
export class LookEditorPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  /**
   * Wait for the editor to load.
   */
  async waitForEditor(): Promise<void> {
    await this.waitForLoading();
    // Look for the save button as indicator editor is loaded
    await expect(this.page.getByRole("button", { name: /save/i })).toBeVisible({ timeout: 10000 });
  }

  /**
   * Set a channel value by channel index.
   */
  async setChannelValue(channelIndex: number, value: number): Promise<void> {
    // Find channel slider or input
    const channelInput = this.page.locator(`input[type="range"], input[type="number"]`).nth(channelIndex);
    await channelInput.fill(String(value));
  }

  /**
   * Save the look.
   */
  async save(): Promise<void> {
    await this.clickButton(/save/i);
    // Wait for save confirmation
    await this.page.waitForTimeout(1000);
  }

  /**
   * Go back to looks list.
   */
  async goBack(): Promise<void> {
    await this.page.goBack();
    await this.page.waitForURL(/\/looks\/?$/);
  }

  /**
   * Open the "Add Fixtures" panel to add fixtures to the look.
   */
  async openAddFixturesPanel(): Promise<void> {
    // Click "Add Fixtures" button to show available fixtures panel
    await this.page.getByRole("button", { name: /add fixtures/i }).click();
    // Wait for the panel to appear
    await expect(this.page.getByText("Available Fixtures")).toBeVisible({ timeout: 5000 });
  }

  /**
   * Add a fixture to the look by name.
   * Assumes the Add Fixtures panel is open.
   * @returns true if the fixture was added, false if already in look or not available
   */
  async addFixtureToLook(fixtureName: string): Promise<boolean> {
    // Check if all fixtures are already in the look
    const noFixturesMessage = this.page.getByText("All project fixtures are already in this look");
    if (await noFixturesMessage.isVisible({ timeout: 1000 }).catch(() => false)) {
      return false; // No fixtures available to add
    }

    // Find and check the fixture checkbox in the Available Fixtures list using exact label match
    const fixtureLabel = this.page.getByLabel(fixtureName, { exact: true });
    if (await fixtureLabel.isVisible({ timeout: 2000 }).catch(() => false)) {
      await fixtureLabel.click();
      return true;
    }
    return false; // Fixture not found (might already be in look)
  }

  /**
   * Get the count of fixtures in the look.
   */
  async getFixtureCount(): Promise<number> {
    const countText = await this.page.locator("text=/Total fixtures in look: \\d+/").textContent();
    if (!countText) return 0;
    const match = countText.match(/(\d+)/);
    return match ? parseInt(match[1], 10) : 0;
  }

  /**
   * Check if the "Copy to Looks" button is visible.
   * This button appears when fixtures are selected in the editor.
   */
  async isCopyToLooksButtonVisible(): Promise<boolean> {
    // Use title selector for consistent behavior with openCopyToLooksModal
    const button = this.page.locator('button[title="Copy selected fixtures to other looks"]');
    return await button.isVisible();
  }

  /**
   * Click the "Copy to Looks" button to open the copy modal.
   * The button appears when fixtures are selected in the editor.
   */
  async openCopyToLooksModal(): Promise<void> {
    // Click the "Copy to Looks" button using title selector (consistent with isCopyToLooksButtonVisible)
    const button = this.page.locator('button[title="Copy selected fixtures to other looks"]');
    await expect(button).toBeVisible({ timeout: 5000 });
    await button.click();
    // Wait for the modal to appear
    await expect(this.page.getByTestId("copy-fixtures-to-looks-modal")).toBeVisible({ timeout: 5000 });
  }

  /**
   * Select a target look in the Copy to Looks modal.
   * @param lookName - Name of the look to select as target
   */
  async selectTargetLook(lookName: string): Promise<void> {
    const modal = this.page.getByTestId("copy-fixtures-to-looks-modal");
    // Click the checkbox associated with the look name to select the target look
    const lookCheckbox = modal.getByRole("checkbox", { name: lookName });
    await expect(lookCheckbox).toBeVisible({ timeout: 5000 });
    await lookCheckbox.click();
  }

  /**
   * Get the count of selected target looks in the modal.
   */
  async getSelectedTargetCount(): Promise<number> {
    const modal = this.page.getByTestId("copy-fixtures-to-looks-modal");
    const selectedText = await modal.locator("text=/\\d+ selected/").textContent();
    if (!selectedText) return 0;
    const match = selectedText.match(/(\d+)/);
    return match ? parseInt(match[1], 10) : 0;
  }

  /**
   * Execute the copy operation by clicking the Copy button in the modal.
   */
  async confirmCopyToLooks(): Promise<void> {
    const modal = this.page.getByTestId("copy-fixtures-to-looks-modal");
    // Find the Copy button (it says "Copy to X Looks" or similar)
    const copyButton = modal.getByRole("button", { name: /copy to.*look/i });
    await expect(copyButton).toBeEnabled({ timeout: 5000 });
    await copyButton.click();
    // Wait for modal to close (indicates success)
    await expect(modal).toBeHidden({ timeout: 10000 });
  }

  /**
   * Cancel and close the Copy to Looks modal.
   */
  async cancelCopyToLooks(): Promise<void> {
    const modal = this.page.getByTestId("copy-fixtures-to-looks-modal");
    await modal.getByRole("button", { name: /cancel/i }).click();
    await expect(modal).toBeHidden({ timeout: 5000 });
  }

  /**
   * Select a fixture in the look editor by clicking on it.
   * This is used for layout mode where fixtures are displayed on a canvas.
   * @param fixtureName - Name of the fixture to select
   */
  async selectFixtureInEditor(fixtureName: string): Promise<void> {
    // In channels mode, fixtures are in cards/rows. Click on the fixture header.
    // Use regex to match fixture name at the start (heading includes mode info like "Generic RGB Fader • U1:1")
    const fixtureCard = this.page.getByRole("heading", { name: new RegExp(`^${fixtureName}`), level: 4 }).first();
    await expect(fixtureCard).toBeVisible({ timeout: 5000 });
    await fixtureCard.click();
  }

  /**
   * Check if a fixture is in the look by name.
   * @param fixtureName - Name of the fixture to check
   */
  async hasFixtureInLook(fixtureName: string): Promise<boolean> {
    // Use getByRole with regex to match fixture name at the start and check visibility
    // The heading includes mode info like "Generic RGB Fader • U1:1" after the fixture name
    const fixtureCard = this.page.getByRole("heading", { name: new RegExp(`^${fixtureName}`), level: 4 }).first();
    return await fixtureCard.isVisible();
  }
}
