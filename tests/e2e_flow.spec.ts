import { test, expect } from "@playwright/test";

const BASE = process.env.E2E_BASE || "http://localhost:27481";

test("login and land on dashboard", async ({ page }) => {
  await page.goto(BASE + "/login");
  await expect(page.getByRole("heading", { name: "流光驿站" })).toBeVisible();
  await page.locator('input[type="password"]').fill("Owner123!");
  await page.getByRole("button", { name: "进入驿站" }).click();
  await expect(page.getByRole("heading", { name: /北极星/ })).toBeVisible({ timeout: 15000 });
});

test("campaign console has send strategy", async ({ page }) => {
  await page.goto(BASE + "/login");
  await page.getByRole("button", { name: "进入驿站" }).click();
  await page.getByRole("link", { name: "活动" }).click();
  await expect(page.getByRole("heading", { name: "活动控制台" })).toBeVisible();
  await expect(page.getByText("立即发送")).toBeVisible();
});
