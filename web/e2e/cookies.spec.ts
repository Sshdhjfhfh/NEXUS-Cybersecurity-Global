import { expect, test } from "@playwright/test";

// Cookie consent: banner appears, accepting hides it and persists, and it can be
// reopened from the footer.
test("accept all hides banner and persists", async ({ page }) => {
  await page.goto("/");
  const banner = page.getByText(/Valoramos tu privacidad|We value your privacy/);
  await expect(banner).toBeVisible();

  await page.getByRole("button", { name: /Aceptar todo|Accept all/ }).click();
  await expect(banner).toHaveCount(0);

  const stored = await page.evaluate(() => localStorage.getItem("nexus_cookie_consent"));
  expect(stored).toContain("analytics");

  // Stays hidden after reload.
  await page.reload();
  await expect(page.getByText(/Valoramos tu privacidad|We value your privacy/)).toHaveCount(0);
});

test("reopen preferences from footer", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: /Aceptar todo|Accept all/ }).click();
  await page.getByRole("button", { name: /Preferencias de cookies|Cookie preferences/ }).click();
  await expect(page.getByText(/Valoramos tu privacidad|We value your privacy/)).toBeVisible();
});
