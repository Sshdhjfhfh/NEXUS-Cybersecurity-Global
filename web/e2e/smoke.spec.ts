import { expect, test } from "@playwright/test";

// Every public route renders without an error overlay and shows a heading.
const routes = [
  "/",
  "/platform",
  "/services",
  "/research",
  "/about",
  "/contact",
  "/pricing",
  "/trust",
  "/legal",
  "/legal/terms",
  "/legal/privacy",
  "/legal/cookies",
  "/legal/aup",
  "/legal/security",
  "/legal/vdp",
];

for (const route of routes) {
  test(`loads ${route}`, async ({ page }) => {
    const res = await page.goto(route, { waitUntil: "domcontentloaded" });
    expect(res, `no response for ${route}`).not.toBeNull();
    expect(res?.status(), `status for ${route}`).toBeLessThan(400);
    // A visible heading means the page rendered real content.
    await expect(page.locator("h1, h2").first()).toBeVisible();
    // No Next.js runtime error overlay.
    await expect(page.getByText("Application error", { exact: false })).toHaveCount(0);
  });
}
