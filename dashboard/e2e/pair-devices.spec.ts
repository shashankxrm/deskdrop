import { expect, test } from "@playwright/test";

test("pair + device list against compose API", async ({ page }) => {
  const ready = await fetch("http://127.0.0.1:8080/readyz");
  expect(ready.ok, "compose API must be up on :8080").toBeTruthy();

  await page.goto("/");
  await page.getByRole("button", { name: /start a new pairing/i }).click();
  await expect(page.getByRole("heading", { name: /^devices$/i })).toBeVisible({ timeout: 15_000 });
  await expect(page.getByText("web", { exact: true })).toBeVisible();
  await expect(page.getByTestId("invite-code")).toBeVisible();
  await expect(page.getByRole("img", { name: /QR for /i })).toBeVisible();
  await page.getByRole("link", { name: /privacy/i }).click();
  await expect(page.getByRole("heading", { name: /privacy/i })).toBeVisible();
});
