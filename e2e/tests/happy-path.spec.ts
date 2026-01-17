import { test, expect } from "@playwright/test";
import { FixturesPage } from "../pages/fixtures.page";
import { LooksPage, LookEditorPage } from "../pages/looks.page";
import { LookBoardListPage, LookBoardPage } from "../pages/look-board.page";
import { CueListsPage, CueListEditorPage } from "../pages/cue-lists.page";
import { EffectsPage, EffectEditorPage } from "../pages/effects.page";

/**
 * LacyLights E2E Happy Path Tests
 *
 * This test suite covers the complete happy path workflow:
 * 1. Adding fixtures to a project
 * 2. Creating looks with fixture values
 * 3. Creating a look board with buttons
 * 4. Creating a cue list with cues
 * 5. Running the cue list playback
 * 6. Playing looks directly
 * 7. Creating and playing effects
 * 8. Playing look board buttons
 *
 * Tests run in serial mode to maintain state between tests.
 */
test.describe("LacyLights Happy Path", () => {
  test.describe.configure({ mode: "serial" });

  // In CI, set up route interception to handle CORS for cross-origin requests
  // between the frontend (localhost:3001) and backend (localhost:4000).
  // This runs BEFORE each test, so routes are configured before any navigation.
  test.beforeEach(async ({ page }) => {
    if (process.env.CI) {
      // Log all console messages from the page to help debug CORS issues
      page.on("console", (msg) => {
        if (msg.type() === "error") {
          console.log(`Browser console error: ${msg.text()}`);
        }
      });

      // Intercept all requests to the backend and ensure CORS headers are present
      // Use a glob pattern instead of regex to ensure proper matching
      await page.route("**/localhost:4000/**", async (route) => {
        const request = route.request();
        console.log(`Intercepted request: ${request.method()} ${request.url()}`);

        // Handle CORS preflight requests (OPTIONS)
        if (request.method() === "OPTIONS") {
          console.log("Handling OPTIONS preflight request");
          await route.fulfill({
            status: 204,
            headers: {
              "Access-Control-Allow-Origin": "*",
              "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
              "Access-Control-Allow-Headers":
                "Content-Type, Authorization, Accept, X-Requested-With",
              "Access-Control-Max-Age": "86400",
            },
          });
          return;
        }

        // For other requests, forward and add CORS headers to response
        console.log(`Forwarding ${request.method()} request to backend`);
        const response = await route.fetch();
        const headers = {
          ...response.headers(),
          "Access-Control-Allow-Origin": "*",
          "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
          "Access-Control-Allow-Headers":
            "Content-Type, Authorization, Accept, X-Requested-With",
        };
        console.log(`Response status: ${response.status()}`);
        await route.fulfill({
          response,
          headers,
        });
      });

      console.log("Route interception configured for CI");
    }
  });

  // Shared test data
  // Note: Fixture models must match actual Open Fixture Library definitions
  // Use proper casing as it appears in OFL (e.g., "RGB Fader", not "rgb-fader")
  const testData = {
    fixtures: [
      {
        name: "Front Wash 1",
        manufacturer: "Generic",
        model: "RGB Fader",  // 3 channels
        universe: 1,
        startChannel: 1,
      },
      {
        name: "Stage Left Par",
        manufacturer: "Generic",
        model: "Strobe",  // 1 channel
        universe: 1,
        startChannel: 4,
      },
    ],
    looks: [
      { name: "Full Bright", description: "All fixtures at full intensity" },
      { name: "Blackout", description: "All fixtures off" },
      { name: "Warm Wash", description: "Amber/warm tones" },
    ],
    lookBoard: {
      name: "Main Controls",
      description: "Primary control board",
    },
    cueList: {
      name: "Act 1",
      description: "Opening scene cues",
    },
    effect: {
      name: "Pulse",
      type: "Waveform (LFO)" as const,
      waveform: "Sine" as const,
      // Note: Don't set frequency - the HTML5 step validation rejects 1.0
      // Let the form use its default value instead
    },
  };

  test("1. Add fixtures to project", async ({ page }) => {
    const fixturesPage = new FixturesPage(page);
    await fixturesPage.goto();

    // Add first fixture
    await fixturesPage.addFixture(testData.fixtures[0]);
    expect(await fixturesPage.hasFixture(testData.fixtures[0].name)).toBe(true);

    // Add second fixture
    await fixturesPage.addFixture(testData.fixtures[1]);
    expect(await fixturesPage.hasFixture(testData.fixtures[1].name)).toBe(true);

    // Verify both fixtures exist
    const count = await fixturesPage.getFixtureCount();
    expect(count).toBeGreaterThanOrEqual(2);
  });

  test("2. Create looks", async ({ page }) => {
    const looksPage = new LooksPage(page);
    await looksPage.goto();

    // Create "Full Bright" look
    await looksPage.createLook(
      testData.looks[0].name,
      testData.looks[0].description
    );
    expect(await looksPage.hasLook(testData.looks[0].name)).toBe(true);

    // Create "Blackout" look
    await looksPage.createLook(
      testData.looks[1].name,
      testData.looks[1].description
    );
    expect(await looksPage.hasLook(testData.looks[1].name)).toBe(true);

    // Create "Warm Wash" look
    await looksPage.createLook(
      testData.looks[2].name,
      testData.looks[2].description
    );
    expect(await looksPage.hasLook(testData.looks[2].name)).toBe(true);

    // Verify all looks exist
    const count = await looksPage.getLookCount();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test("3. Create look board", async ({ page }) => {
    const listPage = new LookBoardListPage(page);
    await listPage.goto();

    // Create a new look board
    await listPage.createBoard(
      testData.lookBoard.name,
      testData.lookBoard.description
    );

    // Open the board
    await listPage.openBoard(testData.lookBoard.name);

    const boardPage = new LookBoardPage(page);
    await boardPage.waitForCanvas();

    // Add looks to the board
    await boardPage.addLookToBoard(testData.looks[0].name, { x: 100, y: 100 });
    await boardPage.addLookToBoard(testData.looks[1].name, { x: 300, y: 100 });

    // Verify buttons were added
    expect(await boardPage.hasLookButton(testData.looks[0].name)).toBe(true);
    expect(await boardPage.hasLookButton(testData.looks[1].name)).toBe(true);
  });

  test("4. Create cue list", async ({ page }) => {
    const cueListsPage = new CueListsPage(page);
    await cueListsPage.goto();

    // Create a new cue list
    await cueListsPage.createCueList(
      testData.cueList.name,
      testData.cueList.description
    );
    expect(await cueListsPage.hasCueList(testData.cueList.name)).toBe(true);

    // Open the cue list
    await cueListsPage.openCueList(testData.cueList.name);

    const editorPage = new CueListEditorPage(page);
    await editorPage.waitForEditor();

    // Enter edit mode to add cues
    await editorPage.enterEditMode();

    // Add cues
    await editorPage.addCue({
      name: "Opening",
      look: testData.looks[0].name,
      fadeIn: 3.0,
    });

    await editorPage.addCue({
      name: "Transition",
      look: testData.looks[2].name,
      fadeIn: 2.0,
    });

    await editorPage.addCue({
      name: "End Scene",
      look: testData.looks[1].name,
      fadeIn: 2.0,
    });

    // Verify cues were added
    const cueCount = await editorPage.getCueCount();
    expect(cueCount).toBeGreaterThanOrEqual(3);
  });

  test("5. Run cue list", async ({ page }) => {
    const cueListsPage = new CueListsPage(page);
    await cueListsPage.goto();

    // Open the cue list
    await cueListsPage.openCueList(testData.cueList.name);

    const editorPage = new CueListEditorPage(page);
    await editorPage.waitForEditor();

    // Start playback
    await editorPage.startPlayback();

    // Wait for the first cue to become active
    await page.waitForTimeout(1000);

    // Check that we're in playback mode
    const activeCue = await editorPage.getActiveCueName();
    expect(activeCue).not.toBeNull();

    // Advance to next cue
    await editorPage.nextCue();
    await page.waitForTimeout(500);

    // Advance to next cue again
    await editorPage.nextCue();
    await page.waitForTimeout(500);

    // Stop playback
    await editorPage.stopPlayback();
  });

  test("6. Play looks directly", async ({ page }) => {
    const looksPage = new LooksPage(page);
    await looksPage.goto();

    // Activate "Full Bright" look
    await looksPage.activateLook(testData.looks[0].name);

    // Wait a moment for the look to activate
    await page.waitForTimeout(1000);

    // Activate "Blackout" look
    await looksPage.activateLook(testData.looks[1].name);

    // Wait a moment
    await page.waitForTimeout(1000);

    // Activate "Warm Wash" look
    await looksPage.activateLook(testData.looks[2].name);
  });

  test("7. Create and play effects", async ({ page }) => {
    const effectsPage = new EffectsPage(page);
    await effectsPage.goto();

    // Create a waveform effect
    await effectsPage.createEffect({
      name: testData.effect.name,
      type: testData.effect.type,
      waveform: testData.effect.waveform,
      frequency: testData.effect.frequency,
    });

    expect(await effectsPage.hasEffect(testData.effect.name)).toBe(true);

    // Activate the effect
    await effectsPage.activateEffect(testData.effect.name);

    // Let the effect run for a moment
    await page.waitForTimeout(2000);

    // Stop the effect
    await effectsPage.stopEffect(testData.effect.name);
  });

  test("8. Play look board buttons", async ({ page }) => {
    const listPage = new LookBoardListPage(page);
    await listPage.goto();

    // Open the look board
    await listPage.openBoard(testData.lookBoard.name);

    const boardPage = new LookBoardPage(page);
    await boardPage.waitForCanvas();

    // Click "Full Bright" button
    await boardPage.clickLookButton(testData.looks[0].name);
    await page.waitForTimeout(500);

    // Click "Blackout" button
    await boardPage.clickLookButton(testData.looks[1].name);
    await page.waitForTimeout(500);
  });

  test("9. Edit effect, add fixtures, and play it", async ({ page }) => {
    const effectsPage = new EffectsPage(page);
    await effectsPage.goto();

    // Open the effect editor for our existing "Pulse" effect
    await effectsPage.openEffectEditor(testData.effect.name);

    const editorPage = new EffectEditorPage(page);
    await editorPage.waitForEditor();

    // Enter edit mode first - the editor starts in view mode
    await editorPage.enterEditMode();

    // Edit the effect parameters - change to a faster square wave
    await editorPage.setWaveform("Square");
    await editorPage.setFrequency(2.02); // Use valid step value
    await editorPage.setAmplitude(80); // Amplitude as percentage (0-100)

    // Add the first fixture (Front Wash 1 - Generic Dimmer)
    await editorPage.addFixture(testData.fixtures[0].name);

    // Add the second fixture (Stage Left Par - RGB)
    await editorPage.addFixture(testData.fixtures[1].name);

    // Verify fixtures were added
    const fixtureCount = await editorPage.getAssignedFixtureCount();
    expect(fixtureCount).toBe(2);

    // Enable all channels on all fixtures using Quick Channel Selection
    await editorPage.enableAllChannels();

    // Save the changes
    await editorPage.save();

    // Activate the effect from the editor and verify it plays
    await editorPage.activate();

    // Let the modified effect run for a moment
    await page.waitForTimeout(2000);

    // Stop the effect
    await editorPage.stop();

    // Go back to effects list
    await editorPage.goBack();

    // Verify the effect still exists
    expect(await effectsPage.hasEffect(testData.effect.name)).toBe(true);

    // Now test playing the effect from the effects page list
    await effectsPage.activateEffect(testData.effect.name);

    // Let it run briefly
    await page.waitForTimeout(1500);

    // Stop the effect from the list
    await effectsPage.stopEffect(testData.effect.name);
  });

  test("10. Add effect to cue and play cue with effect and look", async ({ page }) => {
    const cueListsPage = new CueListsPage(page);
    await cueListsPage.goto();

    // Open the cue list
    await cueListsPage.openCueList(testData.cueList.name);

    const editorPage = new CueListEditorPage(page);
    await editorPage.waitForEditor();

    // Enter edit mode
    await editorPage.enterEditMode();

    // Edit the "Opening" cue to add our effect
    await editorPage.editCue("Opening");

    // Add the Pulse effect to this cue
    await editorPage.addEffectToCue({
      effectName: testData.effect.name,
      intensity: 80,
      speed: 1.5,
    });

    // Verify the effect was added
    expect(await editorPage.cueHasEffect(testData.effect.name)).toBe(true);

    // Save the cue edit
    await editorPage.saveCueEdit();

    // Exit edit mode by going back to cue lists page
    await editorPage.exitEditMode();

    // Reopen the cue list fresh in playback mode
    await cueListsPage.openCueList(testData.cueList.name);
    await editorPage.waitForEditor();

    // Select the Opening cue first to enable playback
    await editorPage.goToCue("Opening");

    // Start playback - this should play both the look AND the effect
    await editorPage.startPlayback();

    // Let the cue with effect play for a bit (look + effect are both active)
    await page.waitForTimeout(3000);

    // Advance to next cue to verify transition works
    await editorPage.nextCue();
    await page.waitForTimeout(1000);

    // Stop playback
    await editorPage.stopPlayback();
  });
});

/**
 * Cleanup test - runs last to clean up test data.
 */
test.describe("Cleanup", () => {
  test.skip("Delete test data", async ({ page }) => {
    // This test is skipped by default
    // Enable it if you want to clean up after running the happy path

    // Delete fixtures
    const fixturesPage = new FixturesPage(page);
    await fixturesPage.goto();
    await fixturesPage.deleteFixture("Front Wash 1");
    await fixturesPage.deleteFixture("Stage Left Par");

    // Delete looks
    const looksPage = new LooksPage(page);
    await looksPage.goto();
    await looksPage.deleteLook("Full Bright");
    await looksPage.deleteLook("Blackout");
    await looksPage.deleteLook("Warm Wash");

    // Delete cue list
    const cueListsPage = new CueListsPage(page);
    await cueListsPage.goto();
    await cueListsPage.deleteCueList("Act 1");

    // Delete effect
    const effectsPage = new EffectsPage(page);
    await effectsPage.goto();
    await effectsPage.deleteEffect("Pulse");
  });
});
