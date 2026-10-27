import { expect, test } from "@playwright/test";

// Language toggle switches ES <-> EN and persists across navigation.
test("toggle switches language and persists", async ({ page }) => {
  await page.goto("/");
  // Default is Spanish.
  await expect(page.getByRole("heading", { name: /el mundo real/i })).toBeVisible();

  // Switch to English.
  await page.getByRole("button", { name: "Change language" }).click();
  await expect(page.getByRole("heading", { name: /the real world/i })).toBeVisible();

  // Persists after navigating to another page.
  await page.goto("/services");
  await expect(page.getByRole("heading", { name: /offense, defense and assurance/i })).toBeVisible();
});
