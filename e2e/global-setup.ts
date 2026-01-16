import { FullConfig } from "@playwright/test";
import * as fs from "fs";
import * as path from "path";
import { waitForService } from "./helpers/wait";

const PROJECT_FILE = path.join(__dirname, ".test-project.json");

/**
 * Global setup for E2E tests.
 *
 * Expects backend and frontend to be already running (started by run-tests.sh or manually).
 * This setup:
 * 1. Waits for backend to be ready
 * 2. Waits for frontend to be ready
 * 3. Creates the test project via GraphQL
 */
async function globalSetup(config: FullConfig): Promise<void> {
  console.log("\n🚀 Setting up E2E test environment...\n");

  // Wait for backend to be ready
  console.log("⏳ Waiting for backend on port 4001...");
  try {
    await waitForService("http://localhost:4001/health", 30000);
    console.log("✅ Backend is ready");
  } catch (error) {
    console.error("❌ Backend is not running on port 4001");
    console.error("   Run: scripts/run-tests.sh e2e");
    console.error("   Or start backend manually: cd ../lacylights-go && PORT=4001 make run");
    throw error;
  }

  // Wait for frontend to be ready
  console.log("⏳ Waiting for frontend on port 3001...");
  try {
    await waitForService("http://localhost:3001", 30000);
    console.log("✅ Frontend is ready");
  } catch (error) {
    console.error("❌ Frontend is not running on port 3001");
    console.error("   Run: scripts/run-tests.sh e2e");
    console.error("   Or start frontend manually: cd ../lacylights-fe && npm run serve:static -- -p 3001");
    throw error;
  }

  // Create initial test project via GraphQL
  console.log("📋 Creating test project...");
  await createTestProject();
  console.log("✅ Test project created");

  console.log("\n✨ E2E test environment ready!\n");
}

async function createTestProject(): Promise<void> {
  // Generate unique project name with timestamp for distinguishing test runs
  const timestamp = new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19);
  const projectName = `E2E Test ${timestamp}`;

  const response = await fetch("http://localhost:4001/graphql", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      query: `
        mutation CreateProject($input: CreateProjectInput!) {
          createProject(input: $input) {
            id
            name
          }
        }
      `,
      variables: {
        input: {
          name: projectName,
          description: "Project for E2E happy path tests",
        },
      },
    }),
  });
  console.log(`   Project name: ${projectName}`);

  const result = await response.json();
  if (result.errors) {
    throw new Error(`Failed to create test project: ${JSON.stringify(result.errors)}`);
  }

  // Store project ID for tests to use
  const projectId = result.data.createProject.id;
  fs.writeFileSync(PROJECT_FILE, JSON.stringify({ projectId }));
}

export default globalSetup;
