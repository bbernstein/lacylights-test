import { Page, Locator, expect } from "@playwright/test";
import { BasePage } from "./base.page";

/**
 * Page object for the Dashboard page.
 * The dashboard displays summary cards for all main sections:
 * Fixtures, Looks, Effects, Look Boards, Cue Lists, and Settings.
 */
export class DashboardPage extends BasePage {
  // Card locators
  readonly fixturesCard: Locator;
  readonly looksCard: Locator;
  readonly effectsCard: Locator;
  readonly lookBoardsCard: Locator;
  readonly cueListsCard: Locator;
  readonly settingsCard: Locator;

  constructor(page: Page) {
    super(page);
    this.fixturesCard = page.getByTestId("fixtures-card");
    this.looksCard = page.getByTestId("looks-card");
    this.effectsCard = page.getByTestId("effects-card");
    this.lookBoardsCard = page.getByTestId("look-boards-card");
    this.cueListsCard = page.getByTestId("cue-lists-card");
    this.settingsCard = page.getByTestId("settings-card");
  }

  /**
   * Navigate to the dashboard page.
   */
  async goto(): Promise<void> {
    await super.goto("/");
    await this.waitForDashboard();
  }

  /**
   * Wait for the dashboard to be fully loaded.
   */
  async waitForDashboard(): Promise<void> {
    await expect(this.page.getByTestId("dashboard-page")).toBeVisible({
      timeout: 15000,
    });
    // Wait for loading to complete
    await this.waitForLoading();
  }

  /**
   * Check if all dashboard cards are visible.
   */
  async hasAllCards(): Promise<boolean> {
    const cards = [
      this.fixturesCard,
      this.looksCard,
      this.effectsCard,
      this.lookBoardsCard,
      this.cueListsCard,
      this.settingsCard,
    ];

    for (const card of cards) {
      if (!(await card.isVisible())) {
        return false;
      }
    }
    return true;
  }

  /**
   * Get the count displayed on a specific card.
   * Cards show a large number indicating the count of items.
   */
  async getCardCount(
    card: "fixtures" | "looks" | "effects" | "look-boards" | "cue-lists"
  ): Promise<number> {
    const cardLocator = this.getCardLocator(card);
    // The count is displayed as a large bold number (text-3xl font-bold)
    const countText = await cardLocator
      .locator(".text-3xl.font-bold")
      .textContent();
    return parseInt(countText || "0", 10);
  }

  /**
   * Get the card locator by name.
   */
  private getCardLocator(
    card: "fixtures" | "looks" | "effects" | "look-boards" | "cue-lists" | "settings"
  ): Locator {
    switch (card) {
      case "fixtures":
        return this.fixturesCard;
      case "looks":
        return this.looksCard;
      case "effects":
        return this.effectsCard;
      case "look-boards":
        return this.lookBoardsCard;
      case "cue-lists":
        return this.cueListsCard;
      case "settings":
        return this.settingsCard;
    }
  }

  /**
   * Check if a card contains specific text.
   */
  async cardContainsText(
    card: "fixtures" | "looks" | "effects" | "look-boards" | "cue-lists" | "settings",
    text: string | RegExp
  ): Promise<boolean> {
    const cardLocator = this.getCardLocator(card);
    const cardText = await cardLocator.textContent();
    if (typeof text === "string") {
      return cardText?.includes(text) || false;
    }
    return text.test(cardText || "");
  }

  /**
   * Click on a card title link to navigate to that section.
   */
  async clickCardLink(
    card: "fixtures" | "looks" | "effects" | "look-boards" | "cue-lists" | "settings"
  ): Promise<void> {
    const titles: Record<string, string> = {
      fixtures: "Fixtures",
      looks: "Looks",
      effects: "Effects",
      "look-boards": "Look Boards",
      "cue-lists": "Cue Lists",
      settings: "Settings",
    };

    const cardLocator = this.getCardLocator(card);
    await cardLocator.getByRole("link", { name: titles[card] }).click();
    await this.page.waitForLoadState("networkidle");
  }

  /**
   * Click the "View all" link on a specific card.
   */
  async clickViewAll(
    card: "fixtures" | "looks" | "effects" | "look-boards" | "cue-lists" | "settings"
  ): Promise<void> {
    const cardLocator = this.getCardLocator(card);
    await cardLocator.getByRole("link", { name: /View all/ }).click();
    await this.page.waitForLoadState("networkidle");
  }

  /**
   * Get the list of item names displayed in a card.
   * Returns the text content of list items.
   */
  async getCardItems(
    card: "fixtures" | "looks" | "effects" | "look-boards" | "cue-lists"
  ): Promise<string[]> {
    const cardLocator = this.getCardLocator(card);
    const listItems = cardLocator.locator("ul li");
    const count = await listItems.count();
    const items: string[] = [];

    for (let i = 0; i < count; i++) {
      const text = await listItems.nth(i).textContent();
      if (text && !text.includes("more...")) {
        items.push(text.trim());
      }
    }
    return items;
  }

  /**
   * Check if an effect is displayed in the Effects card.
   */
  async hasEffectInCard(effectName: string): Promise<boolean> {
    const items = await this.getCardItems("effects");
    return items.some((item) => item.includes(effectName));
  }

  /**
   * Check if a fixture is displayed in the Fixtures card.
   */
  async hasFixtureInCard(fixtureName: string): Promise<boolean> {
    const items = await this.getCardItems("fixtures");
    return items.some((item) => item.includes(fixtureName));
  }

  /**
   * Check if a look is displayed in the Looks card.
   */
  async hasLookInCard(lookName: string): Promise<boolean> {
    const items = await this.getCardItems("looks");
    return items.some((item) => item.includes(lookName));
  }

  /**
   * Get the number of "View all" links on the dashboard.
   */
  async getViewAllLinksCount(): Promise<number> {
    const links = await this.page.getByRole("link", { name: /View all/ }).all();
    return links.length;
  }

  /**
   * Check if the Effects card shows the correct effect type indicator.
   * Effects have colored dots: purple (WAVEFORM), blue (CROSSFADE), yellow (MASTER), gray (STATIC).
   */
  async hasEffectTypeIndicator(
    effectName: string,
    expectedColor: "purple" | "blue" | "yellow" | "gray"
  ): Promise<boolean> {
    const colorClasses: Record<string, string> = {
      purple: "bg-purple-500",
      blue: "bg-blue-500",
      yellow: "bg-yellow-500",
      gray: "bg-gray-400",
    };

    const items = this.effectsCard.locator("ul li");
    const count = await items.count();

    for (let i = 0; i < count; i++) {
      const itemText = await items.nth(i).textContent();
      if (itemText?.includes(effectName)) {
        const indicator = items.nth(i).locator("span.rounded-full");
        const classes = await indicator.getAttribute("class");
        return classes?.includes(colorClasses[expectedColor]) || false;
      }
    }
    return false;
  }
}
