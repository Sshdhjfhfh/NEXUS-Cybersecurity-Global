import { expect, test } from "@playwright/test";

// Unknown legal documents render a graceful "not found" page, not a crash.
test("unknown legal doc shows graceful not-found", async ({ page }) => {
  await page.goto("/legal/does-not-exist");
  await expect(page.getByText(/Documento no encontrado|Document not found/)).toBeVisible();
  await expect(page.getByRole("link", { name: /Volver a Legal|Back to Legal/ })).toBeVisible();
});

// No horizontal overflow on the homepage at the current viewport (responsive).
test("no horizontal scroll on home", async ({ page }) => {
  await page.goto("/");
  const overflow = await page.evaluate(() => {
    const el = document.documentElement;
    return el.scrollWidth - el.clientWidth;
  });
  // Allow a tiny sub-pixel tolerance.
  expect(overflow).toBeLessThanOrEqual(2);
});
