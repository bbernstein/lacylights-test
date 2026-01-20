import { Page, expect } from "@playwright/test";
import { BasePage } from "./base.page";

/**
 * Page object for the Cue Lists page (/cue-lists).
 */
export class CueListsPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async goto(): Promise<void> {
    await super.goto("/cue-lists");
    await this.waitForHeading("Cue Lists");
  }

  /**
   * Create a new cue list.
   */
  async createCueList(name: string, description?: string): Promise<void> {
    // Use exact match to avoid matching Undo button that may contain "Create" in its title
    await this.page.getByRole("button", { name: "New Cue List", exact: true }).click();
    await expect(this.page.getByRole("dialog")).toBeVisible();

    await this.page.getByLabel(/name/i).fill(name);

    if (description) {
      await this.page.getByLabel(/description/i).fill(description);
    }

    await this.page.getByRole("button", { name: /create/i }).last().click();

    await expect(this.page.getByRole("dialog")).toBeHidden({ timeout: 10000 });
    await this.waitForLoading();
  }

  /**
   * Get the cue list row/card element by name.
   * Delegates to the base class getItemRow method.
   */
  private getCueListRow(name: string) {
    return this.getItemRow(name);
  }

  /**
   * Open a cue list by name.
   */
  async openCueList(name: string): Promise<void> {
    // Wait for page to finish loading before interacting
    await this.waitForLoading();

    const row = this.getCueListRow(name);
    const openButton = row.locator('button[title="Open cue list"]');

    // Wait for the button to be visible before clicking
    await expect(openButton).toBeVisible({ timeout: 10000 });
    await openButton.click();
    await this.page.waitForURL(/\/cue-lists\/[a-z0-9-]+/);
  }

  /**
   * Get the count of cue lists.
   */
  async getCueListCount(): Promise<number> {
    const tableRows = await this.page.locator("tbody tr").count();
    if (tableRows > 0) {
      return tableRows;
    }
    const cards = await this.page.locator(".space-y-4 > div").count();
    return cards;
  }

  /**
   * Check if a cue list with the given name exists.
   */
  async hasCueList(name: string): Promise<boolean> {
    return await this.hasText(name);
  }

  /**
   * Delete a cue list by name.
   */
  async deleteCueList(name: string): Promise<void> {
    this.setupDialogHandler(true);

    // Wait for page to finish loading before interacting
    await this.waitForLoading();

    const row = this.getCueListRow(name);
    const deleteButton = row.locator('button[title="Delete cue list"]');

    // Wait for the button to be visible before clicking
    await expect(deleteButton).toBeVisible({ timeout: 10000 });
    await deleteButton.click();
  }
}

/**
 * Page object for a specific Cue List (/cue-lists/[id]).
 */
