import { expect, test } from "@playwright/test";

// Header navigation (desktop) and mobile hamburger menu lead to real routes.
const desktopNav: Array<[string, string]> = [
  ["Plataforma", "/platform"],
  ["Soluciones", "/solutions"],
  ["Investigación IA", "/research"],
  ["Servicios", "/services"],
  ["Trust Center", "/trust"],
  ["Nosotros", "/about"],
  ["Contacto", "/contact"],
];

test.describe("desktop navigation", () => {
  test.skip(({ viewport }) => (viewport?.width ?? 0) < 1024, "header nav only visible on desktop");

  for (const [label, path] of desktopNav) {
    test(`header "${label}" -> ${path}`, async ({ page }) => {
      await page.goto("/");
      await page.locator("header").getByRole("link", { name: label, exact: true }).first().click();
      await expect(page).toHaveURL(new RegExp(`${path.replace("/", "\\/")}$`));
      await expect(page.locator("h1").first()).toBeVisible();
    });
  }
});

test.describe("mobile menu", () => {
  test.skip(({ viewport }) => (viewport?.width ?? 0) >= 1024, "hamburger only on small screens");

  test("opens and navigates", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Open menu" }).click();
    const link = page.getByRole("link", { name: "Servicios", exact: true });
    await expect(link).toBeVisible();
    await link.click();
    await expect(page).toHaveURL(/\/services$/);
  });
});
