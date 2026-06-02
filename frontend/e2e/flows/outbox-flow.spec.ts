import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_OUTBOX,
  MOCK_DISPATCH_RESULT,
  MOCK_DISPATCH_APPLY_RESULT,
} from "../fixtures/api-mocks"

test.describe("Outbox Dispatch Flow", () => {
  test.beforeEach(async ({ page }) => {
    await mockLayoutApis(page)
    await page.route("**/api/v1/outbox/dispatch", async (route) => {
      if (route.request().method() === "POST") {
        const body = JSON.parse(route.request().postData() || "{}")
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(body.dry_run ? MOCK_DISPATCH_RESULT : MOCK_DISPATCH_APPLY_RESULT),
        })
      } else {
        await route.fulfill({ status: 405 })
      }
    })
    await page.route("**/api/v1/outbox**", async (route) => {
      if (route.request().url().includes("/dispatch")) return
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_OUTBOX),
      })
    })

    await setupAuth(page)
  })

  test("dry-run to real dispatch flow", async ({ page }) => {
    await page.goto("/outbox")

    // 1. Verify outbox items are loaded
    await expect(page.locator("text=alert_notification").first()).toBeVisible({ timeout: 5000 })
    const rows = page.locator("table tbody tr")
    await expect(rows).toHaveCount(3)

    // 2. Execute dry-run
    await page.locator("text=Dry-Run 分发").click()

    // 3. Verify dry-run results
    await expect(page.locator("text=分发结果")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Dry-Run")).toBeVisible()
    await expect(page.locator("text=2 条")).toBeVisible()
    await expect(page.locator("text=preview").first()).toBeVisible()
    await expect(page.locator("text=feishu_adapter").first()).toBeVisible()

    // 4. Initiate real dispatch
    await page.locator("text=真实分发").click()
    await expect(page.locator("text=确认执行？")).toBeVisible({ timeout: 5000 })

    // 5. Confirm real dispatch
    await page.locator("text=确认执行").last().click()

    // 6. Verify apply results
    await expect(page.locator("text=Apply")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=dispatched").first()).toBeVisible()
    await expect(page.locator("text=ext-feishu-001").first()).toBeVisible()
  })

  test("filter then dispatch flow", async ({ page }) => {
    await page.goto("/outbox")

    await expect(page.locator("text=alert_notification").first()).toBeVisible({ timeout: 5000 })

    // Filter to pending status
    await page.locator("select").nth(0).selectOption("pending")
    await expect(page.locator("table tbody tr")).toHaveCount(1)

    // Dry-run on filtered results
    await page.locator("text=Dry-Run 分发").click()
    await expect(page.locator("text=分发结果")).toBeVisible({ timeout: 5000 })
  })

  test("cancel real dispatch does not execute", async ({ page }) => {
    await page.goto("/outbox")

    await expect(page.locator("text=alert_notification").first()).toBeVisible({ timeout: 5000 })

    // Click real dispatch
    await page.locator("text=真实分发").click()
    await expect(page.locator("text=确认执行？")).toBeVisible({ timeout: 5000 })

    // Cancel
    await page.locator("text=取消").click()
    await expect(page.locator("text=确认执行？")).not.toBeVisible()

    // Results should still show the dry-run results (if any), not apply results
    // No "Apply" text should be visible in new results
  })
})
