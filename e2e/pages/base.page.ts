import { Page, Locator, expect } from "@playwright/test";

/**
 * Base page object with shared navigation and utilities.
 */
export class BasePage {
  readonly page: Page;

  constructor(page: Page) {
    this.page = page;
  }

  /**
   * Navigate to a route and wait for network to settle.
   */
  async goto(path: string): Promise<void> {
    await this.page.goto(path);
    await this.page.waitForLoadState("networkidle");
  }

  /**
   * Navigate using the bottom navigation (mobile) or sidebar.
   */
  async navigateTo(
    destination: "fixtures" | "looks" | "look-board" | "cue-lists" | "effects" | "settings"
  ): Promise<void> {
    // Try to find the navigation link by text
    const navLinks: Record<string, string> = {
      fixtures: "Fixtures",
      looks: "Looks",
      "look-board": "Look Board",
      "cue-lists": "Cue Lists",
      effects: "Effects",
      settings: "Settings",
    };

    const linkText = navLinks[destination];
    await this.page.getByRole("link", { name: linkText }).first().click();
    await this.page.waitForLoadState("networkidle");
  }

  /**
   * Wait for the page heading to appear.
   */
  async waitForHeading(text: string): Promise<void> {
    await expect(this.page.getByRole("heading", { name: text, level: 2 })).toBeVisible();
  }

  /**
   * Get the page heading text.
   */
  async getHeading(): Promise<string> {
    const heading = this.page.locator("h2").first();
    return await heading.textContent() || "";
  }

  /**
   * Click a button by its text content.
   */
  async clickButton(name: string | RegExp): Promise<void> {
    await this.page.getByRole("button", { name }).click();
  }

  /**
   * Wait for a loading state to complete.
   */
  async waitForLoading(): Promise<void> {
    // Wait for any "Loading..." text to disappear
    await this.page.waitForFunction(() => {
      const body = document.body.textContent || "";
      return !body.includes("Loading...");
    }, { timeout: 10000 });
  }

  /**
   * Fill an input field by its label.
   */
  async fillField(label: string | RegExp, value: string): Promise<void> {
    await this.page.getByLabel(label).fill(value);
  }

  /**
   * Select from a dropdown by label.
   */
  async selectOption(label: string | RegExp, value: string): Promise<void> {
    await this.page.getByLabel(label).selectOption(value);
  }

  /**
   * Confirm a browser dialog (alert/confirm).
   */
  setupDialogHandler(accept: boolean = true): void {
    this.page.on("dialog", async (dialog) => {
      if (accept) {
        await dialog.accept();
      } else {
        await dialog.dismiss();
      }
    });
  }

  /**
   * Wait for an element containing text to be visible.
   */
  async waitForText(text: string | RegExp): Promise<void> {
    await expect(this.page.getByText(text).first()).toBeVisible();
  }

  /**
   * Check if text exists on the page.
   */
  async hasText(text: string | RegExp): Promise<boolean> {
    const elements = await this.page.getByText(text).all();
    return elements.length > 0;
  }
}
