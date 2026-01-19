import { Page } from "@playwright/test";

/**
 * Set up CI proxy to route requests from port 4000 to 4001.
 *
 * In CI, the frontend is built with NEXT_PUBLIC_GRAPHQL_URL pointing to port 4001,
 * but some code might still reference port 4000. This proxy ensures those requests
 * are handled correctly.
 *
 * The backend runs on 4001 with CORS_ALLOW_ALL=true to handle cross-origin requests.
 *
 * @param page - The Playwright page object
 */
export async function setupCiProxy(page: Page): Promise<void> {
  if (!process.env.CI) {
    return;
  }

  await page.route("**/localhost:4000/**", async (route) => {
    const request = route.request();
    const originalUrl = request.url();
    const proxiedUrl = originalUrl.replace(":4000", ":4001");

    if (request.method() === "OPTIONS") {
      await route.fulfill({
        status: 204,
        headers: {
          "Access-Control-Allow-Origin": "*",
          "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
          "Access-Control-Allow-Headers": "Content-Type, Authorization, Accept",
          "Access-Control-Max-Age": "86400",
        },
      });
      return;
    }

    const response = await route.fetch({ url: proxiedUrl });
    await route.fulfill({
      response,
      headers: {
        ...response.headers(),
        "Access-Control-Allow-Origin": "*",
      },
    });
  });
}
