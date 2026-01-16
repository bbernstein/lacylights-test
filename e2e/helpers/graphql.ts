/**
 * GraphQL client utilities for E2E test setup and teardown
 */

const GRAPHQL_URL = "http://localhost:4000/graphql";

export interface GraphQLResponse<T = unknown> {
  data?: T;
  errors?: Array<{ message: string }>;
}

/**
 * Execute a GraphQL query or mutation.
 */
export async function graphql<T = unknown>(
  query: string,
  variables?: Record<string, unknown>
): Promise<GraphQLResponse<T>> {
  const response = await fetch(GRAPHQL_URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      query,
      variables,
    }),
  });

  return response.json();
}

/**
 * Get the test project ID from the setup file.
 */
export function getTestProjectId(): string {
  const fs = require("fs");
  const path = require("path");
  const projectFile = path.join(__dirname, "../.test-project.json");

  if (!fs.existsSync(projectFile)) {
    throw new Error("Test project file not found. Did global-setup run?");
  }

  const data = JSON.parse(fs.readFileSync(projectFile, "utf-8"));
  return data.projectId;
}

/**
 * Create a fixture via GraphQL.
 */
export async function createFixture(
  projectId: string,
  name: string,
  options: {
    manufacturer?: string;
    model?: string;
    universe?: number;
    startChannel?: number;
    mode?: string;
  } = {}
): Promise<string> {
  const result = await graphql<{ createFixtureInstance: { id: string } }>(
    `
      mutation CreateFixture($input: CreateFixtureInstanceInput!) {
        createFixtureInstance(input: $input) {
          id
          name
        }
      }
    `,
    {
      input: {
        projectId,
        name,
        manufacturer: options.manufacturer || "Generic",
        model: options.model || "Dimmer",
        universe: options.universe || 1,
        startChannel: options.startChannel || 1,
        mode: options.mode,
      },
    }
  );

  if (result.errors) {
    throw new Error(`Failed to create fixture: ${JSON.stringify(result.errors)}`);
  }

  return result.data!.createFixtureInstance.id;
}

/**
 * Create a look via GraphQL.
 */
export async function createLook(
  projectId: string,
  name: string,
  description?: string
): Promise<string> {
  const result = await graphql<{ createLook: { id: string } }>(
    `
      mutation CreateLook($input: CreateLookInput!) {
        createLook(input: $input) {
          id
          name
        }
      }
    `,
    {
      input: {
        projectId,
        name,
        description,
      },
    }
  );

  if (result.errors) {
    throw new Error(`Failed to create look: ${JSON.stringify(result.errors)}`);
  }

  return result.data!.createLook.id;
}

/**
 * Create a look board via GraphQL.
 */
export async function createLookBoard(
  projectId: string,
  name: string,
  description?: string
): Promise<string> {
  const result = await graphql<{ createLookBoard: { id: string } }>(
    `
      mutation CreateLookBoard($input: CreateLookBoardInput!) {
        createLookBoard(input: $input) {
          id
          name
        }
      }
    `,
    {
      input: {
        projectId,
        name,
        description,
      },
    }
  );

  if (result.errors) {
    throw new Error(`Failed to create look board: ${JSON.stringify(result.errors)}`);
  }

  return result.data!.createLookBoard.id;
}

/**
 * Create a cue list via GraphQL.
 */
export async function createCueList(
  projectId: string,
  name: string,
  description?: string
): Promise<string> {
  const result = await graphql<{ createCueList: { id: string } }>(
    `
      mutation CreateCueList($input: CreateCueListInput!) {
        createCueList(input: $input) {
          id
          name
        }
      }
    `,
    {
      input: {
        projectId,
        name,
        description,
      },
    }
  );

  if (result.errors) {
    throw new Error(`Failed to create cue list: ${JSON.stringify(result.errors)}`);
  }

  return result.data!.createCueList.id;
}

/**
 * Delete all test data from the project.
 */
export async function cleanupTestData(projectId: string): Promise<void> {
  await graphql(
    `
      mutation DeleteProject($id: ID!, $confirmDelete: Boolean!) {
        deleteProject(id: $id, confirmDelete: $confirmDelete)
      }
    `,
    {
      id: projectId,
      confirmDelete: true,
    }
  );
}
