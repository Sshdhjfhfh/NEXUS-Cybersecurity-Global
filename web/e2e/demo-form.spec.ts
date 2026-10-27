import { expect, test } from "@playwright/test";

type LeadBody = { organization?: string; email?: string; consent?: boolean };

// The Request Demo form builds a correct request and shows a success state.
// The API is intercepted so the suite is deterministic and does not write test
// rows into Postgres on every run. (Real persistence is verified separately.)
test("submits the demo request and shows success", async ({ page }) => {
  await page.route("**/api/public/leads", (route) =>
    route.fulfill({ status: 201, contentType: "application/json", body: JSON.stringify({ ok: true, id: 999 }) }),
  );

  await page.goto("/request-demo");
  await page.locator('input[name="organization"]').fill("E2E Test Org");
  await page.locator('input[name="contact_name"]').fill("E2E Tester");
  await page.locator('input[name="email"]').fill("e2e@example.test");
  await page.check('input[name="consent"]');

  const [request] = await Promise.all([
    page.waitForRequest("**/api/public/leads"),
    page.getByRole("button", { name: /Enviar solicitud|Send request/ }).click(),
  ]);

  await expect(page.getByText(/Solicitud enviada|Request sent/)).toBeVisible();

  const body = request.postDataJSON() as LeadBody;
  expect(body.organization).toBe("E2E Test Org");
  expect(body.email).toBe("e2e@example.test");
  expect(body.consent).toBe(true);
});

test("blocks submit without required fields", async ({ page }) => {
  await page.goto("/request-demo");
  await page.getByRole("button", { name: /Enviar solicitud|Send request/ }).click();
  // Client-side validation surfaces an error and no success message appears.
  await expect(page.getByText(/Solicitud enviada|Request sent/)).toHaveCount(0);
});
