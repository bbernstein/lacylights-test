import { test, expect, devices, Page, Locator } from "@playwright/test";
import { graphql, getTestProjectId } from "../helpers/graphql";
import { setupCiProxy } from "../helpers/ci-proxy";

/**
 * LacyLights E2E Tests: menus on touch screens (lacylights-fe#180)
 *
 * Covers the Export ▾ menu (ExportMenuButton) and the Manage Projects row ⋯
 * menu (ContextMenu inside ProjectManagementModal) with Chromium touch
 * emulation:
 * - a phone (Pixel 7) in portrait and landscape: the menu stays inside the
 *   viewport, a tap on an item runs the export, an outside tap only closes
 *   the menu (even on the fixed bottom navigation), and the row menu closes
 *   on a second tap and on a tap on the dimmed sheet backdrop (without
 *   closing the sheet) and never drifts away from its row on a swipe;
 * - a touch tablet (iPad Pro 11 viewport): 44px touch targets even though
 *   the viewport is wider than Tailwind's `sm` breakpoint.
 *
 * Real-device checks (iOS Safari, Android Chrome) stay manual; see
 * lacylights-fe#180.
 */

// Device descriptors carry defaultBrowserType, which can't be overridden per
// file; these tests run in the chromium project with touch emulation.
const { defaultBrowserType: _phoneBrowser, ...pixel7 } = devices["Pixel 7"];
const { defaultBrowserType: _tabletBrowser, ...ipadPro11 } = devices["iPad Pro 11"];

const EXPORT_MENU = '[role="menu"][aria-label="Export options"]';
// The row menu is portaled to <body>, outside the sheet, so it is identified
// by its item rather than by scope.
const ROW_MENU_ITEM = /more export options/i;
const suffix = `${Date.now().toString().slice(-6)}${Math.floor(Math.random() * 100)}`;
// "E2E Test " prefix: global setup deletes leftovers from aborted runs.
const extraProjectName = (n: number) => `E2E Test Mobile Menu ${suffix} ${String(n).padStart(2, "0")}`;
// Enough projects that the Manage Projects list scrolls on a phone.
const EXTRA_PROJECT_COUNT = 12;
// The row whose ⋯ menu tests 4-7 open (1-based, as in its name). A row near
// the top keeps the menu below its button instead of flipped above it.
const TARGET_ROW = 3;
// The row menu is placed 4px below its button; allow a few px of rounding.
const ROW_MENU_GAP_TOLERANCE = 8;

async function viewport(page: Page): Promise<{ w: number; h: number }> {
  return page.evaluate(() => ({ w: window.innerWidth, h: window.innerHeight }));
}

/**
 * Asserts the element's box lies fully inside the viewport (1px tolerance).
 * Retries briefly: the row menu (ContextMenu) first renders at the button's
 * coordinates and moves itself back inside the viewport in an effect after
 * paint, so a measurement taken right after it becomes visible can catch the
 * pre-adjustment position.
 */
async function expectInsideViewport(page: Page, locator: Locator): Promise<void> {
  await expect(async () => {
    const box = await locator.boundingBox();
    const v = await viewport(page);
    expect(box, "element has a layout box").not.toBeNull();
    expect(box!.x).toBeGreaterThanOrEqual(0);
    expect(box!.y).toBeGreaterThanOrEqual(0);
    expect(box!.x + box!.width).toBeLessThanOrEqual(v.w + 1);
    expect(box!.y + box!.height).toBeLessThanOrEqual(v.h + 1);
  }).toPass({ timeout: 2000 });
}

/**
 * Waits until the element's box stops changing between two samples 100ms
 * apart (e.g. a slide-up animation has finished).
 */
async function waitForStableBox(page: Page, locator: Locator): Promise<void> {
  await expect(async () => {
    const a = JSON.stringify(await locator.boundingBox());
    await page.waitForTimeout(100);
    expect(JSON.stringify(await locator.boundingBox())).toBe(a);
  }).toPass({ timeout: 3000 });
}

/** Taps the centre of an element's current box, as a finger would (no auto-scroll / retries). */
async function tapCenter(page: Page, locator: Locator): Promise<void> {
  const box = await locator.boundingBox();
  expect(box).not.toBeNull();
  await page.touchscreen.tap(box!.x + box!.width / 2, box!.y + box!.height / 2);
}

async function selectTestProject(page: Page): Promise<void> {
  const projectId = getTestProjectId();
  await page.addInitScript((id) => {
    try {
      window.localStorage.setItem("currentProjectId", id);
    } catch {
      // Storage may be unavailable; the app then picks a project itself.
    }
  }, projectId);
}

async function gotoFixtures(page: Page): Promise<Locator> {
  await page.goto("/fixtures");
  const trigger = page.locator('button[aria-label="Export"]:visible').first();
  await expect(trigger).toBeVisible({ timeout: 15000 });
  return trigger;
}

