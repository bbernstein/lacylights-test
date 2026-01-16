import { FullConfig } from "@playwright/test";
import * as fs from "fs";
import * as path from "path";

const PROJECT_FILE = path.join(__dirname, ".test-project.json");

/**
 * Global teardown for E2E tests.
 *
 * Cleans up generated files. Server processes are managed by run-tests.sh.
 */
async function globalTeardown(config: FullConfig): Promise<void> {
  console.log("\n🧹 Cleaning up E2E test environment...\n");

  // Clean up project file
  try {
    if (fs.existsSync(PROJECT_FILE)) {
      fs.unlinkSync(PROJECT_FILE);
    }
  } catch {
    // Ignore errors
  }

  console.log("✅ Cleanup complete\n");
}

export default globalTeardown;
