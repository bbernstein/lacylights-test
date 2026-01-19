import { FullConfig } from "@playwright/test";
import * as fs from "fs";
import * as path from "path";
import { waitForService, waitForGraphQL } from "./helpers/wait";

const PROJECT_FILE = path.join(__dirname, ".test-project.json");

/**
 * Global setup for E2E tests.
 *
 * Expects backend and frontend to be already running (started by run-tests.sh or manually).
 * This setup:
 * 1. Waits for backend to be ready
 * 2. Waits for frontend to be ready
 * 3. Cleans up global state from previous runs
 * 4. Creates the test project via GraphQL
 */
async function globalSetup(config: FullConfig): Promise<void> {
  console.log("\n🚀 Setting up E2E test environment...\n");

  // Wait for backend to be ready (use GraphQL query as health check)
  console.log("⏳ Waiting for backend on port 4001...");
  try {
    await waitForGraphQL("http://localhost:4001/graphql", 10000);
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
    await waitForService("http://localhost:3001", 10000);
    console.log("✅ Frontend is ready");
  } catch (error) {
    console.error("❌ Frontend is not running on port 3001");
    console.error("   Run: scripts/run-tests.sh e2e");
    console.error("   Or start frontend manually: cd ../lacylights-fe && npm run serve:static -- -p 3001");
    throw error;
  }

  // Clean up global state from previous test runs
  console.log("🧹 Cleaning up global state...");
  await cleanupGlobalState();

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

/**
 * Clean up global state from previous test runs.
 * This stops any active cue lists and cleans up old test projects.
 */
async function cleanupGlobalState(): Promise<void> {
  // Stop any active cue list
  try {
    const stopResponse = await fetch("http://localhost:4001/graphql", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: `
          mutation StopCueList {
            stopCueList {
              success
            }
          }
        `,
      }),
    });
    const stopResult = await stopResponse.json();
    if (stopResult.data?.stopCueList?.success) {
      console.log("   Stopped active cue list");
    }
  } catch (error) {
    // Ignore errors - there may not be an active cue list
  }

  // Fade to black to reset lighting state
  try {
    await fetch("http://localhost:4001/graphql", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: `
          mutation FadeToBlack {
            fadeToBlack(input: { fadeOutTime: 0 }) {
              success
            }
          }
        `,
      }),
    });
    console.log("   Reset lighting state");
  } catch (error) {
    // Ignore errors
  }

  // Delete old E2E test projects (keep only last few to avoid clutter)
  try {
    const projectsResponse = await fetch("http://localhost:4001/graphql", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: `
          query ListProjects {
            projects {
              id
              name
            }
          }
        `,
      }),
    });
    const projectsResult = await projectsResponse.json();
    const projects = projectsResult.data?.projects || [];

    // Find old E2E test projects (more than 1 hour old based on timestamp in name)
    const e2eProjects = projects.filter((p: { name: string }) =>
      p.name.startsWith("E2E Test ")
    );

    // Delete all old E2E test projects
    for (const project of e2eProjects) {
      try {
        await fetch("http://localhost:4001/graphql", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            query: `
              mutation DeleteProject($id: ID!) {
                deleteProject(id: $id)
              }
            `,
            variables: { id: project.id },
          }),
        });
        console.log(`   Deleted old test project: ${project.name}`);
      } catch (error) {
        // Ignore individual deletion errors
      }
    }
  } catch (error) {
    // Ignore errors
  }
}

export default globalSetup;
