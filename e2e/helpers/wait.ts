/**
 * Wait utilities for E2E tests
 */

/**
 * Wait for a service to be available by polling its URL.
 * @param url - The URL to poll
 * @param timeoutMs - Maximum time to wait in milliseconds
 * @param intervalMs - Polling interval in milliseconds
 */
export async function waitForService(
  url: string,
  timeoutMs: number = 30000,
  intervalMs: number = 500
): Promise<void> {
  const startTime = Date.now();

  while (Date.now() - startTime < timeoutMs) {
    try {
      const response = await fetch(url, {
        method: "GET",
        signal: AbortSignal.timeout(5000),
      });

      if (response.ok) {
        return;
      }
    } catch {
      // Service not ready yet, continue polling
    }

    await sleep(intervalMs);
  }

  throw new Error(`Service at ${url} did not become available within ${timeoutMs}ms`);
}

/**
 * Sleep for a specified duration.
 * @param ms - Duration in milliseconds
 */
export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Wait for a condition to be true.
 * @param condition - Function that returns true when the condition is met
 * @param timeoutMs - Maximum time to wait in milliseconds
 * @param intervalMs - Polling interval in milliseconds
 * @param message - Error message if timeout occurs
 */
export async function waitForCondition(
  condition: () => Promise<boolean> | boolean,
  timeoutMs: number = 10000,
  intervalMs: number = 100,
  message: string = "Condition not met"
): Promise<void> {
  const startTime = Date.now();

  while (Date.now() - startTime < timeoutMs) {
    const result = await condition();
    if (result) {
      return;
    }
    await sleep(intervalMs);
  }

  throw new Error(`${message} within ${timeoutMs}ms`);
}