export class CueListEditorPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  /**
   * Wait for the cue list editor to load.
   */
  async waitForEditor(): Promise<void> {
    // Wait for playback controls (like "START" button) to appear
    await this.page.getByRole("button", { name: /start|go/i }).waitFor({
      state: "visible",
      timeout: 10000,
    });
  }

  /**
   * Enter edit mode to add/modify cues.
   */
  async enterEditMode(): Promise<void> {
    // Click the Edit button to enter edit mode
    const editButton = this.page.getByRole("button", { name: /^edit$/i });
    if (await editButton.isVisible()) {
      await editButton.click();
      await this.page.waitForTimeout(500);
    }
  }

  /**
   * Exit edit mode to enable playback.
   * This navigates back to the cue lists page and reopens the cue list fresh.
   */
  async exitEditMode(): Promise<void> {
    // Click the Back button to go to cue lists, then reopen will give fresh playback mode
    const backButton = this.page.getByRole("button", { name: "Back" });
    if (await backButton.isVisible()) {
      await backButton.click();
      await this.page.waitForURL(/\/cue-lists\/?$/);
      // Wait for the page to fully load before returning
      await this.waitForLoading();
    }
  }

  /**
   * Add a cue to the list.
   */
  async addCue(options: {
    name: string;
    look: string;
    fadeIn?: number;
    fadeOut?: number;
    followTime?: number;
  }): Promise<void> {
    await this.clickButton(/add.*cue/i);
    await expect(this.page.getByRole("dialog")).toBeVisible();

    await this.page.getByLabel(/name/i).fill(options.name);

    // Select the look - use the specific "Look *" label to avoid matching checkbox
    const lookSelect = this.page.getByLabel("Look *");
    await lookSelect.selectOption({ label: options.look });

    // Fade times are under "Advanced Timing" - expand if needed
    if (options.fadeIn !== undefined || options.fadeOut !== undefined || options.followTime !== undefined) {
      const advancedButton = this.page.getByRole("button", { name: /advanced timing/i });
      if (await advancedButton.isVisible()) {
        await advancedButton.click();
        await this.page.waitForTimeout(300);
      }

      if (options.fadeIn !== undefined) {
        await this.page.getByLabel(/fade.*in/i).fill(String(options.fadeIn));
      }

      if (options.fadeOut !== undefined) {
        await this.page.getByLabel(/fade.*out/i).fill(String(options.fadeOut));
      }

      if (options.followTime !== undefined) {
        await this.page.getByLabel(/follow/i).fill(String(options.followTime));
      }
    }

    // Click "Add Only" button to add the cue
    await this.page.getByRole("button", { name: /add only/i }).click();

    await expect(this.page.getByRole("dialog")).toBeHidden({ timeout: 10000 });
    // Wait briefly for the cue list to update
    await this.page.waitForTimeout(500);
  }

  /**
   * Get the count of cues in the list.
   */
  async getCueCount(): Promise<number> {
    // Count cue rows
    const cueRows = await this.page.locator("[class*='cue'], tr").count();
    return cueRows;
  }

  /**
   * Start playback of the cue list.
   */
  async startPlayback(): Promise<void> {
    // In Player mode the button is "START", in edit mode it's "GO"
    const startButton = this.page.getByRole("button", { name: "START" });
    const goButton = this.page.getByRole("button", { name: "GO" });

    if (await startButton.isVisible()) {
      await startButton.click();
    } else if (await goButton.isVisible()) {
      await goButton.click();
    }
  }

  /**
   * Stop playback of the cue list.
   */
  async stopPlayback(): Promise<void> {
    // Use specific pattern to avoid matching Undo button "Undo: Stop cue list..."
    await this.page.getByRole("button", { name: /^Stop/ }).click();
  }

  /**
   * Go to the next cue.
   */
  async nextCue(): Promise<void> {
    // Use specific title to avoid matching Next.js Dev Tools button
    await this.page.getByRole("button", { name: "Next (→)" }).click();
  }

  /**
   * Go to the previous cue.
   */
  async previousCue(): Promise<void> {
    // Use specific title pattern to avoid matching unrelated buttons
    await this.page.getByRole("button", { name: /Prev.*\(←\)/ }).click();
  }

  /**
   * Check if a cue with the given name is currently active.
   */
  async isCueActive(cueName: string): Promise<boolean> {
    const activeCue = this.page.locator(".cue-active, [class*='active']");
    const text = await activeCue.textContent();
    return text?.includes(cueName) || false;
  }

  /**
   * Get the name of the currently active cue.
   */
  async getActiveCueName(): Promise<string | null> {
    const activeCue = this.page.locator(".cue-active, [class*='bg-blue'], [aria-current]").first();
    if (await activeCue.isVisible()) {
      return await activeCue.textContent();
    }
    return null;
  }

  /**
   * Open the edit dialog for a cue by name.
   * Uses right-click context menu to open EditCueDialog.
   */
  async editCue(cueName: string): Promise<void> {
    // Find the cue row - escape for safe use in selector
    const escapedForSelector = this.escapeTextForSelector(cueName);
    const cueRow = this.page.locator(`tr:has-text("${escapedForSelector}"), [data-cue-name="${escapedForSelector}"]`).first();

    // Right-click to open context menu
    await cueRow.click({ button: "right" });

    // Wait for context menu to appear and click "Edit Cue"
    const contextMenu = this.page.locator('[role="menu"], .context-menu, [class*="dropdown"]').first();
    await expect(contextMenu).toBeVisible({ timeout: 3000 });

    // Click the "Edit Cue" option in the context menu
    const editOption = contextMenu.getByText(/edit cue/i);
    await editOption.click();

    // Wait for the EditCueDialog (bottom sheet) to appear
    await expect(this.page.locator('[role="dialog"], [class*="sheet"], [class*="dialog"]')).toBeVisible({ timeout: 5000 });
  }

  /**
   * Add an effect to the currently edited cue.
   * Must be called after editCue() opens the dialog.
   */
  async addEffectToCue(options: {
    effectName: string;
    intensity?: number;
    speed?: number;
  }): Promise<void> {
    // Find the dialog
    const dialog = this.page.getByRole("dialog");

    // Expand the Effects section by clicking the "Effects" button
    // The Effects section is collapsed by default
    const effectsButton = dialog.getByRole("button", { name: /effects/i });
    await effectsButton.click();
    await this.page.waitForTimeout(500);

    // Wait for the effect dropdown to appear
    // The dropdown should now show "Select an effect..." option
    const effectCombobox = dialog.locator('select').last();
    await expect(effectCombobox).toBeVisible({ timeout: 5000 });

    // Get all options and find one containing our effect name
    const optionElements = await effectCombobox.locator('option').all();
    let matchingLabel = "";
    for (const opt of optionElements) {
      const text = await opt.textContent();
      if (text && text.includes(options.effectName)) {
        matchingLabel = text;
        break;
      }
    }

    if (matchingLabel) {
      await effectCombobox.selectOption({ label: matchingLabel });
    } else {
      // Fallback: try selecting by the exact name
      await effectCombobox.selectOption({ label: options.effectName });
    }

    await this.page.waitForTimeout(300);

    // Set intensity if provided
    if (options.intensity !== undefined) {
      // Find spinbuttons after the effect dropdown - intensity is typically the first
      const spinbuttons = dialog.getByRole("spinbutton");
      // Get count to find the right one (after cue number)
      const count = await spinbuttons.count();
      // The last spinbuttons should be intensity and speed
      if (count >= 2) {
        const intensityInput = spinbuttons.nth(count - 2);
        await intensityInput.clear();
        await intensityInput.fill(String(options.intensity));
      }
    }

    // Set speed if provided
    if (options.speed !== undefined) {
      const spinbuttons = dialog.getByRole("spinbutton");
      const count = await spinbuttons.count();
      if (count >= 1) {
        const speedInput = spinbuttons.nth(count - 1);
        await speedInput.clear();
        await speedInput.fill(String(options.speed));
      }
    }

    // Click the "Add Effect" button
    const addEffectBtn = dialog.getByRole("button", { name: /add effect/i });
    await addEffectBtn.click();
    await this.page.waitForTimeout(500);
  }

  /**
   * Check if an effect is attached to the cue being edited.
   */
  async cueHasEffect(effectName: string): Promise<boolean> {
    const dialog = this.page.getByRole("dialog");
    const effectItem = dialog.getByText(effectName);
    return await effectItem.isVisible();
  }

  /**
   * Save the cue edit and close the dialog.
   */
  async saveCueEdit(): Promise<void> {
    const dialog = this.page.getByRole("dialog");
    // Click the "Save" button (exact match to avoid "Save & Edit Look")
    const saveButton = dialog.getByRole("button", { name: "Save", exact: true });
    await saveButton.click();
    await expect(dialog).toBeHidden({ timeout: 10000 });
  }

  /**
   * Cancel the cue edit and close the dialog.
   */
  async cancelCueEdit(): Promise<void> {
    const dialog = this.page.getByRole("dialog");
    const cancelButton = dialog.getByRole("button", { name: /cancel|close/i });
    await cancelButton.click();
    await expect(dialog).toBeHidden({ timeout: 5000 });
  }

  /**
   * Check if an effect indicator is shown for a cue.
   */
  async cueShowsEffectIndicator(cueName: string): Promise<boolean> {
    const escapedForSelector = this.escapeTextForSelector(cueName);
    const cueRow = this.page.locator(`tr:has-text("${escapedForSelector}"), [data-cue-name="${escapedForSelector}"]`).first();
    // Look for effect indicator (icon, badge, or text)
    const effectIndicator = cueRow.locator("[class*='effect'], [title*='effect'], [aria-label*='effect']");
    return await effectIndicator.isVisible();
  }

  /**
   * Go to a specific cue by name.
   * Works in both player view (buttons) and edit view (table rows).
   */
  async goToCue(cueName: string): Promise<void> {
    // Escape special characters for safe use in regex and selector strings
    const escapedForRegex = this.escapeRegex(cueName);
    const escapedForSelector = this.escapeTextForSelector(cueName);

    // In player view, cues are shown as buttons like "0.5: Opening"
    // Use a regex that matches cue number format to avoid matching Undo button
    // Cue buttons have format like "0.5: Opening" or "1: Blackout"
    const cueNumberPattern = new RegExp(`\\d+(\\.\\d+)?:\\s*${escapedForRegex}`);
    const cueButton = this.page.getByRole("button", { name: cueNumberPattern });
    if (await cueButton.first().isVisible()) {
      await cueButton.first().click();
      await this.page.waitForTimeout(500);
      return;
    }

    // In edit view, cues are in table rows
    const cueRow = this.page.locator(`tr:has-text("${escapedForSelector}"), [data-cue-name="${escapedForSelector}"]`).first();
    await cueRow.click();
    await this.page.waitForTimeout(500);
  }
}
