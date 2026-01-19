import { Page, expect } from "@playwright/test";
import { BasePage } from "./base.page";

/**
 * Page object for the Fixtures page (/fixtures).
 */
export class FixturesPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async goto(): Promise<void> {
    await super.goto("/fixtures");
    await this.waitForHeading("Fixtures");
  }

  /**
   * Click the "Add Fixture" button to open the add modal.
   */
  async openAddFixtureModal(): Promise<void> {
    // Use exact match to avoid matching Undo button that may contain "Add Fixture" in its title
    await this.page.getByRole("button", { name: "Add Fixture", exact: true }).click();
    // Wait for modal to appear
    await expect(this.page.getByRole("dialog")).toBeVisible();
  }

  /**
   * Add a new fixture via the modal.
   */
  async addFixture(options: {
    name: string;
    manufacturer?: string;
    model?: string;
    mode?: string;
    universe?: number;
    startChannel?: number;
  }): Promise<void> {
    await this.openAddFixtureModal();

    // Fill the fixture name
    await this.page.getByLabel(/fixture name/i).fill(options.name);

    if (options.manufacturer) {
      // The manufacturer is an autocomplete - type to search and click the result
      const manufacturerInput = this.page.getByLabel(/manufacturer/i);
      await manufacturerInput.click();
      await manufacturerInput.fill(options.manufacturer);

      // Wait for dropdown to show options and click the matching button
      await this.page.waitForTimeout(500);
      const manufacturerOption = this.page
        .locator("button")
        .filter({ hasText: new RegExp(`^${options.manufacturer}$`, "i") });
      await manufacturerOption.first().click();

      // Wait for models to load after manufacturer selection
      await this.page.waitForTimeout(500);
    }

    if (options.model) {
      // The model is also an autocomplete
      const modelInput = this.page.getByLabel(/model/i);
      await modelInput.click();
      await modelInput.fill(options.model);

      // Wait for dropdown and click the matching button
      await this.page.waitForTimeout(500);
      const modelOption = this.page
        .locator("button")
        .filter({ hasText: new RegExp(`^${options.model}$`, "i") });
      await modelOption.first().click();

      // Wait for mode dropdown to appear after model selection
      await this.page.waitForTimeout(500);
    }

    // Select mode if specified, or select the first available mode
    // The mode dropdown has id="mode" in the AddFixtureModal
    const modeSelect = this.page.locator("select#mode");
    if (await modeSelect.isVisible()) {
      if (options.mode) {
        await modeSelect.selectOption({ label: new RegExp(options.mode, "i") });
      } else {
        // Select the first non-empty option
        const optionValues = await modeSelect.locator("option").allTextContents();
        const firstMode = optionValues.find((opt) => opt && !opt.includes("Select"));
        if (firstMode) {
          await modeSelect.selectOption({ label: firstMode });
        }
      }
    }

    if (options.universe !== undefined) {
      const universeInput = this.page.getByLabel(/universe/i);
      await universeInput.clear();
      await universeInput.fill(String(options.universe));
    }

    if (options.startChannel !== undefined) {
      const channelInput = this.page.getByLabel(/start channel/i);
      await channelInput.clear();
      await channelInput.fill(String(options.startChannel));
    }

    // Submit the form - the button is in the modal (BottomSheet with testId="add-fixture-modal")
    const modal = this.page.getByTestId("add-fixture-modal");
    const addButton = modal.getByRole("button", { name: /add fixture/i });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();

    // Wait for modal to close
    await expect(this.page.getByRole("dialog")).toBeHidden({ timeout: 10000 });

    // Wait for fixture to appear in the list (name may be auto-generated)
    await this.waitForLoading();
  }

  /**
   * Get the count of fixtures displayed.
   */
  async getFixtureCount(): Promise<number> {
    // Look for table rows or cards
    const tableRows = await this.page.locator("tbody tr").count();
    if (tableRows > 0) {
      return tableRows;
    }
    // Mobile cards
    const cards = await this.page.locator(".space-y-4 > div").count();
    return cards;
  }

  /**
   * Check if a fixture with the given name exists.
   */
  async hasFixture(name: string): Promise<boolean> {
    return await this.hasText(name);
  }

  /**
   * Delete a fixture by name.
   */
  async deleteFixture(name: string): Promise<void> {
    this.setupDialogHandler(true);

    // Find the row/card containing this fixture
    const row = this.page.locator(`tr:has-text("${name}"), div:has-text("${name}")`).first();

    // Click the delete button (red trash icon)
    await row.getByRole("button", { name: /delete/i }).click();
  }

  /**
   * Edit a fixture by name.
   */
  async editFixture(name: string): Promise<void> {
    const row = this.page.locator(`tr:has-text("${name}"), div:has-text("${name}")`).first();
    await row.getByRole("button", { name: /edit/i }).click();
    await expect(this.page.getByRole("dialog")).toBeVisible();
  }
}
