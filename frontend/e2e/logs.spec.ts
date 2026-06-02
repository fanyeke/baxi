import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_ERROR_LOGS,
  MOCK_AUDIT_LOGS,
  MOCK_RECENT_LOGS,
} from "./fixtures/api-mocks"

test.describe("Logs page", () => {
  test.beforeEach(async ({ page }) => {
    await mockLayoutApis(page)
    await page.route("**/api/v1/logs/errors**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_ERROR_LOGS),
      })
    })
    await page.route("**/api/v1/logs/audit**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_AUDIT_LOGS),
      })
    })
    await page.route("**/api/v1/logs/recent**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_RECENT_LOGS),
      })
    })

    await setupAuth(page)
    await page.goto("/logs")
  })

  test("renders page title", async ({ page }) => {
    await expect(page.locator("h1")).toContainText("日志诊断")
  })

  test("renders three tab buttons", async ({ page }) => {
    await expect(page.locator("text=错误日志")).toBeVisible()
    await expect(page.locator("text=审计日志")).toBeVisible()
    await expect(page.locator("text=最近请求")).toBeVisible()
  })

  test("defaults to errors tab", async ({ page }) => {
    // Errors tab should be active by default, showing error log data
    await expect(page.locator("text=DB_CONN_REFUSED").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Connection refused to database")).toBeVisible()
  })

  test("errors tab shows error logs table", async ({ page }) => {
    const rows = page.locator("table tbody tr")
    await expect(rows).toHaveCount(2)
    await expect(page.locator("text=RATE_LIMIT")).toBeVisible()
  })

  test("switches to audit tab", async ({ page }) => {
    await page.locator("text=审计日志").click()

    await expect(page.locator("text=outbox_dispatch").first()).toBeVisible({ timeout: 5000 })
    const rows = page.locator("table tbody tr")
    await expect(rows).toHaveCount(2)
  })

  test("audit tab shows mode and status", async ({ page }) => {
    await page.locator("text=审计日志").click()

    await expect(page.locator("text=dry_run").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=feishu_cli").first()).toBeVisible()
  })

  test("switches to recent tab", async ({ page }) => {
    await page.locator("text=最近请求").click()

    await expect(page.locator("text=Pipeline started").first()).toBeVisible({ timeout: 5000 })
    const rows = page.locator("table tbody tr")
    await expect(rows).toHaveCount(2)
  })

  test("recent tab shows method and path", async ({ page }) => {
    await page.locator("text=最近请求").click()

    await expect(page.locator("text=POST").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=/api/v1/pipeline/run")).toBeVisible()
  })

  test("switching between tabs updates content", async ({ page }) => {
    // Start on errors tab
    await expect(page.locator("text=DB_CONN_REFUSED")).toBeVisible({ timeout: 5000 })

    // Switch to audit
    await page.locator("text=审计日志").click()
    await expect(page.locator("text=outbox_dispatch").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=DB_CONN_REFUSED")).not.toBeVisible()

    // Switch to recent
    await page.locator("text=最近请求").click()
    await expect(page.locator("text=Pipeline started").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=outbox_dispatch")).not.toBeVisible()

    // Switch back to errors
    await page.locator("text=错误日志").click()
    await expect(page.locator("text=DB_CONN_REFUSED")).toBeVisible({ timeout: 5000 })
  })

  test("errors tab shows error codes in styled badges", async ({ page }) => {
    const badge = page.locator("span").filter({ hasText: "DB_CONN_REFUSED" }).first()
    await expect(badge).toBeVisible()
    await expect(badge).toHaveClass(/bg-red-50/)
  })

  test("shows empty state when no logs", async ({ page }) => {
    await page.route("**/api/v1/logs/errors**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items: [], total: 0 }),
      })
    })
    await page.route("**/api/v1/logs/audit**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items: [], total: 0 }),
      })
    })
    await page.route("**/api/v1/logs/recent**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items: [], total: 0 }),
      })
    })

    await page.goto("/logs")
    await expect(page.locator("text=暂无错误日志")).toBeVisible({ timeout: 5000 })
  })
})
