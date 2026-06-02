import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_AGENT_LOGS,
} from "./fixtures/api-mocks"

test.describe("Agent Logs page", () => {
  test.beforeEach(async ({ page }) => {
    await mockLayoutApis(page)
    await page.route("**/api/v1/logs/agent**", async (route) => {
      const url = new URL(route.request().url())
      const tool = url.searchParams.get("tool") || ""
      const status = url.searchParams.get("status") || ""

      let items = MOCK_AGENT_LOGS.items
      if (tool) items = items.filter((l) => l.tool_name === tool)
      if (status) items = items.filter((l) => l.status === status)

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items, total: items.length }),
      })
    })

    await setupAuth(page)
    await page.goto("/agent-logs")
  })

  test("renders page title", async ({ page }) => {
    await expect(page.locator("h1")).toContainText("Agent 执行日志")
  })

  test("renders agent logs table with data", async ({ page }) => {
    await expect(page.locator("text=exec-001").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=create_decision_case").first()).toBeVisible()

    const rows = page.locator("table tbody tr")
    await expect(rows).toHaveCount(3)
  })

  test("renders tool and status filter dropdowns", async ({ page }) => {
    const selects = page.locator("select")
    await expect(selects).toHaveCount(2)
  })

  test("tool filter has correct options", async ({ page }) => {
    const toolSelect = selects(page).nth(0)
    await expect(toolSelect.locator("option")).toHaveCount(4) // 全部工具 + 3
  })

  test("status filter has correct options", async ({ page }) => {
    const statusSelect = selects(page).nth(1)
    await expect(statusSelect.locator("option")).toHaveCount(3) // 全部状态 + 2
  })

  test("filters by tool", async ({ page }) => {
    await page.locator("select").nth(0).selectOption("create_decision_case")
    await expect(page.locator("table tbody tr")).toHaveCount(1)
    await expect(page.locator("text=exec-001")).toBeVisible({ timeout: 5000 })
  })

  test("filters by status", async ({ page }) => {
    await page.locator("select").nth(1).selectOption("failed")
    await expect(page.locator("table tbody tr")).toHaveCount(1)
    await expect(page.locator("text=exec-002")).toBeVisible({ timeout: 5000 })
  })

  test("shows success status badge with green color", async ({ page }) => {
    const successBadge = page.locator("table tbody tr").first().locator("span").filter({ hasText: "success" })
    await expect(successBadge).toBeVisible()
    await expect(successBadge).toHaveClass(/bg-green-100/)
  })

  test("shows failed status badge with red color", async ({ page }) => {
    const failedBadge = page.locator("table tbody tr").nth(1).locator("span").filter({ hasText: "failed" })
    await expect(failedBadge).toBeVisible()
    await expect(failedBadge).toHaveClass(/bg-red-100/)
  })

  test("displays duration in ms", async ({ page }) => {
    await expect(page.locator("text=1200ms").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=30000ms")).toBeVisible()
  })

  test("shows session ID or dash", async ({ page }) => {
    await expect(page.locator("text=sess-001").first()).toBeVisible({ timeout: 5000 })
    // exec-003 has null session_id, should show dash
  })

  test("shows empty state when no logs", async ({ page }) => {
    await page.route("**/api/v1/logs/agent**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items: [], total: 0 }),
      })
    })

    await page.goto("/agent-logs")
    await expect(page.locator("text=暂无执行日志")).toBeVisible({ timeout: 5000 })
  })
})

// Helper to avoid TypeScript issues with selector
function selects(page: import("@playwright/test").Page) {
  return page.locator("select")
}
