import { Page, expect } from "@playwright/test";
import { BasePage } from "./base.page";

/**
 * Page object for the Effects page (/effects).
 */
export class EffectsPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async goto(): Promise<void> {
    await super.goto("/effects");
    await this.waitForHeading("Effects");
  }

  /**
   * Create a new effect.
   */
  async createEffect(options: {
    name: string;
    type?: string;
    waveform?: string;
    frequency?: number;
    amplitude?: number;
  }): Promise<void> {
    await this.clickButton(/create|new.*effect/i);
    await expect(this.page.getByRole("dialog")).toBeVisible();

    await this.page.getByLabel(/effect name/i).fill(options.name);

    if (options.type) {
      // Use specific "Effect Type" label to avoid matching "Waveform Type"
      const typeSelect = this.page.getByLabel("Effect Type *");
      await typeSelect.selectOption({ label: options.type });
    }

    if (options.waveform) {
      // Use specific "Waveform Type" label
      const waveformSelect = this.page.getByLabel("Waveform Type");
      await waveformSelect.selectOption({ label: options.waveform });
    }

    // Always set a valid frequency value - the form's default "1" is invalid
    // due to step constraint (valid values are like 0.91, 1.01, etc.)
    const frequencyValue = options.frequency !== undefined ? options.frequency : 1.01;
    const frequencyInput = this.page.getByLabel(/frequency.*hz/i);
    await frequencyInput.clear();
    await frequencyInput.fill(String(frequencyValue));

    if (options.amplitude !== undefined) {
      await this.page.getByLabel(/amplitude/i).fill(String(options.amplitude));
    }

    // Click the "Create Effect" button inside the dialog
    const dialog = this.page.getByRole("dialog");
    const createButton = dialog.getByRole("button", { name: /create effect/i });

    // Click outside the input first to ensure validation runs
    await dialog.getByText("Create a new lighting effect").click();
    await this.page.waitForTimeout(200);

    await createButton.click();

    await expect(dialog).toBeHidden({ timeout: 10000 });
    // Wait briefly for effect to be created
    await this.page.waitForTimeout(500);
  }

  /**
   * Activate an effect by name.
   */
  async activateEffect(name: string): Promise<void> {
    const row = this.page.locator(`tr:has-text("${name}"), div:has-text("${name}")`).first();
    await row.getByRole("button", { name: /activate|start|play/i }).click();
  }

  /**
   * Stop an effect by name.
   */
  async stopEffect(name: string): Promise<void> {
    const row = this.page.locator(`tr:has-text("${name}"), div:has-text("${name}")`).first();
    await row.getByRole("button", { name: /stop|deactivate/i }).click();
  }

  /**
   * Delete an effect by name.
   */
  async deleteEffect(name: string): Promise<void> {
    this.setupDialogHandler(true);

    const row = this.page.locator(`tr:has-text("${name}"), div:has-text("${name}")`).first();
    await row.getByRole("button", { name: /delete/i }).click();
  }

  /**
   * Get the count of effects.
   */
  async getEffectCount(): Promise<number> {
    const tableRows = await this.page.locator("tbody tr").count();
    if (tableRows > 0) {
      return tableRows;
    }
    const cards = await this.page.locator(".space-y-4 > div").count();
    return cards;
  }

  /**
   * Check if an effect with the given name exists.
   */
  async hasEffect(name: string): Promise<boolean> {
    return await this.hasText(name);
  }

  /**
   * Check if an effect is currently active.
   */
  async isEffectActive(name: string): Promise<boolean> {
    const row = this.page.locator(`tr:has-text("${name}"), div:has-text("${name}")`).first();
    // Look for active indicator
    const activeIndicator = row.locator("[class*='active'], [class*='running']");
    return await activeIndicator.isVisible();
  }

  /**
   * Open the effect editor for an effect by name.
   */
  async openEffectEditor(name: string): Promise<void> {
    const row = this.page.locator(`tr:has-text("${name}"), div:has-text("${name}")`).first();
    // The edit button has title="Edit effect" (icon-only button)
    // There may be both desktop and mobile buttons - click the visible one
    const editButtons = row.locator('button[title="Edit effect"]');
    const count = await editButtons.count();
    for (let i = 0; i < count; i++) {
      const btn = editButtons.nth(i);
      if (await btn.isVisible()) {
        await btn.click();
        break;
      }
    }
    // Wait for navigation to effect editor
    await this.page.waitForURL(/\/effects\/[a-z0-9-]+\/edit/);
  }
}