test.describe("Menus on a touch phone", () => {
  test.use({ ...pixel7 });

  const extraProjectIds: string[] = [];

  test.beforeAll(async () => {
    // Sequential so extraProjectIds[i - 1] is project i.
    for (let i = 1; i <= EXTRA_PROJECT_COUNT; i++) {
      const res = await graphql<{ createProject: { id: string } }>(
        `mutation CreateProject($input: CreateProjectInput!) { createProject(input: $input) { id } }`,
        { input: { name: extraProjectName(i) } }
      );
      if (res.errors) throw new Error(`createProject failed: ${JSON.stringify(res.errors)}`);
      extraProjectIds.push(res.data!.createProject.id);
    }
  });

  test.afterAll(async () => {
    // Try every delete even if one fails; global setup's "E2E Test " prefix
    // cleanup removes anything left over on the next run.
    const results = await Promise.allSettled(
      extraProjectIds.map((id) => graphql(`mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`, { id }))
    );
    results.forEach((r, i) => {
      if (r.status === "rejected" || r.value.errors) {
        const reason = r.status === "rejected" ? String(r.reason) : JSON.stringify(r.value.errors);
        console.warn(`cleanup: deleteProject ${extraProjectIds[i]} failed: ${reason}`);
      }
    });
  });

  test.beforeEach(async ({ page }) => {
    await setupCiProxy(page);
    await selectTestProject(page);
  });

  test("1. Export menu opens inside the viewport and a tapped item runs the export", async ({ page }) => {
    const trigger = await gotoFixtures(page);
    const triggerBox = await trigger.boundingBox();
    expect(triggerBox!.height).toBeGreaterThanOrEqual(44);

    await trigger.tap();
    const menu = page.locator(EXPORT_MENU);
    await expect(menu).toBeVisible();
    // On phones the Export button sits at the left edge of the page; a
    // right-aligned menu used to start off-screen.
    await expectInsideViewport(page, menu);

    const exportRequest = page.waitForRequest(
      (req) => req.url().includes("/graphql") && (req.postData() ?? "").includes("exportDataTable")
    );
    await menu.getByRole("menuitem", { name: /csv/i }).first().tap();
    await exportRequest;
    await expect(menu).toBeHidden();
    await expect(page).toHaveURL(/\/fixtures\/?$/);
  });

  test("2. A tap outside the Export menu only closes it", async ({ page }) => {
    const trigger = await gotoFixtures(page);
    await trigger.tap();
    const menu = page.locator(EXPORT_MENU);
    await expect(menu).toBeVisible();

    // Tap a bottom-navigation link: the menu's backdrop must swallow it.
    const looksLink = page.getByRole("link", { name: /^looks$/i }).last();
    await tapCenter(page, looksLink);

    await expect(menu).toBeHidden();
    await expect(page).toHaveURL(/\/fixtures\/?$/);
  });

  test("3. Export menu stays inside the viewport in landscape", async ({ page }) => {
    const portrait = page.viewportSize()!;
    await page.setViewportSize({ width: portrait.height, height: portrait.width });
    try {
      const trigger = await gotoFixtures(page);
      await trigger.tap();
      const menu = page.locator(EXPORT_MENU);
      await expect(menu).toBeVisible();
      await expectInsideViewport(page, menu);
      // Taller than the space left: the menu scrolls instead of overflowing,
      // so its last item can be scrolled into view without leaving the screen.
      const scroll = await menu.evaluate((el) => ({
        clipped: el.scrollHeight > el.clientHeight + 1,
        overflowY: getComputedStyle(el).overflowY,
      }));
      if (scroll.clipped) {
        expect(["auto", "scroll"], "a clipped menu must scroll").toContain(scroll.overflowY);
      }
      const lastItem = menu.getByRole("menuitem", { name: /more import\/export/i });
      await lastItem.scrollIntoViewIfNeeded();
      await expectInsideViewport(page, lastItem);
      await expectInsideViewport(page, menu);
    } finally {
      await page.setViewportSize(portrait);
    }
  });

  async function openManageProjects(page: Page): Promise<Locator> {
    await page.goto("/");
    const res = await graphql<{ project: { name: string } }>(
      `query P($id: ID!) { project(id: $id) { name } }`,
      { id: getTestProjectId() }
    );
    const projectName = res.data!.project.name;
    await page.locator("button:visible", { hasText: projectName }).first().tap();
    await page.getByRole("button", { name: /manage projects/i }).tap();
    const sheet = page.getByTestId("project-management-modal");
    await expect(sheet).toBeVisible();
    // Let the slide-up animation finish before measuring positions.
    await waitForStableBox(page, sheet);
    return sheet;
  }

  async function openRowMenu(page: Page): Promise<{ button: Locator; menu: Locator }> {
    const button = page.getByRole("button", { name: `More actions for ${extraProjectName(TARGET_ROW)}`, exact: true });
    await button.scrollIntoViewIfNeeded();
    await button.tap();
    const menu = page.getByRole("menu").filter({ has: page.getByRole("menuitem", { name: ROW_MENU_ITEM }) });
    await expect(menu).toBeVisible();
    return { button, menu };
  }

  test("4. Row ⋯ menu opens next to its button and a second tap closes it", async ({ page }) => {
    await openManageProjects(page);
    const { button, menu } = await openRowMenu(page);
    await expectInsideViewport(page, menu);

    const buttonBox = (await button.boundingBox())!;
    const menuBox = (await menu.boundingBox())!;
    // Vertically: just below the button.
    expect(Math.abs(menuBox.y - (buttonBox.y + buttonBox.height))).toBeLessThanOrEqual(ROW_MENU_GAP_TOLERANCE);
    // Horizontally: the menu overlaps the button's column (it may be shifted
    // left to stay inside the viewport, but not to an unrelated position).
    expect(menuBox.x, "menu starts left of the button's right edge").toBeLessThan(buttonBox.x + buttonBox.width);
    expect(menuBox.x + menuBox.width, "menu ends right of the button's left edge").toBeGreaterThan(buttonBox.x);

    await tapCenter(page, button);
    await expect(menu).toBeHidden();
    await expect(button).toHaveAttribute("aria-expanded", "false");
  });

  test("5. A tap on the dimmed backdrop closes the row menu but not the sheet", async ({ page }) => {
    const sheet = await openManageProjects(page);
    const sheetBox = (await sheet.boundingBox())!;
    // On a Pixel 7 the sheet leaves a dimmed strip above it; if a layout
    // change ever removes it this fails rather than silently skipping.
    expect(sheetBox.y, "dimmed backdrop area above the sheet").toBeGreaterThanOrEqual(20);
    const { menu } = await openRowMenu(page);

    const v = await viewport(page);
    await page.touchscreen.tap(v.w / 2, sheetBox.y / 2);

    await expect(menu).toBeHidden();
    await expect(sheet).toBeVisible();
  });

  test("6. Swiping the project list never leaves the row menu away from its row", async ({ page }) => {
    const sheet = await openManageProjects(page);
    const { button, menu } = await openRowMenu(page);
    // Wait for the menu to settle into its final position before measuring.
    await expectInsideViewport(page, menu);
    const offset = async () => (await menu.boundingBox())!.y - (await button.boundingBox())!.y;
    const before = await offset();

    const list = sheet.locator(".overflow-y-auto").first();
    const listBox = await list.boundingBox();
    // Swipe towards the side with more room. The target row can end up at
    // the very bottom of the list (depending on how many other projects the
    // backend holds); a swipe that cannot scroll would pass vacuously.
    const { scrollTop, maxScroll } = await list.evaluate((el) => ({
      scrollTop: el.scrollTop,
      maxScroll: el.scrollHeight - el.clientHeight,
    }));
    const scrollDown = maxScroll - scrollTop >= scrollTop;
    const room = scrollDown ? maxScroll - scrollTop : scrollTop;
    expect(room, "the project list can scroll in the swipe direction").toBeGreaterThanOrEqual(100);
    // CDP is Chromium-only; this file runs in the chromium project only.
    const cdp = await page.context().newCDPSession(page);
    await cdp.send("Input.synthesizeScrollGesture", {
      x: Math.round(listBox!.x + 30),
      y: Math.round(listBox!.y + listBox!.height * 0.5),
      // Negative yDistance scrolls the content down (finger moves up).
      yDistance: scrollDown ? -200 : 200,
      gestureSourceType: "touch",
      speed: 800,
    });
    await cdp.detach();
    // Let any momentum scroll and the resulting dismissal settle.
    await expect(async () => {
      const a = await list.evaluate((el) => el.scrollTop);
      await page.waitForTimeout(100);
      expect(await list.evaluate((el) => el.scrollTop)).toBe(a);
    }).toPass({ timeout: 3000 });

    // The menu is fixed-positioned: either the swipe is absorbed while it is
    // open (it stays next to its row) or the scroll closes it. It must never
    // stay open while its row scrolls away.
    if (await menu.isVisible()) {
      expect(Math.abs((await offset()) - before)).toBeLessThanOrEqual(2);
    }
    await expect(sheet).toBeVisible();
  });

  test("7. Tapping the row menu item opens Import / Export for that project", async ({ page }) => {
    await openManageProjects(page);
    const { menu } = await openRowMenu(page);
    await menu.getByRole("menuitem", { name: ROW_MENU_ITEM }).tap();

    const targetId = extraProjectIds[TARGET_ROW - 1];
    await expect(page).toHaveURL(new RegExp(`/import-export/?\\?section=export&project=${targetId}`));
  });
});

test.describe("Menus on a touch tablet", () => {
  test.use({ ...ipadPro11 });

  test.beforeEach(async ({ page }) => {
    await setupCiProxy(page);
    await selectTestProject(page);
  });

  test("8. Export button and items keep 44px touch targets above the sm breakpoint", async ({ page }) => {
    const trigger = await gotoFixtures(page);
    expect((await viewport(page)).w).toBeGreaterThan(640);
    expect((await trigger.boundingBox())!.height).toBeGreaterThanOrEqual(44);

    await trigger.tap();
    const menu = page.locator(EXPORT_MENU);
    await expect(menu).toBeVisible();
    await expectInsideViewport(page, menu);
    for (const item of await menu.getByRole("menuitem").all()) {
      expect((await item.boundingBox())!.height).toBeGreaterThanOrEqual(44);
    }
  });
});
