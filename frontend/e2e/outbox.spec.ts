import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_OUTBOX,
  MOCK_DISPATCH_RESULT,
  MOCK_DISPATCH_APPLY_RESULT,
} from "./fixtures/api-mocks"

test.describe("Outbox page", () => {
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
      const url = new URL(route.request().url())
      const statusFilter = url.searchParams.get("status") || ""
      const channelFilter = url.searchParams.get("channel") || ""

      let items = MOCK_OUTBOX.items
      if (statusFilter) items = items.filter((o) => o.status === statusFilter)
      if (channelFilter) items = items.filter((o) => o.target_channel === channelFilter)

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items, total: items.length }),
      })
    })

    await setupAuth(page)
    await page.goto("/outbox")
  })

  test("renders page title", async ({ page }) => {
    await expect(page.locator("h1")).toContainText("Outbox 分发")
  })

  test("renders outbox table with data", async ({ page }) => {
    await expect(page.locator("text=alert_notification").first()).toBeVisible({ timeout: 5000 })

    const rows = page.locator("table tbody tr")
    await expect(rows).toHaveCount(3)
  })

  test("renders filter dropdowns", async ({ page }) => {
    const selects = page.locator("select")
    await expect(selects).toHaveCount(2)
  })

  test("filters by status", async ({ page }) => {
    await page.locator("select").nth(0).selectOption("pending")
    await expect(page.locator("table tbody tr")).toHaveCount(1)
    await expect(page.locator("text=alert_notification")).toBeVisible({ timeout: 5000 })
  })

  test("filters by channel", async ({ page }) => {
    await page.locator("select").nth(1).selectOption("feishu_cli")
    await expect(page.locator("table tbody tr")).toHaveCount(1)
  })

  test("renders dry-run and real dispatch buttons", async ({ page }) => {
    await expect(page.locator("text=Dry-Run 分发")).toBeVisible()
    await expect(page.locator("text=真实分发")).toBeVisible()
  })

  test("dry-run dispatch shows results", async ({ page }) => {
    await page.locator("text=Dry-Run 分发").click()

    await expect(page.locator("text=分发结果")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Dry-Run")).toBeVisible()
    await expect(page.locator("text=2 条")).toBeVisible()
    await expect(page.locator("text=preview").first()).toBeVisible()
  })

  test("real dispatch opens confirmation dialog", async ({ page }) => {
    await page.locator("text=真实分发").click()

    await expect(page.locator("text=确认执行？")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=这将执行真实的分发操作，不可撤销。")).toBeVisible()
  })

  test("real dispatch confirm executes and shows results", async ({ page }) => {
    await page.locator("text=真实分发").click()

    await expect(page.locator("text=确认执行？")).toBeVisible({ timeout: 5000 })

    // Click the confirm button inside the dialog
    await page.locator("text=确认执行").last().click()

    await expect(page.locator("text=分发结果")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Apply")).toBeVisible()
    await expect(page.locator("text=dispatched").first()).toBeVisible()
  })

  test("real dispatch cancel closes dialog", async ({ page }) => {
    await page.locator("text=真实分发").click()
    await expect(page.locator("text=确认执行？")).toBeVisible({ timeout: 5000 })

    await page.locator("text=取消").click()

    // Dialog should be closed
    await expect(page.locator("text=确认执行？")).not.toBeVisible()
  })

  test("shows status color badges", async ({ page }) => {
    const pendingBadge = page.locator("table tbody tr").first().locator("span").filter({ hasText: "pending" })
    await expect(pendingBadge).toBeVisible()
    await expect(pendingBadge).toHaveClass(/bg-yellow-100/)
  })

  test("shows outbox IDs truncated", async ({ page }) => {
    // The table shows outbox_id.slice(0,8)
    await expect(page.locator("text=ob-001-").first()).toBeVisible({ timeout: 5000 })
  })

  test("shows empty state when no items", async ({ page }) => {
    await page.route("**/api/v1/outbox**", async (route) => {
      if (route.request().url().includes("/dispatch")) return
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items: [], total: 0 }),
      })
    })

    await page.goto("/outbox")
    await expect(page.locator("text=暂无 outbox 事件")).toBeVisible({ timeout: 5000 })
  })
})