/**
 * Page object for the Effect Editor (/effects/[id]/edit).
 */
export class EffectEditorPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  /**
   * Wait for the editor to load.
   */
  async waitForEditor(): Promise<void> {
    await this.waitForLoading();
    // The effect editor has "Basic Information" section heading
    await expect(this.page.getByRole("heading", { name: /basic information/i })).toBeVisible({ timeout: 10000 });
  }

  /**
   * Enter edit mode to modify the effect.
   * The editor starts in view mode and needs to click "Edit" to show form fields.
   */
  async enterEditMode(): Promise<void> {
    const editButton = this.page.getByRole("button", { name: /^edit$/i });
    await editButton.click();
    // Wait for the form fields to appear - look for the Save button which appears in edit mode
    await expect(this.page.getByRole("button", { name: /^save$/i })).toBeVisible({ timeout: 5000 });
    // Also wait for a textbox to be visible in the Basic Information section
    await expect(this.page.getByRole("textbox").first()).toBeVisible({ timeout: 5000 });
  }

  /**
   * Update the effect name.
   * The form is in the main content area under "Basic Information".
   */
  async setName(name: string): Promise<void> {
    // The first textbox in the main content is the effect name
    const mainContent = this.page.locator("main main");
    const nameInput = mainContent.getByRole("textbox").first();
    await nameInput.clear();
    await nameInput.fill(name);
  }

  /**
   * Update the waveform type.
   */
  async setWaveform(waveform: string): Promise<void> {
    // The waveform select is the first combobox in the Waveform Parameters section
    const waveformSection = this.page.locator('section, div').filter({ hasText: /^Waveform Parameters/ }).first();
    const waveformSelect = waveformSection.getByRole("combobox").first();
    await waveformSelect.selectOption({ label: waveform });
  }

  /**
   * Update the frequency.
   */
  async setFrequency(frequency: number): Promise<void> {
    // Frequency is the first spinbutton in Waveform Parameters section
    const waveformSection = this.page.locator('section, div').filter({ hasText: /^Waveform Parameters/ }).first();
    const frequencyInput = waveformSection.getByRole("spinbutton").first();
    await frequencyInput.clear();
    await frequencyInput.fill(String(frequency));
  }

  /**
   * Update the amplitude.
   */
  async setAmplitude(amplitude: number): Promise<void> {
    // Amplitude is the second spinbutton in Waveform Parameters section
    const waveformSection = this.page.locator('section, div').filter({ hasText: /^Waveform Parameters/ }).first();
    const amplitudeInput = waveformSection.getByRole("spinbutton").nth(1);
    await amplitudeInput.clear();
    await amplitudeInput.fill(String(amplitude));
  }

  /**
   * Update the offset.
   */
  async setOffset(offset: number): Promise<void> {
    // Offset is the third spinbutton in Waveform Parameters section
    const waveformSection = this.page.locator('section, div').filter({ hasText: /^Waveform Parameters/ }).first();
    const offsetInput = waveformSection.getByRole("spinbutton").nth(2);
    await offsetInput.clear();
    await offsetInput.fill(String(offset));
  }

  /**
   * Add a fixture to the effect.
   * The "Add Fixture" button toggles to show available fixtures panel.
   */
  async addFixture(fixtureName: string): Promise<void> {
    // Find the Assigned Fixtures section
    const fixturesSection = this.page.locator('div, section').filter({ hasText: /Assigned Fixtures/ }).first();

    // Click "Add Fixture" button to show available fixtures panel (if not already visible)
    const availableFixturesHeading = this.page.getByRole("heading", { name: /available fixtures/i });
    if (!(await availableFixturesHeading.isVisible())) {
      const addFixtureBtn = fixturesSection.getByRole("button", { name: /add fixture/i });
      await addFixtureBtn.click();
      await expect(availableFixturesHeading).toBeVisible({ timeout: 5000 });
    }

    // Find the fixture row by looking for the text and then the nearest Add button
    // The structure is: div > (div with name + div with model) + button "Add"
    // Use a more specific selector: find the Add button that's a sibling of the fixture name
    const fixtureContainer = this.page.locator(`text="${fixtureName}"`).locator('..').locator('..');
    const addBtn = fixtureContainer.getByRole("button", { name: /^add$/i });
    await addBtn.click();

    // Wait for the fixture to appear in assigned list
    await this.page.waitForTimeout(500);
  }

  /**
   * Expand a fixture in the assigned fixtures list to show its settings.
   */
  async expandFixture(fixtureName: string): Promise<void> {
    // Find the fixture button in the assigned fixtures section
    const fixtureButton = this.page.getByRole("button", { name: new RegExp(fixtureName, "i") });
    await fixtureButton.click();
    await this.page.waitForTimeout(300);
  }

  /**
   * Enable all channels of a given type using Quick Channel Selection.
   * Channel types: "Dim" (dimmer/intensity), "R" (red), "G" (green), "B" (blue)
   */
  async enableChannelType(channelType: string): Promise<void> {
    // Find the Quick Channel Selection section
    const quickSection = this.page.locator('div, section').filter({ hasText: /Quick Channel Selection/ }).first();

    // Find the row for this channel type (e.g., "Dim", "R", "G", "B")
    const channelRow = quickSection.locator('div').filter({ hasText: new RegExp(`^${channelType}$`) }).first();

    // Click the "Enable all" button in this row
    const enableBtn = channelRow.locator('..').getByRole("button", { name: /enable all/i });
    await enableBtn.click();
    await this.page.waitForTimeout(300);
  }

  /**
   * Enable all channels on all fixtures at once.
   */
  async enableAllChannels(): Promise<void> {
    // Find the Quick Channel Selection section
    const quickSection = this.page.locator('div, section').filter({ hasText: /Quick Channel Selection/ }).first();

    // Click the "Enable All" button at the bottom (exact match to distinguish from smaller "Enable all" buttons)
    const enableAllBtn = quickSection.getByRole("button", { name: "Enable All", exact: true });
    await enableAllBtn.click();
    await this.page.waitForTimeout(500);
  }

  /**
   * Enable a channel type on a specific fixture.
   * The fixture must be expanded first.
   */
  async enableChannel(channelType: string): Promise<void> {
    // Map friendly names to UI abbreviations
    const channelMap: Record<string, string> = {
      "Intensity": "Dim",
      "Dimmer": "Dim",
      "Red": "R",
      "Green": "G",
      "Blue": "B",
    };
    const abbrev = channelMap[channelType] || channelType;

    // Use Quick Channel Selection to enable this channel type on all fixtures
    await this.enableChannelType(abbrev);
  }

  /**
   * Set channel values using the Quick Channel Selection bulk editor.
   * This sets values for all fixtures that have the channel type enabled.
   */
  async setChannelValues(options: {
    channelType: string;
    ampScale?: number;  // 0-200%
    freqScale?: number; // 0-500%
    minValue?: number;  // 0-100%
    maxValue?: number;  // 0-100%
  }): Promise<void> {
    // Find the Quick Channel Selection section
    const quickSection = this.page.locator('div, section').filter({ hasText: /Quick Channel Selection/i }).first();

    // Find the channel type row and expand it
    const channelRow = quickSection.locator('div').filter({ hasText: new RegExp(`^${options.channelType}`, "i") }).first();
    const expandBtn = channelRow.locator('button').filter({ has: this.page.locator('svg, img') }).first();
    await expandBtn.click();
    await this.page.waitForTimeout(300);

    // Find the bulk editor panel that appeared
    const bulkEditor = this.page.locator('div').filter({ hasText: /Apply to All/i }).first();

    if (options.minValue !== undefined || options.maxValue !== undefined) {
      // Switch to Min/Max mode if needed
      const minMaxToggle = bulkEditor.getByText(/use min.*max/i);
      if (await minMaxToggle.isVisible()) {
        const checkbox = minMaxToggle.locator('input[type="checkbox"]');
        if (!(await checkbox.isChecked())) {
          await minMaxToggle.click();
          await this.page.waitForTimeout(200);
        }
      }

      if (options.minValue !== undefined) {
        const minInput = bulkEditor.locator('input').filter({ hasText: /min/i }).first();
        await minInput.clear();
        await minInput.fill(String(options.minValue));
      }

      if (options.maxValue !== undefined) {
        const maxInput = bulkEditor.locator('input').filter({ hasText: /max/i }).first();
        await maxInput.clear();
        await maxInput.fill(String(options.maxValue));
      }
    } else {
      // Use Amp Scale mode
      if (options.ampScale !== undefined) {
        const ampInputs = bulkEditor.getByRole("spinbutton");
        const ampInput = ampInputs.first();
        await ampInput.clear();
        await ampInput.fill(String(options.ampScale));
      }
    }

    if (options.freqScale !== undefined) {
      const freqInputs = bulkEditor.getByRole("spinbutton");
      // Freq is typically the second spinbutton
      const freqInput = freqInputs.nth(1);
      await freqInput.clear();
      await freqInput.fill(String(options.freqScale));
    }

    // Click the apply button
    const applyBtn = bulkEditor.getByRole("button", { name: /apply/i });
    await applyBtn.click();
    await this.page.waitForTimeout(500);
  }

  /**
   * Check if a fixture is in the effect.
   */
  async hasFixture(fixtureName: string): Promise<boolean> {
    const fixturesSection = this.page.locator('div, section').filter({ hasText: /Assigned Fixtures/ });
    const fixtureElement = fixturesSection.getByText(new RegExp(fixtureName, "i"));
    return await fixtureElement.isVisible();
  }

  /**
   * Get the count of assigned fixtures.
   */
  async getAssignedFixtureCount(): Promise<number> {
    // Look for the count in the "Assigned Fixtures (N)" heading
    const heading = this.page.getByText(/Assigned Fixtures \((\d+)\)/i);
    const text = await heading.textContent();
    const match = text?.match(/\((\d+)\)/);
    return match ? parseInt(match[1], 10) : 0;
  }

  /**
   * Save the effect changes.
   */
  async save(): Promise<void> {
    await this.clickButton(/save/i);
    // Wait for save confirmation
    await this.page.waitForTimeout(1000);
  }

  /**
   * Activate/play the effect from the editor.
   */
  async activate(): Promise<void> {
    await this.clickButton(/activate|play|start/i);
  }

  /**
   * Stop the effect from the editor.
   */
  async stop(): Promise<void> {
    await this.clickButton(/stop|deactivate/i);
  }

  /**
   * Go back to effects list.
   */
  async goBack(): Promise<void> {
    await this.page.goBack();
    await this.page.waitForURL(/\/effects$/);
    // Wait for the effects list to load
    await this.page.waitForLoadState("networkidle");
    // Wait for the Effects heading to be visible
    await expect(this.page.getByRole("heading", { name: /^effects$/i })).toBeVisible({ timeout: 5000 });
  }
}
