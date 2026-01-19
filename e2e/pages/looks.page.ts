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
    const row = this.page.locator(`tr:has-text("${name}"), div:has-text("${name}")`).first();
    await row.getByRole("button", { name: /edit/i }).click();

    // Wait for navigation to look editor
    await this.page.waitForURL(/\/looks\/[a-z0-9-]+\/edit/);
  }

  /**
   * Activate a look directly from the list.
   */
  async activateLook(name: string): Promise<void> {
    // Use main content area to avoid matching cue list tables
    const mainTable = this.page.locator("main tbody");
    const row = mainTable.locator(`tr:has-text("${name}")`).first();
    await row.getByRole("button", { name: /activate/i }).first().click();
  }

  /**
   * Delete a look by name.
   */
  async deleteLook(name: string): Promise<void> {
    this.setupDialogHandler(true);

    const row = this.page.locator(`tr:has-text("${name}"), div:has-text("${name}")`).first();
    // Use .first() to handle responsive layouts with multiple buttons
    await row.getByRole("button", { name: /delete/i }).first().click();
  }

  /**
   * Duplicate a look by name.
   */
  async duplicateLook(name: string): Promise<void> {
    this.setupDialogHandler(true);

    const row = this.page.locator(`tr:has-text("${name}"), div:has-text("${name}")`).first();
    // Use .first() to handle responsive layouts with multiple buttons
    await row.getByRole("button", { name: /duplicate/i }).first().click();
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
    await this.page.waitForURL(/\/looks$/);
  }
}
