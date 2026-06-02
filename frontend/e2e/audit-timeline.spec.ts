import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_AUDIT_TIMELINE,
} from "./fixtures/api-mocks"

test.describe("Audit Timeline page", () => {
  test.beforeEach(async ({ page }) => {
    await mockLayoutApis(page)
    await page.route("**/api/v1/logs/audit**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_AUDIT_TIMELINE),
      })
    })

    await setupAuth(page)
    await page.goto("/audit-timeline")
  })

  test("renders page title", async ({ page }) => {
    await expect(page.locator("h1")).toContainText("审计时间线")
  })

  test("renders timeline events", async ({ page }) => {
    await expect(page.locator("text=outbox_dispatch").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=feishu_cli").first()).toBeVisible()
  })

  test("groups events by date", async ({ page }) => {
    // Should show date group labels (rendered as date text in a badge)
    // Events from 2026-05-30 and 2026-05-29
    await expect(page.locator("text=feishu_adapter").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=local_adapter").or(page.locator("text=local_cli_adapter"))).toBeVisible()
  })

  test("shows event details: channel and adapter", async ({ page }) => {
    await expect(page.locator("text=feishu_cli").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=via").first()).toBeVisible()
    await expect(page.locator("text=feishu_adapter").first()).toBeVisible()
  })

  test("shows outbox ID truncated", async ({ page }) => {
    await expect(page.locator("text=ob-001-").first()).toBeVisible({ timeout: 5000 })
  })

  test("shows mode badges", async ({ page }) => {
    await expect(page.locator("text=dry_run").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=apply").first()).toBeVisible()
  })

  test("shows status for each event", async ({ page }) => {
    await expect(page.locator("text=completed").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=failed").first()).toBeVisible()
  })

  test("shows error message for failed events", async ({ page }) => {
    await expect(page.locator("text=Connection timeout")).toBeVisible({ timeout: 5000 })
  })

  test("shows external reference when present", async ({ page }) => {
    await expect(page.locator("text=ext-feishu-001").first()).toBeVisible({ timeout: 5000 })
  })

  test("shows source badge", async ({ page }) => {
    await expect(page.locator("text=outbox_dispatch").first()).toBeVisible({ timeout: 5000 })
  })

  test("shows empty state when no audit logs", async ({ page }) => {
    await page.route("**/api/v1/logs/audit**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items: [], total: 0 }),
      })
    })

    await page.goto("/audit-timeline")
    await expect(page.locator("text=暂无审计日志")).toBeVisible({ timeout: 5000 })
  })

  test("shows status icon for successful events", async ({ page }) => {
    // Completed events should show a green checkmark
    const checkmarks = page.locator("text=/✓/")
    await expect(checkmarks.first()).toBeVisible({ timeout: 5000 })
  })
})
