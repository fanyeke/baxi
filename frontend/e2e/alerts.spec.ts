import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_ALERTS,
} from "./fixtures/api-mocks"

test.describe("Alerts page", () => {
  test.beforeEach(async ({ page }) => {
    await mockLayoutApis(page)
    await page.route("**/api/v1/alerts**", async (route) => {
      const url = new URL(route.request().url())
      const severity = url.searchParams.get("severity") || ""
      const status = url.searchParams.get("status") || ""

      let items = MOCK_ALERTS.items
      if (severity) items = items.filter((a) => a.severity === severity)
      if (status) items = items.filter((a) => a.status === status)

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items, total: items.length }),
      })
    })

    await setupAuth(page)
    await page.goto("/alerts")
  })

  test("renders page title", async ({ page }) => {
    await expect(page.locator("h1")).toContainText("告警中心")
  })

  test("renders alerts table with data", async ({ page }) => {
    await expect(page.locator("text=evt-001").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=gmv_drop").first()).toBeVisible()
    await expect(page.locator("text=high").first()).toBeVisible()

    // Should have 3 rows in the table body
    const rows = page.locator("table tbody tr")
    await expect(rows).toHaveCount(3)
  })

  test("renders severity and status filter dropdowns", async ({ page }) => {
    const selects = page.locator("select")
    await expect(selects).toHaveCount(2)

    // First select should have severity options
    const severitySelect = selects.nth(0)
    await expect(severitySelect.locator("option")).toHaveCount(4) // 全部等级 + 3

    // Second select should have status options
    const statusSelect = selects.nth(1)
    await expect(statusSelect.locator("option")).toHaveCount(4) // 全部状态 + 3
  })

  test("filters by severity", async ({ page }) => {
    await page.locator("select").nth(0).selectOption("high")

    // Only high severity alerts should be shown
    await expect(page.locator("table tbody tr")).toHaveCount(1)
    await expect(page.locator("text=evt-001")).toBeVisible({ timeout: 5000 })
  })

  test("filters by status", async ({ page }) => {
    await page.locator("select").nth(1).selectOption("resolved")

    await expect(page.locator("table tbody tr")).toHaveCount(1)
    await expect(page.locator("text=evt-003")).toBeVisible({ timeout: 5000 })
  })

  test("shows empty state when no results match filter", async ({ page }) => {
    // Filter for a combination that yields no results
    await page.route("**/api/v1/alerts**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items: [], total: 0 }),
      })
    })

    await page.locator("select").nth(0).selectOption("low")
    await page.locator("select").nth(1).selectOption("new")

    await expect(page.locator("text=暂无告警")).toBeVisible({ timeout: 5000 })
  })

  test("resets filter shows all alerts again", async ({ page }) => {
    // Filter to high severity
    await page.locator("select").nth(0).selectOption("high")
    await expect(page.locator("table tbody tr")).toHaveCount(1)

    // Reset back to all
    await page.locator("select").nth(0).selectOption("")
    await expect(page.locator("table tbody tr")).toHaveCount(3)
  })

  test("displays severity color badges", async ({ page }) => {
    // High severity should have red styling
    const highBadge = page.locator("table tbody tr").first().locator("span").filter({ hasText: "high" })
    await expect(highBadge).toBeVisible()
    await expect(highBadge).toHaveClass(/bg-red-100/)
  })
})
